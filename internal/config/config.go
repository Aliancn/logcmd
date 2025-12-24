package config

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config 定义日志配置
type Config struct {
	LogDir       string        // 日志根目录
	TimeZone     *time.Location // 时区（默认东八区）
	BufferSize   int           // 缓冲区大小
	AutoCompress bool          // 是否自动压缩
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	// 东八区时间
	tz, _ := time.LoadLocation("Asia/Shanghai")

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

	// 从当前目录开始向上查找 .logcmd
	currentDir := cwd
	for {
		// 检查当前目录是否存在 .logcmd
		logcmdPath := filepath.Join(currentDir, ".logcmd")
		if info, err := os.Stat(logcmdPath); err == nil && info.IsDir() {
			// 找到了 .logcmd 目录
			return logcmdPath
		}

		// 获取父目录
		parentDir := filepath.Dir(currentDir)

		// 如果已经到达根目录，停止查找
		if parentDir == currentDir {
			break
		}

		currentDir = parentDir
	}

	// 没有找到 .logcmd，在当前工作目录创建
	return filepath.Join(cwd, ".logcmd")
}

// isSubDir 检查 dir 是否是 parent 的子目录
func isSubDir(dir, parent string) bool {
	// 确保路径以分隔符结尾，避免误匹配
	// 例如 /home/user2 不应该匹配 /home/user
	if !strings.HasSuffix(parent, string(filepath.Separator)) {
		parent += string(filepath.Separator)
	}
	if !strings.HasSuffix(dir, string(filepath.Separator)) {
		dir += string(filepath.Separator)
	}

	// 检查 dir 是否以 parent 开头
	return strings.HasPrefix(dir, parent)
}

// GetLogFilePath 生成日志文件路径，按日期组织目录
func (c *Config) GetLogFilePath() (string, error) {
	now := time.Now().In(c.TimeZone)

	// 按日期创建子目录 logs/2024-01-15/
	dateDir := filepath.Join(c.LogDir, now.Format("2006-01-02"))
	if err := os.MkdirAll(dateDir, 0755); err != nil {
		return "", err
	}

	// 日志文件名 log_20240115_143052.log
	filename := now.Format("log_20060102_150405.log")
	return filepath.Join(dateDir, filename), nil
}
