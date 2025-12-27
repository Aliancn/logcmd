package stats

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	maxHeaderScanLines = 32
	logFooterReadSize  = 16 * 1024
)

var (
	cmdRegex      = regexp.MustCompile(`^命令:\s*(.+)$`)
	exitCodeRegex = regexp.MustCompile(`^退出码:\s*(\d+)$`)
	statusRegex   = regexp.MustCompile(`^执行状态:\s*(\S+)$`)
	durationRegex = regexp.MustCompile(`^执行时长:\s*(.+)$`)
	dateRegex     = regexp.MustCompile(`^# 时间:\s*(.+)$`)
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
			Source:        SourceLogFiles,
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

	if a.stats.TotalCommands > 0 {
		a.stats.AvgDuration = a.stats.TotalDuration / time.Duration(a.stats.TotalCommands)
		if a.stats.MinDuration == 0 {
			a.stats.MinDuration = a.stats.MaxDuration
		}
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

	if err := parseLogHeader(file, metadata); err != nil {
		return err
	}

	if err := parseLogFooter(file, metadata); err != nil {
		return err
	}

	if metadata.Command == "" {
		fmt.Fprintf(os.Stderr, "跳过缺少元数据的日志: %s\n", filePath)
		return nil
	}

	if metadata.Date == "" {
		fmt.Fprintf(os.Stderr, "警告: 日志缺少时间信息，仅统计命令: %s\n", filePath)
	}

	a.updateStats(metadata)
	return nil
}

func parseLogHeader(file *os.File, meta *LogMetadata) error {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}

	scanner := bufio.NewScanner(file)
	lines := 0
	for scanner.Scan() {
		line := scanner.Text()
		lines++
		if matches := dateRegex.FindStringSubmatch(line); matches != nil {
			if t, err := time.Parse("2006-01-02 15:04:05", matches[1]); err == nil {
				meta.Date = t.Format("2006-01-02")
			}
			break
		}
		if lines >= maxHeaderScanLines {
			break
		}
	}

	return scanner.Err()
}

func parseLogFooter(file *os.File, meta *LogMetadata) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}

	size := info.Size()
	if size == 0 {
		return nil
	}

	readSize := logFooterReadSize
	if int64(readSize) > size {
		readSize = int(size)
	}
	start := size - int64(readSize)
	buf := make([]byte, readSize)
	if _, err := file.ReadAt(buf, start); err != nil && err != io.EOF {
		return err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(buf)))
	for scanner.Scan() {
		line := scanner.Text()

		if matches := cmdRegex.FindStringSubmatch(line); matches != nil {
			parts := strings.Fields(matches[1])
			if len(parts) > 0 {
				meta.Command = parts[0]
			}
		}

		if matches := exitCodeRegex.FindStringSubmatch(line); matches != nil {
			fmt.Sscanf(matches[1], "%d", &meta.ExitCode)
		}

		if matches := statusRegex.FindStringSubmatch(line); matches != nil {
			meta.Success = matches[1] == "成功"
		}

		if matches := durationRegex.FindStringSubmatch(line); matches != nil {
			duration, _ := time.ParseDuration(strings.ReplaceAll(matches[1], " ", ""))
			meta.Duration = duration
		}
	}

	return scanner.Err()
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
	if meta.Duration > a.stats.MaxDuration {
		a.stats.MaxDuration = meta.Duration
	}
	if meta.Duration > 0 && (a.stats.MinDuration == 0 || meta.Duration < a.stats.MinDuration) {
		a.stats.MinDuration = meta.Duration
	}
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
