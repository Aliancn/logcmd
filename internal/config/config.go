package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aliancn/logcmd/internal/template"
)

// Config 定义日志配置
type Config struct {
	LogDir       string         // 日志根目录
	TimeZone     *time.Location // 时区（默认本地时区）
	BufferSize   int            // 缓冲区大小
	AutoCompress bool           // 是否自动压缩
	Command      string         // 当前执行的命令
	CommandArgs  []string       // 命令参数
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	// 使用系统本地时区，保证与运行环境一致
	tz := time.Local

	// 查找合适的日志目录
	logDir := findLogDir()

	return &Config{
		LogDir:       logDir,
		TimeZone:     tz,
		BufferSize:   8192,
		AutoCompress: false,
	}
}

// findLogDir 查找合适的日志目录
// 1. 优先在当前目录下查找 .logcmd
// 2. 向上查找父目录中的 .logcmd
// 3. 如果都没找到，在当前目录创建 .logcmd
func findLogDir() string {
	// 获取当前工作目录
	cwd, err := os.Getwd()
	if err != nil {
		return ".logcmd" // 出错则使用当前目录
	}

	// 标准化路径
	cwd = filepath.Clean(cwd)

	// 记录全局目录（$HOME/.logcmd），仅当用户直接在 home 下使用时才使用
	var homeDir, homeLogcmd string
	if home, err := os.UserHomeDir(); err == nil {
		homeDir = filepath.Clean(home)
		homeLogcmd = filepath.Join(homeDir, ".logcmd")
	}

	// 从当前目录开始向上查找 .logcmd
	currentDir := cwd
	for {
		// 检查当前目录是否存在 .logcmd
		logcmdPath := filepath.Join(currentDir, ".logcmd")
		if info, err := os.Stat(logcmdPath); err == nil && info.IsDir() {
			// 只有当命令在 home 目录直接执行时才使用全局 .logcmd
			if homeLogcmd != "" && filepath.Clean(logcmdPath) == homeLogcmd {
				if homeDir != "" && filepath.Clean(cwd) == homeDir {
					return logcmdPath
				}
			} else {
				return logcmdPath
			}
		}

		// 获取父目录
		parentDir := filepath.Dir(currentDir)

		// 如果已经到达根目录，停止查找
		if parentDir == currentDir {
			break
		}

		currentDir = parentDir
	}

	// 没有找到 .logcmd，如果当前目录是 home，则返回全局目录
	if homeDir != "" && filepath.Clean(cwd) == homeDir && homeLogcmd != "" {
		return homeLogcmd
	}

	// 没有找到 .logcmd，在当前工作目录创建
	return filepath.Join(cwd, ".logcmd")
}

// GetLogFilePath 生成日志文件路径，按日期组织目录
func (c *Config) GetLogFilePath() (string, error) {
	now := time.Now().In(c.TimeZone)

	// 确保LogDir存在
	if err := os.MkdirAll(c.LogDir, 0755); err != nil {
		return "", err
	}

	// 按日期创建子目录 logs/2024-01-15/
	dateDir := filepath.Join(c.LogDir, now.Format("2006-01-02"))
	if err := os.MkdirAll(dateDir, 0755); err != nil {
		return "", err
	}

	// 加载命名模板
	tmpl, err := template.Load()
	if err != nil {
		// 如果加载失败，使用默认命名
		tmpl = template.DefaultTemplate()
	}

	// 获取项目名称
	projectName := template.GetProjectName(c.LogDir)

	// 使用模板生成文件名
	filename := tmpl.GenerateLogName(c.Command, c.CommandArgs, projectName, c.TimeZone)

	logPath, err := ensureUniqueLogPath(dateDir, filename)
	if err != nil {
		return "", err
	}

	return logPath, nil
}

// ensureUniqueLogPath 如果同名日志已存在，则在文件名后添加序号，确保命名唯一
func ensureUniqueLogPath(dir, filename string) (string, error) {
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)

	candidate := filepath.Join(dir, filename)
	if _, err := os.Stat(candidate); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return candidate, nil
		}
		return "", fmt.Errorf("检查日志文件失败: %w", err)
	}

	for i := 1; i < 10000; i++ {
		newName := fmt.Sprintf("%s_%d%s", base, i, ext)
		candidate = filepath.Join(dir, newName)
		if _, err := os.Stat(candidate); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return candidate, nil
			}
			return "", fmt.Errorf("检查日志文件失败: %w", err)
		}
	}

	return "", fmt.Errorf("无法生成唯一日志文件: %s/%s", dir, filename)
}
