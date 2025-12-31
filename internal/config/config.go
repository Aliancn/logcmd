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
	TimeFormat   string         // 时间格式
	Command      string         // 当前执行的命令
	CommandArgs  []string       // 命令参数
}

// Load 加载配置
// 优先级: 默认值 < 全局配置 < 局部配置
func Load() (*Config, error) {
	// 1. 初始化基础配置（硬编码默认值）
	baseCfg := DefaultConfig()

	// 2. 加载全局配置 (~/.logcmd/config.json)
	globalPath, err := GetGlobalConfigPath()
	if err == nil {
		if globalCfg, err := LoadConfigFile(globalPath); err == nil && globalCfg != nil {
			mergeConfig(baseCfg, globalCfg)
		}
	}

	// 3. 加载局部配置 (.logcmd/config.json)
	cwd, _ := os.Getwd()
	localPath, err := GetLocalConfigPath(cwd)
	if err == nil && localPath != "" {
		if localCfg, err := LoadConfigFile(localPath); err == nil && localCfg != nil {
			mergeConfig(baseCfg, localCfg)
		}
	}

	return baseCfg, nil
}

// mergeConfig 将 PersistentConfig 合并到 Config
func mergeConfig(dst *Config, src *PersistentConfig) {
	if src.BufferSize > 0 {
		dst.BufferSize = src.BufferSize
	}
	// bool 类型比较难判断是否设置了（默认为false），这里简单覆盖
	// 实际生产中可能需要使用指针或 map 来区分"未设置"和"设置为false"
	// 但为了简单，假设配置文件中存在即覆盖（当前 JSON omitempty 机制下，false 也会被忽略，需要注意）
	// TODO: 改进 bool 类型的合并策略，当前暂且认为如果 src 加载了就覆盖
	// 由于 PersistentConfig 定义了 omitempty，如果文件中是 false，Unmarshal 后也是 false
	// 这里暂时无法区分"文件中显式写了false"和"文件中没写默认false"。
	// 鉴于目前默认值是 false，如果用户想开启 (true)，则可以覆盖。
	// 如果默认值改成 true，用户想关闭 (false)，则需要指针。
	// 既然默认是 false，那么只有 true 有意义。
	if src.AutoCompress {
		dst.AutoCompress = src.AutoCompress
	}

	if src.TimeFormat != "" {
		dst.TimeFormat = src.TimeFormat
	}
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
		TimeFormat:   "20060102_150405",
	}
}

// findLogDir 查找合适的日志目录
func findLogDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ".logcmd"
	}
	return findLogDirFrom(cwd)
}

// findLogDirFrom 从指定目录开始查找合适的日志目录
// 1. 优先在当前目录查找 .logcmd
// 2. 向上查找父目录中的 .logcmd
// 3. 如果都没找到，在当前目录创建 .logcmd
func findLogDirFrom(startDir string) string {
	// 标准化路径
	startDir = filepath.Clean(startDir)

	// 记录全局目录（$HOME/.logcmd）
	var homeDir, homeLogcmd string
	if home, err := os.UserHomeDir(); err == nil {
		homeDir = filepath.Clean(home)
		homeLogcmd = filepath.Join(homeDir, ".logcmd")
	}

	// 从当前目录开始向上查找 .logcmd
	currentDir := startDir
	for {
		// 检查当前目录是否存在 .logcmd
		logcmdPath := filepath.Join(currentDir, ".logcmd")
		if info, err := os.Stat(logcmdPath); err == nil && info.IsDir() {
			// 只有当命令在 home 目录直接执行时才使用全局 .logcmd
			if homeLogcmd != "" && filepath.Clean(logcmdPath) == homeLogcmd {
				if homeDir != "" && filepath.Clean(startDir) == homeDir {
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
	if homeDir != "" && filepath.Clean(startDir) == homeDir && homeLogcmd != "" {
		return homeLogcmd
	}

	// 没有找到 .logcmd，在当前工作目录创建
	return filepath.Join(startDir, ".logcmd")
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
	filename := tmpl.GenerateLogName(c.Command, c.CommandArgs, projectName, c.TimeZone, c.TimeFormat)

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
