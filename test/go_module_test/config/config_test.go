package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aliancn/logcmd/internal/config"
	"github.com/aliancn/logcmd/internal/template"
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

	// 验证默认时区跟随系统本地配置
	if cfg.TimeZone.String() != time.Local.String() {
		t.Errorf("默认时区应为本地时区: got %s, want %s", cfg.TimeZone, time.Local)
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

func TestGetLogFilePathGeneratesUniqueNames(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	// 配置模板为固定文件名，确保命名冲突
	customTemplate := &template.LogNameTemplate{
		Separator: "_",
		Elements: []template.NameElement{
			{Type: template.ElementTypeCustom, Config: map[string]string{"text": "fixed"}},
		},
	}
	if err := customTemplate.Save(); err != nil {
		t.Fatalf("保存模板失败: %v", err)
	}

	cfg := &config.Config{
		LogDir:      filepath.Join(tempDir, "logs"),
		TimeZone:    time.UTC,
		Command:     "test",
		CommandArgs: []string{},
	}

	path1, err := cfg.GetLogFilePath()
	if err != nil {
		t.Fatalf("第一次生成日志路径失败: %v", err)
	}

	// 创建文件模拟一次执行
	if err := os.MkdirAll(filepath.Dir(path1), 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}
	if err := os.WriteFile(path1, []byte("log"), 0644); err != nil {
		t.Fatalf("写入日志文件失败: %v", err)
	}

	path2, err := cfg.GetLogFilePath()
	if err != nil {
		t.Fatalf("第二次生成日志路径失败: %v", err)
	}

	if path1 == path2 {
		t.Errorf("应该生成唯一的日志文件路径，但得到相同路径: %s", path1)
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
