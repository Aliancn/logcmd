package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"sync"
	"syscall"
	"time"

	"github.com/aliancn/logcmd/internal/config"
	"github.com/aliancn/logcmd/internal/model"
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
		return runSearch(cmd)
	},
}

type searchJob struct {
	index int
	entry *model.Project
}

type searchJobResult struct {
	index   int
	entry   *model.Project
	matches []*search.SearchResult
	err     error
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

func runSearch(cmd *cobra.Command) error {
	if searchKeyword == "" {
		return fmt.Errorf("错误: 请使用 --keyword 参数指定搜索关键词")
	}

	ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	regex, err := compileSearchRegex()
	if err != nil {
		return err
	}

	if searchAll {
		return runSearchAllProjects(ctx, regex)
	}

	searchDirPath := searchDir
	if searchDirPath == "" {
		searchDirPath = config.DefaultConfig().LogDir
	}

	opts, err := buildSearchOptions(searchDirPath, regex)
	if err != nil {
		return err
	}

	searcher, err := search.New(opts)
	if err != nil {
		return fmt.Errorf("创建搜索器失败: %w", err)
	}

	var count int
	err = searcher.Search(ctx, func(result *search.SearchResult) error {
		if count == 0 {
			fmt.Println("匹配结果:")
			fmt.Println()
		}
		printSearchResult(result)
		count++
		return nil
	})
	if err != nil {
		return fmt.Errorf("搜索失败: %w", err)
	}

	if count == 0 {
		fmt.Println("未找到匹配的日志")
	} else {
		fmt.Printf("找到 %d 条匹配记录\n", count)
	}
	return nil
}

func buildSearchOptions(dir string, compiled *regexp.Regexp) (*search.SearchOptions, error) {
	opts := &search.SearchOptions{
		LogDir:        dir,
		Keyword:       searchKeyword,
		UseRegex:      searchRegex,
		CaseSensitive: searchCase,
		ShowContext:   searchContext,
		CompiledRegex: compiled,
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

func runSearchAllProjects(ctx context.Context, compiled *regexp.Regexp) error {
	services, err := newCLIServices()
	if err != nil {
		return err
	}
	defer services.Close()
	reg := services.Registry()

	entries, err := reg.List()
	if err != nil {
		return fmt.Errorf("获取项目列表失败: %w", err)
	}

	if len(entries) == 0 {
		return fmt.Errorf("错误: 没有已注册的项目")
	}

	fmt.Printf("正在搜索 %d 个项目...\n\n", len(entries))

	jobs := make(chan searchJob)
	resultsCh := make(chan searchJobResult, workerCount(len(entries)))
	var wg sync.WaitGroup

	workerTotal := workerCount(len(entries))
	for i := 0; i < workerTotal; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				if ctx.Err() != nil {
					resultsCh <- searchJobResult{index: job.index, entry: job.entry, err: ctx.Err()}
					continue
				}

				opts, err := buildSearchOptions(job.entry.Path, compiled)
				if err != nil {
					resultsCh <- searchJobResult{index: job.index, entry: job.entry, err: err}
					continue
				}

				searcher, err := search.New(opts)
				if err != nil {
					resultsCh <- searchJobResult{index: job.index, entry: job.entry, err: err}
					continue
				}

				var matches []*search.SearchResult
				err = searcher.Search(ctx, func(result *search.SearchResult) error {
					matches = append(matches, result)
					return nil
				})
				resultsCh <- searchJobResult{index: job.index, entry: job.entry, matches: matches, err: err}
			}
		}()
	}

	scheduledEntries := make([]*model.Project, 0, len(entries))

dispatchLoop:
	for _, entry := range entries {
		if _, statErr := os.Stat(entry.Path); os.IsNotExist(statErr) {
			fmt.Printf("[%d/%d] 跳过（已删除）: %s\n", len(scheduledEntries)+1, len(entries), entry.Path)
			if err := reg.Delete(fmt.Sprintf("%d", entry.ID)); err != nil {
				fmt.Fprintf(os.Stderr, "  警告: 删除无效项目失败: %v\n", err)
			}
			continue
		}

		jobIndex := len(scheduledEntries)
		scheduledEntries = append(scheduledEntries, entry)
		job := searchJob{index: jobIndex, entry: entry}

		select {
		case <-ctx.Done():
			scheduledEntries = scheduledEntries[:jobIndex]
			break dispatchLoop
		case jobs <- job:
		}
	}

	close(jobs)
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	results := make([]searchJobResult, len(scheduledEntries))
	for res := range resultsCh {
		results[res.index] = res
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	totalResults := 0
	for i, entry := range scheduledEntries {
		fmt.Printf("[%d/%d] 搜索: %s\n", i+1, len(entries), entry.Path)
		result := results[i]
		if result.err != nil {
			fmt.Fprintf(os.Stderr, "  警告: 搜索失败: %v\n", result.err)
			continue
		}

		if len(result.matches) > 0 {
			fmt.Printf("  找到 %d 条结果\n", len(result.matches))
			for _, match := range result.matches {
				printSearchResult(match)
			}
			totalResults += len(result.matches)
		} else {
			fmt.Println("  未找到结果")
		}
		fmt.Println()

		reg.UpdateLastChecked(fmt.Sprintf("%d", entry.ID))
	}

	fmt.Printf("搜索完成，总共找到 %d 条结果\n", totalResults)
	return nil
}

func compileSearchRegex() (*regexp.Regexp, error) {
	if !searchRegex {
		return nil, nil
	}

	flags := ""
	if !searchCase {
		flags = "(?i)"
	}

	regex, err := regexp.Compile(flags + searchKeyword)
	if err != nil {
		return nil, fmt.Errorf("正则表达式编译失败: %w", err)
	}
	return regex, nil
}

func printSearchResult(result *search.SearchResult) {
	fmt.Printf("文件: %s:%d\n", result.FilePath, result.LineNum)
	if len(result.Context) > 0 {
		fmt.Println("上下文:")
		for _, line := range result.Context {
			fmt.Printf("  %s\n", line)
		}
	} else {
		fmt.Printf("  %s\n", result.Line)
	}
	fmt.Println()
}
