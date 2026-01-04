package stats_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aliancn/logcmd/internal/stats"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	analyzer := stats.New(tmpDir)

	if analyzer == nil {
		t.Fatal("New() 返回了 nil")
	}
}

func TestAnalyzeEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	analyzer := stats.New(tmpDir)

	result, err := analyzer.Analyze(context.Background())
	if err != nil {
		t.Fatalf("Analyze() 失败: %v", err)
	}

	if result == nil {
		t.Fatal("result 不应为 nil")
	}

	if result.TotalCommands != 0 {
		t.Errorf("空目录的 TotalCommands = %d, want 0", result.TotalCommands)
	}
}

func TestAnalyzeWithLogFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建测试日志文件
	logContent := `
################################################################################
# LogCmd - 命令执行日志
# 时间: 2024-01-15 10:00:00
# 命令: echo [test]
################################################################################

test

================================================================================
命令: echo [test]
开始时间: 2024-01-15 10:00:00
结束时间: 2024-01-15 10:00:01
执行时长: 1s
退出码: 0
执行状态: 成功
================================================================================
`

	dateDir := filepath.Join(tmpDir, "2024-01-15")
	os.MkdirAll(dateDir, 0755)

	logPath := filepath.Join(dateDir, "test.log")
	if err := os.WriteFile(logPath, []byte(logContent), 0644); err != nil {
		t.Fatalf("创建测试日志文件失败: %v", err)
	}

	analyzer := stats.New(tmpDir)
	result, err := analyzer.Analyze(context.Background())
	if err != nil {
		t.Fatalf("Analyze() 失败: %v", err)
	}

	if result.TotalCommands != 1 {
		t.Errorf("TotalCommands = %d, want 1", result.TotalCommands)
	}

	if result.SuccessCommands != 1 {
		t.Errorf("SuccessCommands = %d, want 1", result.SuccessCommands)
	}

	if result.FailedCommands != 0 {
		t.Errorf("FailedCommands = %d, want 0", result.FailedCommands)
	}

	if result.CommandCounts["echo"] != 1 {
		t.Errorf("CommandCounts[echo] = %d, want 1", result.CommandCounts["echo"])
	}

	if result.ExitCodes[0] != 1 {
		t.Errorf("ExitCodes[0] = %d, want 1", result.ExitCodes[0])
	}
}

func TestAnalyzeMultipleLogFiles(t *testing.T) {
	tmpDir := t.TempDir()

	logFiles := []struct {
		filename string
		command  string
		exitCode int
		status   string
	}{
		{"test1.log", "echo", 0, "成功"},
		{"test2.log", "ls", 0, "成功"},
		{"test3.log", "false", 1, "失败"},
	}

	dateDir := filepath.Join(tmpDir, "2024-01-15")
	os.MkdirAll(dateDir, 0755)

	for _, lf := range logFiles {
		content := `
################################################################################
# 时间: 2024-01-15 10:00:00
# 命令: ` + lf.command + ` []
################################################################################

================================================================================
命令: ` + lf.command + ` []
开始时间: 2024-01-15 10:00:00
结束时间: 2024-01-15 10:00:01
执行时长: 1s
退出码: ` + string(rune(lf.exitCode+'0')) + `
执行状态: ` + lf.status + `
================================================================================
`
		logPath := filepath.Join(dateDir, lf.filename)
		if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
			t.Fatalf("创建测试日志文件失败: %v", err)
		}
	}

	analyzer := stats.New(tmpDir)
	result, err := analyzer.Analyze(context.Background())
	if err != nil {
		t.Fatalf("Analyze() 失败: %v", err)
	}

	if result.TotalCommands != 3 {
		t.Errorf("TotalCommands = %d, want 3", result.TotalCommands)
	}

	if result.SuccessCommands != 2 {
		t.Errorf("SuccessCommands = %d, want 2", result.SuccessCommands)
	}

	if result.FailedCommands != 1 {
		t.Errorf("FailedCommands = %d, want 1", result.FailedCommands)
	}
}

func TestAnalyzeWithDailyStats(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建不同日期的日志
	dates := []string{"2024-01-15", "2024-01-16"}

	for _, date := range dates {
		dateDir := filepath.Join(tmpDir, date)
		os.MkdirAll(dateDir, 0755)

		content := `
################################################################################
# 时间: ` + date + ` 10:00:00
# 命令: echo []
################################################################################

================================================================================
命令: echo []
执行时长: 1s
退出码: 0
执行状态: 成功
================================================================================
`
		logPath := filepath.Join(dateDir, "test.log")
		if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
			t.Fatalf("创建测试日志文件失败: %v", err)
		}
	}

	analyzer := stats.New(tmpDir)
	result, err := analyzer.Analyze(context.Background())
	if err != nil {
		t.Fatalf("Analyze() 失败: %v", err)
	}

	if len(result.DailyStats) != 2 {
		t.Errorf("DailyStats 长度 = %d, want 2", len(result.DailyStats))
	}

	for _, date := range dates {
		dayStats, exists := result.DailyStats[date]
		if !exists {
			t.Errorf("DailyStats 应该包含日期 %s", date)
			continue
		}

		if dayStats.Commands != 1 {
			t.Errorf("日期 %s 的 Commands = %d, want 1", date, dayStats.Commands)
		}
	}
}

func TestAnalyzeLogWithoutDate(t *testing.T) {
	tmpDir := t.TempDir()
	content := `
################################################################################
# LogCmd - 命令执行日志
# 命令: echo []
################################################################################

output

================================================================================
命令: echo []
开始时间: 2024-01-15 10:00:00
结束时间: 2024-01-15 10:00:01
执行时长: 1s
退出码: 0
执行状态: 成功
================================================================================
`
	logPath := filepath.Join(tmpDir, "test.log")
	if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
		t.Fatalf("创建日志失败: %v", err)
	}

	analyzer := stats.New(tmpDir)
	result, err := analyzer.Analyze(context.Background())
	if err != nil {
		t.Fatalf("Analyze() 失败: %v", err)
	}

	if result.TotalCommands != 1 {
		t.Fatalf("TotalCommands = %d, want 1", result.TotalCommands)
	}

	if len(result.DailyStats) != 0 {
		t.Fatalf("DailyStats 长度 = %d, want 0", len(result.DailyStats))
	}
}

func TestAnalyzeLogMissingMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	content := `
################################################################################
# LogCmd - 命令执行日志
# 时间: 2024-01-15 10:00:00
# 命令: echo []
################################################################################

output
`
	logPath := filepath.Join(tmpDir, "test.log")
	if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
		t.Fatalf("创建日志失败: %v", err)
	}

	analyzer := stats.New(tmpDir)
	result, err := analyzer.Analyze(context.Background())
	if err != nil {
		t.Fatalf("Analyze() 失败: %v", err)
	}

	if result.TotalCommands != 0 {
		t.Fatalf("总命令数 = %d, want 0", result.TotalCommands)
	}
}

func TestAnalyzeNonExistentDirectory(t *testing.T) {
	analyzer := stats.New("/nonexistent/directory")
	_, err := analyzer.Analyze(context.Background())
	if err == nil {
		t.Error("Analyze() 应该对不存在的目录返回错误")
	}
}

func TestStatsInitialization(t *testing.T) {
	tmpDir := t.TempDir()
	analyzer := stats.New(tmpDir)

	result, err := analyzer.Analyze(context.Background())
	if err != nil {
		t.Fatalf("Analyze() 失败: %v", err)
	}

	// 验证统计数据结构已初始化
	if result.CommandCounts == nil {
		t.Error("CommandCounts 不应为 nil")
	}

	if result.ExitCodes == nil {
		t.Error("ExitCodes 不应为 nil")
	}

	if result.DailyStats == nil {
		t.Error("DailyStats 不应为 nil")
	}

	if result.TotalDuration < 0 {
		t.Error("TotalDuration 不应为负数")
	}
}

func TestAnalyzeSkipNonLogFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建非.log文件
	nonLogFiles := []string{"test.txt", "readme.md", "data.json"}
	for _, filename := range nonLogFiles {
		path := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatalf("创建文件失败: %v", err)
		}
	}

	analyzer := stats.New(tmpDir)
	result, err := analyzer.Analyze(context.Background())
	if err != nil {
		t.Fatalf("Analyze() 失败: %v", err)
	}

	// 非.log文件不应被分析
	if result.TotalCommands != 0 {
		t.Errorf("非.log文件不应被分析: TotalCommands = %d, want 0", result.TotalCommands)
	}
}

func TestPrintStats(t *testing.T) {
	// 创建测试统计数据
	testStats := &stats.Stats{
		TotalCommands:   10,
		SuccessCommands: 8,
		FailedCommands:  2,
		TotalDuration:   10 * time.Second,
		CommandCounts: map[string]int{
			"echo": 5,
			"ls":   3,
			"pwd":  2,
		},
		ExitCodes: map[int]int{
			0: 8,
			1: 2,
		},
		DailyStats: map[string]*stats.DayStats{
			"2024-01-15": {
				Date:     "2024-01-15",
				Commands: 5,
				Success:  4,
				Failed:   1,
				Duration: 5 * time.Second,
			},
		},
	}

	// PrintStats 应该不会 panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PrintStats() panic: %v", r)
		}
	}()

	stats.PrintStats(testStats)
}

func TestPrintStatsWithEmptyData(t *testing.T) {
	emptyStats := &stats.Stats{
		CommandCounts: make(map[string]int),
		ExitCodes:     make(map[int]int),
		DailyStats:    make(map[string]*stats.DayStats),
	}

	// PrintStats 应该能处理空数据
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PrintStats() panic on empty data: %v", r)
		}
	}()

	stats.PrintStats(emptyStats)
}
