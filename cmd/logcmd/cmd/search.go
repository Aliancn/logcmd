package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/aliancn/logcmd/internal/config"
	"github.com/aliancn/logcmd/internal/registry"
	"github.com/aliancn/logcmd/internal/search"
	"github.com/spf13/cobra"
)

var (
	searchKeyword string
	searchRegex   bool
	searchCase    bool
	searchContext int
	searchStart   string
	searchEnd     string
	searchAll     bool
	searchDir     string
)

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "搜索日志内容",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSearch()
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)

	searchCmd.Flags().StringVar(&searchKeyword, "keyword", "", "搜索关键词（必需）")
	searchCmd.Flags().BoolVar(&searchRegex, "regex", false, "使用正则表达式搜索")
	searchCmd.Flags().BoolVar(&searchCase, "case", false, "区分大小写")
	searchCmd.Flags().IntVar(&searchContext, "context", 0, "显示上下文行数")
	searchCmd.Flags().StringVar(&searchStart, "start", "", "搜索开始日期 (YYYY-MM-DD)")
	searchCmd.Flags().StringVar(&searchEnd, "end", "", "搜索结束日期 (YYYY-MM-DD)")
	searchCmd.Flags().BoolVar(&searchAll, "all", false, "搜索所有项目")
	searchCmd.Flags().StringVar(&searchDir, "dir", "", "日志目录路径")
}

func runSearch() error {
	if searchKeyword == "" {
		return fmt.Errorf("错误: 请使用 --keyword 参数指定搜索关键词")
	}

	if searchAll {
		return runSearchAllProjects()
	}

	searchDirPath := searchDir
	if searchDirPath == "" {
		searchDirPath = config.DefaultConfig().LogDir
	}

	opts, err := buildSearchOptions(searchDirPath)
	if err != nil {
		return err
	}

	searcher, err := search.New(opts)
	if err != nil {
		return fmt.Errorf("创建搜索器失败: %w", err)
	}

	results, err := searcher.Search()
	if err != nil {
		return fmt.Errorf("搜索失败: %w", err)
	}

	search.PrintResults(results)
	return nil
}

func buildSearchOptions(dir string) (*search.SearchOptions, error) {
	opts := &search.SearchOptions{
		LogDir:        dir,
		Keyword:       searchKeyword,
		UseRegex:      searchRegex,
		CaseSensitive: searchCase,
		ShowContext:   searchContext,
	}

	if searchStart != "" {
		t, err := time.Parse("2006-01-02", searchStart)
		if err != nil {
			return nil, fmt.Errorf("错误: 开始日期格式无效: %w", err)
		}
		opts.StartDate = t
	}

	if searchEnd != "" {
		t, err := time.Parse("2006-01-02", searchEnd)
		if err != nil {
			return nil, fmt.Errorf("错误: 结束日期格式无效: %w", err)
		}
		opts.EndDate = t
	}

	return opts, nil
}

func runSearchAllProjects() error {
	reg, err := registry.New()
	if err != nil {
		return fmt.Errorf("初始化项目注册表失败: %w", err)
	}
	defer reg.Close()

	entries, err := reg.List()
	if err != nil {
		return fmt.Errorf("获取项目列表失败: %w", err)
	}

	if len(entries) == 0 {
		return fmt.Errorf("错误: 没有已注册的项目")
	}

	fmt.Printf("正在搜索 %d 个项目...\n\n", len(entries))

	totalResults := 0
	for i, entry := range entries {
		if _, err := os.Stat(entry.Path); os.IsNotExist(err) {
			fmt.Printf("[%d/%d] 跳过（已删除）: %s\n", i+1, len(entries), entry.Path)
			if err := reg.Delete(fmt.Sprintf("%d", entry.ID)); err != nil {
				fmt.Fprintf(os.Stderr, "  警告: 删除无效项目失败: %v\n", err)
			}
			continue
		}

		fmt.Printf("[%d/%d] 搜索: %s\n", i+1, len(entries), entry.Path)

		opts, err := buildSearchOptions(entry.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  警告: 构建搜索参数失败: %v\n", err)
			continue
		}

		searcher, err := search.New(opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  警告: 创建搜索器失败: %v\n", err)
			continue
		}

		results, err := searcher.Search()
		if err != nil {
			fmt.Fprintf(os.Stderr, "  警告: 搜索失败: %v\n", err)
			continue
		}

		if len(results) > 0 {
			fmt.Printf("  找到 %d 条结果\n", len(results))
			search.PrintResults(results)
			totalResults += len(results)
		} else {
			fmt.Println("  未找到结果")
		}
		fmt.Println()

		reg.UpdateLastChecked(fmt.Sprintf("%d", entry.ID))
	}

	fmt.Printf("搜索完成，总共找到 %d 条结果\n", totalResults)
	return nil
}
