package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/aliancn/logcmd/internal/config"
	"github.com/aliancn/logcmd/internal/logger"
	"github.com/aliancn/logcmd/internal/registry"
	"github.com/aliancn/logcmd/internal/search"
	"github.com/aliancn/logcmd/internal/stats"
)

const version = "1.0.0"

var (
	// 全局参数
	globalFlags = flag.NewFlagSet("global", flag.ExitOnError)
	logDir      = globalFlags.String("dir", "", "日志目录路径（默认：自动选择）")
	showVersion = globalFlags.Bool("version", false, "显示版本信息")
	showHelp    = globalFlags.Bool("help", false, "显示帮助信息")

	// 搜索参数
	searchFlags   = flag.NewFlagSet("search", flag.ExitOnError)
	searchKeyword = searchFlags.String("keyword", "", "搜索关键词")
	searchRegex   = searchFlags.Bool("regex", false, "使用正则表达式搜索")
	searchCase    = searchFlags.Bool("case", false, "区分大小写")
	searchContext = searchFlags.Int("context", 0, "显示上下文行数")
	searchStart   = searchFlags.String("start", "", "搜索开始日期 (YYYY-MM-DD)")
	searchEnd     = searchFlags.String("end", "", "搜索结束日期 (YYYY-MM-DD)")
	searchAll     = searchFlags.Bool("all", false, "搜索所有项目")
	searchDir     = searchFlags.String("dir", "", "日志目录路径")

	// 统计参数
	statsFlags = flag.NewFlagSet("stats", flag.ExitOnError)
	statsAll   = statsFlags.Bool("all", false, "统计所有项目")
	statsDir   = statsFlags.String("dir", "", "日志目录路径")
)

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	// 检查第一个参数
	subCommand := os.Args[1]

	switch subCommand {
	case "-version", "--version":
		fmt.Printf("logcmd version %s\n", version)
		return

	case "-help", "--help", "help":
		printHelp()
		return

	case "search":
		searchFlags.Parse(os.Args[2:])
		searchLogs()
		return

	case "stats":
		statsFlags.Parse(os.Args[2:])
		analyzeLogs()
		return

	case "project":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "错误: 请指定project子命令")
			fmt.Fprintln(os.Stderr, "用法: logcmd project <list|clean|delete>")
			os.Exit(1)
		}
		manageProject(os.Args[2], os.Args[3:])
		return

	default:
		// 默认执行命令
		globalFlags.Parse(os.Args[1:])
		runCommandWithArgs(globalFlags.Args())
	}
}
func runCommandWithArgs(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "错误: 请提供要执行的命令")
		os.Exit(1)
	}

	cfg := config.DefaultConfig()

	// 只有当用户明确指定了 -dir 参数时，才覆盖默认值
	if logDir != nil && *logDir != "" {
		cfg.LogDir = *logDir
	}

	log, err := logger.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建日志记录器失败: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	command := args[0]
	cmdArgs := args[1:]

	if err := log.Run(ctx, command, cmdArgs...); err != nil {
		fmt.Fprintf(os.Stderr, "执行失败: %v\n", err)
		os.Exit(1)
	}
}

func searchLogs() {
	if *searchKeyword == "" {
		fmt.Fprintln(os.Stderr, "错误: 请使用 -keyword 参数指定搜索关键词")
		os.Exit(1)
	}

	// 如果使用-all参数，搜索所有项目
	if *searchAll {
		searchAllProjects()
		return
	}

	// 确定搜索目录
	searchDirPath := *searchDir
	if searchDirPath == "" {
		searchDirPath = config.DefaultConfig().LogDir
	}

	opts := &search.SearchOptions{
		LogDir:        searchDirPath,
		Keyword:       *searchKeyword,
		UseRegex:      *searchRegex,
		CaseSensitive: *searchCase,
		ShowContext:   *searchContext,
	}

	// 解析日期范围
	if *searchStart != "" {
		t, err := time.Parse("2006-01-02", *searchStart)
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: 开始日期格式无效: %v\n", err)
			os.Exit(1)
		}
		opts.StartDate = t
	}

	if *searchEnd != "" {
		t, err := time.Parse("2006-01-02", *searchEnd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: 结束日期格式无效: %v\n", err)
			os.Exit(1)
		}
		opts.EndDate = t
	}

	searcher, err := search.New(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建搜索器失败: %v\n", err)
		os.Exit(1)
	}

	results, err := searcher.Search()
	if err != nil {
		fmt.Fprintf(os.Stderr, "搜索失败: %v\n", err)
		os.Exit(1)
	}

	search.PrintResults(results)
}

func searchAllProjects() {
	reg, err := registry.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化项目注册表失败: %v\n", err)
		os.Exit(1)
	}
	defer reg.Close()

	entries, err := reg.List()
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取项目列表失败: %v\n", err)
		os.Exit(1)
	}

	if len(entries) == 0 {
		fmt.Fprintln(os.Stderr, "错误: 没有已注册的项目")
		os.Exit(1)
	}

	fmt.Printf("正在搜索 %d 个项目...\n\n", len(entries))

	// 对每个项目执行搜索，同时清理不存在的项目
	totalResults := 0
	for i, entry := range entries {
		// 检查目录是否存在
		if _, err := os.Stat(entry.Path); os.IsNotExist(err) {
			fmt.Printf("[%d/%d] 跳过（已删除）: %s\n", i+1, len(entries), entry.Path)
			// 自动删除不存在的项目
			if err := reg.Delete(fmt.Sprintf("%d", entry.ID)); err != nil {
				fmt.Fprintf(os.Stderr, "  警告: 删除无效项目失败: %v\n", err)
			}
			continue
		}

		fmt.Printf("[%d/%d] 搜索: %s\n", i+1, len(entries), entry.Path)

		opts := &search.SearchOptions{
			LogDir:        entry.Path,
			Keyword:       *searchKeyword,
			UseRegex:      *searchRegex,
			CaseSensitive: *searchCase,
			ShowContext:   *searchContext,
		}

		// 解析日期范围
		if *searchStart != "" {
			t, err := time.Parse("2006-01-02", *searchStart)
			if err != nil {
				fmt.Fprintf(os.Stderr, "错误: 开始日期格式无效: %v\n", err)
				continue
			}
			opts.StartDate = t
		}

		if *searchEnd != "" {
			t, err := time.Parse("2006-01-02", *searchEnd)
			if err != nil {
				fmt.Fprintf(os.Stderr, "错误: 结束日期格式无效: %v\n", err)
				continue
			}
			opts.EndDate = t
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

		// 更新最后检查时间
		reg.UpdateLastChecked(fmt.Sprintf("%d", entry.ID))
	}

	fmt.Printf("搜索完成，总共找到 %d 条结果\n", totalResults)
}

func analyzeLogs() {
	// 如果使用-all参数，统计所有项目
	if *statsAll {
		analyzeAllProjects()
		return
	}

	// 确定统计目录
	statsDirPath := *statsDir
	if statsDirPath == "" {
		statsDirPath = config.DefaultConfig().LogDir
	}

	analyzer := stats.New(statsDirPath)
	statistics, err := analyzer.Analyze()
	if err != nil {
		fmt.Fprintf(os.Stderr, "统计分析失败: %v\n", err)
		os.Exit(1)
	}

	stats.PrintStats(statistics)
}

func analyzeAllProjects() {
	reg, err := registry.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化项目注册表失败: %v\n", err)
		os.Exit(1)
	}
	defer reg.Close()

	entries, err := reg.List()
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取项目列表失败: %v\n", err)
		os.Exit(1)
	}

	if len(entries) == 0 {
		fmt.Fprintln(os.Stderr, "错误: 没有已注册的项目")
		os.Exit(1)
	}

	fmt.Printf("正在统计 %d 个项目...\n\n", len(entries))

	// 对每个项目执行统计，同时清理不存在的项目
	for i, entry := range entries {
		// 检查目录是否存在
		if _, err := os.Stat(entry.Path); os.IsNotExist(err) {
			fmt.Printf("[%d/%d] 跳过（已删除）: %s\n", i+1, len(entries), entry.Path)
			// 自动删除不存在的项目
			if err := reg.Delete(fmt.Sprintf("%d", entry.ID)); err != nil {
				fmt.Fprintf(os.Stderr, "  警告: 删除无效项目失败: %v\n", err)
			}
			continue
		}

		fmt.Printf("[%d/%d] 统计: %s\n", i+1, len(entries), entry.Path)

		analyzer := stats.New(entry.Path)
		statistics, err := analyzer.Analyze()
		if err != nil {
			fmt.Fprintf(os.Stderr, "  警告: 统计分析失败: %v\n", err)
			continue
		}

		stats.PrintStats(statistics)
		fmt.Println()

		// 更新最后检查时间
		reg.UpdateLastChecked(fmt.Sprintf("%d", entry.ID))
	}

	fmt.Println("统计完成")
}

func manageProject(cmd string, args []string) {
	reg, err := registry.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化项目注册表失败: %v\n", err)
		os.Exit(1)
	}
	defer reg.Close()

	switch cmd {
	case "list":
		// 列出所有项目
		entries, err := reg.List()
		if err != nil {
			fmt.Fprintf(os.Stderr, "列出项目失败: %v\n", err)
			os.Exit(1)
		}

		if len(entries) == 0 {
			fmt.Println("没有已注册的项目")
			return
		}

		fmt.Printf("已注册的项目 (共%d个):\n\n", len(entries))
		fmt.Printf("%-5s %-50s %-20s\n", "ID", "路径", "最后检查时间")
		fmt.Println("--------------------------------------------------------------------------------")
		for _, entry := range entries {
			// 检查目录是否仍然存在
			exists := "✓"
			if _, err := os.Stat(entry.Path); os.IsNotExist(err) {
				exists = "✗"
			}
			fmt.Printf("%-5d %-50s %-20s %s\n",
				entry.ID,
				entry.Path,
				entry.LastChecked.Format("2006-01-02 15:04:05"),
				exists)
		}

	case "clean":
		// 清理不存在的项目
		if err := reg.CheckAndCleanup(); err != nil {
			fmt.Fprintf(os.Stderr, "清理失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("清理完成")

	case "delete":
		// 删除指定项目
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "错误: 请指定要删除的项目ID或路径")
			fmt.Fprintln(os.Stderr, "用法: logcmd project delete <id|path>")
			os.Exit(1)
		}

		target := args[0]
		if err := reg.Delete(target); err != nil {
			fmt.Fprintf(os.Stderr, "删除项目失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("成功删除项目: %s\n", target)

	default:
		fmt.Fprintf(os.Stderr, "错误: 未知的project命令: %s\n", cmd)
		fmt.Fprintln(os.Stderr, "支持的命令: list, clean, delete")
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Printf(`logcmd - 高性能命令日志记录工具 v%s

用法:
  logcmd [选项] <command> [args...]     执行命令并记录日志
  logcmd search [选项]                  搜索日志
  logcmd stats [选项]                   统计分析
  logcmd project <command>              管理项目

执行命令:
  logcmd ls -la                         # 执行ls命令并记录日志
  logcmd -dir ./mylogs npm test         # 指定日志目录

搜索日志:
  logcmd search -keyword "error"                    # 搜索包含"error"的日志
  logcmd search -keyword "panic" -regex             # 使用正则表达式搜索
  logcmd search -keyword "error" -context 3         # 显示上下文3行
  logcmd search -keyword "err" -start 2024-01-01    # 搜索指定日期范围
  logcmd search -keyword "err" -all                 # 搜索所有已注册项目

统计分析:
  logcmd stats                          # 统计所有日志
  logcmd stats -dir ./mylogs            # 统计指定目录的日志
  logcmd stats -all                     # 统计所有已注册项目

项目管理:
  logcmd project list                   # 列出所有已注册的项目
  logcmd project clean                  # 清理不存在的项目
  logcmd project delete 1               # 删除ID为1的项目
  logcmd project delete /path/.logcmd   # 删除指定路径的项目

全局选项:
  -dir string       日志目录路径（默认：自动查找或创建 .logcmd）
                    * 优先在当前目录查找 .logcmd
                    * 向上查找父目录中的 .logcmd
                    * 都没找到则在当前目录创建 .logcmd并自动注册
  -version          显示版本信息
  -help, help       显示帮助信息

搜索选项:
  -keyword string   搜索关键词（必需）
  -regex            使用正则表达式
  -case             区分大小写
  -context int      显示上下文行数
  -start string     开始日期 (YYYY-MM-DD)
  -end string       结束日期 (YYYY-MM-DD)
  -dir string       日志目录路径
  -all              搜索所有已注册项目

统计选项:
  -dir string       日志目录路径
  -all              统计所有已注册项目

项目管理命令:
  list              列出所有已注册的项目
  clean             清理不存在的项目
  delete <id|path>  删除指定的项目（支持ID或路径）

特性:
  ✓ 智能日志目录查找（类似 Git，向上查找或创建 .logcmd）
  ✓ 自动项目注册（创建.logcmd时自动注册到全局数据库）
  ✓ 集中状态管理（使用SQLite管理所有项目）
  ✓ 跨项目搜索和统计（-all 参数）
  ✓ 自动清理无效项目（在搜索和统计时自动清理）
  ✓ 自动按日期组织日志文件 (.logcmd/2024-01-15/log_20240115_143052.log)
  ✓ 实时显示命令输出并同步记录
  ✓ 记录命令执行时间、退出码等元数据
  ✓ 支持正则表达式搜索
  ✓ 详细的统计分析报告
  ✓ 高性能流式处理，支持大输出

术语:
  - Project（项目）: 一个.logcmd目录及其管理的所有日志
  - Run（运行）: 一次命令执行及其产生的日志

示例:
  # 执行并记录npm测试（首次自动注册项目）
  logcmd npm test

  # 搜索所有已注册项目中的错误日志
  logcmd search -keyword "error|fail" -regex -all

  # 查看所有项目的统计
  logcmd stats -all

  # 列出所有项目
  logcmd project list

  # 清理无效项目
  logcmd project clean

`, version)
}
