package stats

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Stats 统计数据
type Stats struct {
	TotalCommands   int                // 总命令数
	SuccessCommands int                // 成功命令数
	FailedCommands  int                // 失败命令数
	TotalDuration   time.Duration      // 总执行时长
	CommandCounts   map[string]int     // 各命令执行次数
	ExitCodes       map[int]int        // 退出码分布
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

// LogMetadata 从日志中解析的元数据
type LogMetadata struct {
	Command  string
	ExitCode int
	Success  bool
	Duration time.Duration
	Date     string
}

// Analyzer 统计分析器
type Analyzer struct {
	logDir string
	stats  *Stats
}

// New 创建统计分析器
func New(logDir string) *Analyzer {
	return &Analyzer{
		logDir: logDir,
		stats: &Stats{
			CommandCounts: make(map[string]int),
			ExitCodes:     make(map[int]int),
			DailyStats:    make(map[string]*DayStats),
		},
	}
}

// Analyze 执行统计分析
func (a *Analyzer) Analyze() (*Stats, error) {
	err := filepath.Walk(a.logDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".log") {
			return nil
		}

		// 分析单个日志文件
		if err := a.analyzeFile(path); err != nil {
			fmt.Fprintf(os.Stderr, "分析文件 %s 失败: %v\n", path, err)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("遍历日志目录失败: %w", err)
	}

	return a.stats, nil
}

// analyzeFile 分析单个日志文件
func (a *Analyzer) analyzeFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	metadata := &LogMetadata{}
	scanner := bufio.NewScanner(file)

	// 正则表达式匹配元数据
	cmdRegex := regexp.MustCompile(`^命令:\s*(.+)$`)
	exitCodeRegex := regexp.MustCompile(`^退出码:\s*(\d+)$`)
	statusRegex := regexp.MustCompile(`^执行状态:\s*(\S+)$`)
	durationRegex := regexp.MustCompile(`^执行时长:\s*(.+)$`)
	dateRegex := regexp.MustCompile(`^# 时间:\s*(.+)$`)

	for scanner.Scan() {
		line := scanner.Text()

		// 解析命令
		if matches := cmdRegex.FindStringSubmatch(line); matches != nil {
			parts := strings.Fields(matches[1])
			if len(parts) > 0 {
				metadata.Command = parts[0]
			}
		}

		// 解析退出码
		if matches := exitCodeRegex.FindStringSubmatch(line); matches != nil {
			fmt.Sscanf(matches[1], "%d", &metadata.ExitCode)
		}

		// 解析执行状态
		if matches := statusRegex.FindStringSubmatch(line); matches != nil {
			metadata.Success = matches[1] == "成功"
		}

		// 解析执行时长
		if matches := durationRegex.FindStringSubmatch(line); matches != nil {
			duration, _ := time.ParseDuration(strings.ReplaceAll(matches[1], " ", ""))
			metadata.Duration = duration
		}

		// 解析日期
		if matches := dateRegex.FindStringSubmatch(line); matches != nil {
			if t, err := time.Parse("2006-01-02 15:04:05", matches[1]); err == nil {
				metadata.Date = t.Format("2006-01-02")
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// 更新统计数据
	if metadata.Command != "" {
		a.updateStats(metadata)
	}

	return nil
}

// updateStats 更新统计数据
func (a *Analyzer) updateStats(meta *LogMetadata) {
	a.stats.TotalCommands++

	if meta.Success {
		a.stats.SuccessCommands++
	} else {
		a.stats.FailedCommands++
	}

	a.stats.TotalDuration += meta.Duration
	a.stats.CommandCounts[meta.Command]++
	a.stats.ExitCodes[meta.ExitCode]++

	// 更新每日统计
	if meta.Date != "" {
		dayStats, exists := a.stats.DailyStats[meta.Date]
		if !exists {
			dayStats = &DayStats{Date: meta.Date}
			a.stats.DailyStats[meta.Date] = dayStats
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
	fmt.Println("日志统计分析报告")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()

	// 总体统计
	fmt.Printf("总命令数: %d\n", stats.TotalCommands)
	fmt.Printf("成功: %d (%.1f%%)\n", stats.SuccessCommands,
		float64(stats.SuccessCommands)/float64(stats.TotalCommands)*100)
	fmt.Printf("失败: %d (%.1f%%)\n", stats.FailedCommands,
		float64(stats.FailedCommands)/float64(stats.TotalCommands)*100)
	fmt.Printf("总执行时长: %v\n", stats.TotalDuration)
	if stats.TotalCommands > 0 {
		avgDuration := stats.TotalDuration / time.Duration(stats.TotalCommands)
		fmt.Printf("平均执行时长: %v\n", avgDuration)
	}
	fmt.Println()

	// 命令使用统计（Top 10）
	if len(stats.CommandCounts) > 0 {
		fmt.Println("命令使用频率 (Top 10):")
		fmt.Println(strings.Repeat("-", 40))

		// 简单排序找出top 10
		type cmdCount struct {
			cmd   string
			count int
		}
		var cmdList []cmdCount
		for cmd, count := range stats.CommandCounts {
			cmdList = append(cmdList, cmdCount{cmd, count})
		}

		// 冒泡排序（简单实现）
		for i := 0; i < len(cmdList); i++ {
			for j := i + 1; j < len(cmdList); j++ {
				if cmdList[j].count > cmdList[i].count {
					cmdList[i], cmdList[j] = cmdList[j], cmdList[i]
				}
			}
		}

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
		for code, count := range stats.ExitCodes {
			fmt.Printf("  退出码 %d: %d 次\n", code, count)
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
