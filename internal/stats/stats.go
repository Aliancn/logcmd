package stats

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aliancn/logcmd/internal/logparser"
	"github.com/aliancn/logcmd/internal/walker"
)

// SourceType 标识统计数据来源
type SourceType string

const (
	SourceLogFiles SourceType = "logs"
	SourceDatabase SourceType = "database"
)

// Stats 统计数据
type Stats struct {
	ProjectName     string               // 项目名称
	RangeLabel      string               // 统计范围
	Source          SourceType           // 数据来源
	TotalCommands   int                  // 总命令数
	SuccessCommands int                  // 成功命令数
	FailedCommands  int                  // 失败命令数
	TotalDuration   time.Duration        // 总执行时长
	AvgDuration     time.Duration        // 平均执行时长
	MaxDuration     time.Duration        // 最长执行时长
	MinDuration     time.Duration        // 最短执行时长
	CommandCounts   map[string]int       // 各命令执行次数
	ExitCodes       map[int]int          // 退出码分布
	DailyStats      map[string]*DayStats // 每日统计
}

// DayStats 单日统计
type DayStats struct {
	Date     string
	Commands int
	Success  int
	Failed   int
	Duration time.Duration
}

// Analyzer 统计分析器
type Analyzer struct {
	logDir string
	stats  *Stats
	mu     sync.Mutex
}

// New 创建统计分析器
func New(logDir string) *Analyzer {
	return &Analyzer{
		logDir: logDir,
		stats: &Stats{
			Source:        SourceLogFiles,
			CommandCounts: make(map[string]int),
			ExitCodes:     make(map[int]int),
			DailyStats:    make(map[string]*DayStats),
		},
	}
}

// Analyze 执行统计分析
func (a *Analyzer) Analyze(ctx context.Context) (*Stats, error) {
	fileWalker, err := walker.New(walker.Options{
		Root: a.logDir,
		FileFilter: func(path string, info os.FileInfo) bool {
			return strings.HasSuffix(path, ".log")
		},
	})
	if err != nil {
		return nil, fmt.Errorf("创建文件遍历器失败: %w", err)
	}

	err = fileWalker.Walk(ctx, func(ctx context.Context, path string, info os.FileInfo) error {
		if err := a.analyzeFile(ctx, path); err != nil {
			fmt.Fprintf(os.Stderr, "分析文件 %s 失败: %v\n", path, err)
		}
		return nil
	})

	if err != nil {
		if ctx.Err() != nil && err == ctx.Err() {
			return nil, err
		}
		return nil, fmt.Errorf("遍历日志目录失败: %w", err)
	}

	if a.stats.TotalCommands > 0 {
		a.stats.AvgDuration = a.stats.TotalDuration / time.Duration(a.stats.TotalCommands)
		if a.stats.MinDuration == 0 {
			a.stats.MinDuration = a.stats.MaxDuration
		}
	}

	return a.stats, nil
}

// analyzeFile 分析单个日志文件
func (a *Analyzer) analyzeFile(ctx context.Context, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	metadata, err := logparser.ParseFile(ctx, filePath)
	if err != nil {
		return err
	}

	if metadata.CommandName == "" || !metadata.CommandFromFooter {
		fmt.Fprintf(os.Stderr, "跳过缺少元数据的日志: %s\n", filePath)
		return nil
	}

	if metadata.LogDate == "" {
		fmt.Fprintf(os.Stderr, "警告: 日志缺少时间信息，仅统计命令: %s\n", filePath)
	}

	a.updateStats(metadata)
	return nil
}

// updateStats 更新统计数据
func (a *Analyzer) updateStats(meta *logparser.Metadata) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.stats.TotalCommands++

	if meta.Success {
		a.stats.SuccessCommands++
	} else {
		a.stats.FailedCommands++
	}

	a.stats.TotalDuration += meta.Duration
	if meta.Duration > a.stats.MaxDuration {
		a.stats.MaxDuration = meta.Duration
	}
	if meta.Duration > 0 && (a.stats.MinDuration == 0 || meta.Duration < a.stats.MinDuration) {
		a.stats.MinDuration = meta.Duration
	}
	a.stats.CommandCounts[meta.CommandName]++
	a.stats.ExitCodes[meta.ExitCode]++

	// 更新每日统计
	if meta.LogDate != "" {
		dayStats, exists := a.stats.DailyStats[meta.LogDate]
		if !exists {
			dayStats = &DayStats{Date: meta.LogDate}
			a.stats.DailyStats[meta.LogDate] = dayStats
		}
		dayStats.Commands++
		if meta.Success {
			dayStats.Success++
		} else {
			dayStats.Failed++
		}
		dayStats.Duration += meta.Duration
	}
}

// PrintStats 打印统计结果
func PrintStats(stats *Stats) {
	fmt.Println(strings.Repeat("=", 60))
	if stats.ProjectName != "" {
		fmt.Printf("%s 的日志统计分析\n", stats.ProjectName)
	} else {
		fmt.Println("日志统计分析报告")
	}
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()

	if stats.RangeLabel != "" {
		fmt.Printf("统计范围: %s\n", stats.RangeLabel)
	}
	if stats.Source != "" {
		fmt.Printf("数据来源: %s\n", sourceLabel(stats.Source))
	}
	if stats.RangeLabel != "" || stats.Source != "" {
		fmt.Println()
	}

	// 总体统计
	fmt.Printf("总命令数: %d\n", stats.TotalCommands)
	successRate := 0.0
	if stats.TotalCommands > 0 {
		successRate = float64(stats.SuccessCommands) / float64(stats.TotalCommands) * 100
	}
	fmt.Printf("成功: %d (%.1f%%)\n", stats.SuccessCommands, successRate)
	fmt.Printf("失败: %d (%.1f%%)\n", stats.FailedCommands, 100-successRate)
	fmt.Printf("总执行时长: %v\n", stats.TotalDuration)
	if stats.AvgDuration > 0 {
		fmt.Printf("平均执行时长: %v\n", stats.AvgDuration)
	}
	if stats.MaxDuration > 0 {
		fmt.Printf("最长执行时长: %v\n", stats.MaxDuration)
	}
	if stats.MinDuration > 0 && stats.MinDuration != stats.MaxDuration {
		fmt.Printf("最短执行时长: %v\n", stats.MinDuration)
	}
	fmt.Println()

	// 命令使用统计（Top 10）
	if len(stats.CommandCounts) > 0 {
		fmt.Println("命令使用频率 (Top 10):")
		fmt.Println(strings.Repeat("-", 40))

		type cmdCount struct {
			cmd   string
			count int
		}
		var cmdList []cmdCount
		for cmd, count := range stats.CommandCounts {
			cmdList = append(cmdList, cmdCount{cmd, count})
		}

		sort.Slice(cmdList, func(i, j int) bool {
			if cmdList[i].count == cmdList[j].count {
				return cmdList[i].cmd < cmdList[j].cmd
			}
			return cmdList[i].count > cmdList[j].count
		})

		limit := 10
		if len(cmdList) < limit {
			limit = len(cmdList)
		}

		for i := 0; i < limit; i++ {
			fmt.Printf("  %d. %s: %d 次\n", i+1, cmdList[i].cmd, cmdList[i].count)
		}
		fmt.Println()
	}

	// 退出码分布
	if len(stats.ExitCodes) > 0 {
		fmt.Println("退出码分布:")
		fmt.Println(strings.Repeat("-", 40))
		var codes []int
		for code := range stats.ExitCodes {
			codes = append(codes, code)
		}
		sort.Ints(codes)
		for _, code := range codes {
			fmt.Printf("  退出码 %d: %d 次\n", code, stats.ExitCodes[code])
		}
		fmt.Println()
	}

	// 每日统计
	if len(stats.DailyStats) > 0 {
		fmt.Println("每日统计:")
		fmt.Println(strings.Repeat("-", 40))
		for date, dayStats := range stats.DailyStats {
			fmt.Printf("  %s: %d 个命令 (成功: %d, 失败: %d, 总时长: %v)\n",
				date, dayStats.Commands, dayStats.Success, dayStats.Failed, dayStats.Duration)
		}
	}

	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
}

func sourceLabel(source SourceType) string {
	switch source {
	case SourceDatabase:
		return "database"
	case SourceLogFiles:
		return "logs"
	default:
		return string(source)
	}
}
