package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aliancn/logcmd/internal/config"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig() 返回了 nil")
	}

	if cfg.LogDir == "" {
		t.Error("LogDir 不应为空")
	}

	if cfg.TimeZone == nil {
		t.Error("TimeZone 不应为 nil")
	}

	// 验证时区是否为东八区
	_, offset := time.Now().In(cfg.TimeZone).Zone()
	expectedOffset := 8 * 3600 // 东八区偏移量（秒）
	if offset != expectedOffset {
		t.Errorf("时区偏移量不正确: got %d, want %d", offset, expectedOffset)
	}

	if cfg.BufferSize <= 0 {
		t.Error("BufferSize 应该大于 0")
	}
}

func TestGetLogFilePath(t *testing.T) {
	// 创建临时目录用于测试
	tempDir := t.TempDir()

	cfg := &config.Config{
		LogDir:      tempDir,
		TimeZone:    time.UTC,
		Command:     "echo",
		CommandArgs: []string{"test"},
	}

	logPath, err := cfg.GetLogFilePath()
	if err != nil {
		t.Fatalf("GetLogFilePath() 失败: %v", err)
	}

	if logPath == "" {
		t.Error("日志路径不应为空")
	}

	// 验证路径包含日期目录
	now := time.Now().In(cfg.TimeZone)
	expectedDateDir := now.Format("2006-01-02")
	if !filepath.HasPrefix(logPath, filepath.Join(tempDir, expectedDateDir)) {
		t.Errorf("日志路径不包含预期的日期目录: got %s", logPath)
	}

	// 验证日志文件扩展名
	if filepath.Ext(logPath) != ".log" {
		t.Errorf("日志文件应该有 .log 扩展名: got %s", logPath)
	}

	// 验证日期目录已创建
	dateDir := filepath.Dir(logPath)
	if _, err := os.Stat(dateDir); os.IsNotExist(err) {
		t.Error("日期目录应该已经创建")
	}
}

func TestGetLogFilePathWithDifferentCommands(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name    string
		command string
		args    []string
	}{
		{"简单命令", "ls", []string{"-la"}},
		{"带空格命令", "git", []string{"commit", "-m", "test message"}},
		{"无参数命令", "pwd", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				LogDir:      tempDir,
				TimeZone:    time.UTC,
				Command:     tt.command,
				CommandArgs: tt.args,
			}

			logPath, err := cfg.GetLogFilePath()
			if err != nil {
				t.Fatalf("GetLogFilePath() 失败: %v", err)
			}

			if logPath == "" {
				t.Error("日志路径不应为空")
			}
		})
	}
}

func TestConfigWithNilTimeZone(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		LogDir:      tempDir,
		TimeZone:    nil, // 测试 nil 时区
		Command:     "test",
		CommandArgs: []string{},
	}

	// TimeZone为nil时会panic，这是预期的行为
	defer func() {
		if r := recover(); r == nil {
			t.Error("GetLogFilePath() 在 TimeZone 为 nil 时应该 panic")
		}
	}()

	cfg.GetLogFilePath()
}
