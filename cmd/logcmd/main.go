package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/aliancn/logcmd/internal/config"
	"github.com/aliancn/logcmd/internal/logger"
	"github.com/aliancn/logcmd/internal/search"
	"github.com/aliancn/logcmd/internal/stats"
)

const version = "1.0.0"

var (
	// 全局参数
	logDir = flag.String("dir", "", "日志目录路径（默认：自动选择）")

	// 执行命令参数
	runCmd = flag.Bool("run", false, "执行命令并记录日志")

	// 搜索参数
	searchCmd      = flag.Bool("search", false, "搜索日志")
	searchKeyword  = flag.String("keyword", "", "搜索关键词")
	searchRegex    = flag.Bool("regex", false, "使用正则表达式搜索")
	searchCase     = flag.Bool("case", false, "区分大小写")
	searchContext  = flag.Int("context", 0, "显示上下文行数")
	searchStart    = flag.String("start", "", "搜索开始日期 (YYYY-MM-DD)")
	searchEnd      = flag.String("end", "", "搜索结束日期 (YYYY-MM-DD)")

	// 统计参数
	statsCmd = flag.Bool("stats", false, "统计分析日志")

	// 其他
	showVersion = flag.Bool("version", false, "显示版本信息")
	showHelp    = flag.Bool("help", false, "显示帮助信息")
)

func main() {
	flag.Parse()

	// 显示版本
	if *showVersion {
		fmt.Printf("logcmd version %s\n", version)
		return
	}

	// 显示帮助
	if *showHelp {
		printHelp()
		return
	}

	// 执行命令
	if *runCmd {
		runCommand()
		return
	}

	// 搜索日志
	if *searchCmd {
		searchLogs()
		return
	}

	// 统计分析
	if *statsCmd {
		analyzeLogs()
		return
	}

	// 如果没有子命令但有参数，默认执行命令
	args := flag.Args()
	if len(args) > 0 {
		runCommandWithArgs(args)
		return
	}

	// 没有任何参数，显示帮助
	printHelp()
}

func runCommand() {
	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "错误: 请提供要执行的命令")
		os.Exit(1)
	}
	runCommandWithArgs(args)
}

func runCommandWithArgs(args []string) {
	cfg := config.DefaultConfig()

	// 只有当用户明确指定了 -dir 参数时，才覆盖默认值
	if *logDir != "" {
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

	// 确定搜索目录
	searchDir := *logDir
	if searchDir == "" {
		searchDir = config.DefaultConfig().LogDir
	}

	opts := &search.SearchOptions{
		LogDir:        searchDir,
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

func analyzeLogs() {
	// 确定统计目录
	statsDir := *logDir
	if statsDir == "" {
		statsDir = config.DefaultConfig().LogDir
	}

	analyzer := stats.New(statsDir)
	statistics, err := analyzer.Analyze()
	if err != nil {
		fmt.Fprintf(os.Stderr, "统计分析失败: %v\n", err)
		os.Exit(1)
	}

	stats.PrintStats(statistics)
}

func printHelp() {
	fmt.Printf(`logcmd - 高性能命令日志记录工具 v%s

用法:
  logcmd [选项] <command> [args...]     执行命令并记录日志
  logcmd -search -keyword <关键词>      搜索日志
  logcmd -stats                         统计分析

执行命令:
  logcmd ls -la                         # 执行ls命令并记录日志
  logcmd -dir ./mylogs npm test         # 指定日志目录

搜索日志:
  logcmd -search -keyword "error"                    # 搜索包含"error"的日志
  logcmd -search -keyword "panic" -regex             # 使用正则表达式搜索
  logcmd -search -keyword "error" -context 3         # 显示上下文3行
  logcmd -search -keyword "err" -start 2024-01-01    # 搜索指定日期范围

统计分析:
  logcmd -stats                         # 统计所有日志
  logcmd -stats -dir ./mylogs           # 统计指定目录的日志

全局选项:
  -dir string       日志目录路径（默认：自动查找或创建 .logcmd）
                    * 优先在当前目录查找 .logcmd
                    * 向上查找父目录中的 .logcmd
                    * 都没找到则在当前目录创建 .logcmd
  -version          显示版本信息
  -help             显示帮助信息

执行选项:
  -run              明确指定执行模式（可选，默认行为）

搜索选项:
  -search           启用搜索模式
  -keyword string   搜索关键词
  -regex            使用正则表达式
  -case             区分大小写
  -context int      显示上下文行数
  -start string     开始日期 (YYYY-MM-DD)
  -end string       结束日期 (YYYY-MM-DD)

统计选项:
  -stats            启用统计分析模式

特性:
  ✓ 智能日志目录查找（类似 Git，向上查找或创建 .logcmd）
  ✓ 自动按日期组织日志文件 (.logcmd/2024-01-15/log_20240115_143052.log)
  ✓ 实时显示命令输出并同步记录
  ✓ 记录命令执行时间、退出码等元数据
  ✓ 支持正则表达式搜索
  ✓ 详细的统计分析报告
  ✓ 高性能流式处理，支持大输出

示例:
  # 执行并记录npm测试
  logcmd npm test

  # 搜索所有错误日志
  logcmd -search -keyword "error|fail" -regex

  # 查看今天的统计
  logcmd -stats

`, version)
}
