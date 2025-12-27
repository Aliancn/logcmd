package logger

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aliancn/logcmd/internal/config"
	"github.com/aliancn/logcmd/internal/executor"
	"github.com/aliancn/logcmd/internal/history"
	"github.com/aliancn/logcmd/internal/model"
	"github.com/aliancn/logcmd/internal/registry"
	"github.com/aliancn/logcmd/internal/stats"
)

// Logger 日志记录器
type Logger struct {
	config *config.Config
	file   *os.File
	writer *bufio.Writer
}

// New 创建新的日志记录器
func New(cfg *config.Config) (*Logger, error) {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	return &Logger{
		config: cfg,
	}, nil
}

// Run 执行命令并记录日志
func (l *Logger) Run(ctx context.Context, command string, args ...string) error {
	// 设置命令信息（用于生成日志文件名）
	l.config.Command = command
	l.config.CommandArgs = args

	// 生成日志文件路径
	logPath, err := l.config.GetLogFilePath()
	if err != nil {
		return fmt.Errorf("生成日志路径失败: %w", err)
	}

	// 自动注册项目（如果尚未注册）
	project := registerProject(l.config.LogDir)

	// 打开日志文件
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}
	defer file.Close()

	l.file = file
	l.writer = bufio.NewWriterSize(file, l.config.BufferSize)
	defer l.writer.Flush()

	// 显示日志文件路径
	fmt.Printf("正在记录日志到: %s\n", logPath)

	// 写入日志头部
	l.writeHeader(command, args)

	// 创建执行器并执行命令
	exec := executor.New(l.writer)
	result, err := exec.Execute(ctx, command, args...)

	// 写入元数据
	if result != nil {
		exec.WriteMetadata(result)
		updateProjectStats(project, result)
		persistCommandHistory(project, result, logPath)
	}

	// 刷新缓冲区
	if err := l.writer.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "刷新日志缓冲区失败: %v\n", err)
	}

	if err != nil {
		return fmt.Errorf("命令执行失败: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("命令退出码: %d", result.ExitCode)
	}

	return nil
}

// registerProject 自动注册项目到全局数据库
func registerProject(logDir string) *model.Project {
	reg, err := registry.New()
	if err != nil {
		return nil
	}
	defer reg.Close()

	project, err := reg.Register(logDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "注册项目失败: %v\n", err)
		return nil
	}

	return project
}

// updateProjectStats 根据执行结果更新项目级统计
func updateProjectStats(project *model.Project, result *executor.Result) {
	if project == nil || result == nil {
		return
	}

	reg, err := registry.New()
	if err != nil {
		return
	}
	defer reg.Close()

	if err := reg.UpdateStats(project.ID, result.Command, result.Success, result.Duration); err != nil {
		fmt.Fprintf(os.Stderr, "更新项目统计失败: %v\n", err)
	}
}

// persistCommandHistory 将执行记录写入数据库并刷新统计缓存。
func persistCommandHistory(project *model.Project, result *executor.Result, logFilePath string) {
	if project == nil || result == nil {
		return
	}

	reg, err := registry.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化命令历史存储失败: %v\n", err)
		return
	}
	defer reg.Close()

	db := reg.GetDB()
	historyManager := history.NewManager(db)
	statsManager := stats.NewCacheManager(db)

	logDate := result.StartTime.Format("2006-01-02")
	if logDate == "" {
		logDate = time.Now().Format("2006-01-02")
	}

	record := &model.CommandHistory{
		ProjectID:        project.ID,
		Command:          buildCommandString(result.Command, result.Args),
		CommandArgs:      result.Args,
		StartTime:        result.StartTime,
		EndTime:          result.EndTime,
		DurationMs:       result.Duration.Milliseconds(),
		ExitCode:         result.ExitCode,
		Status:           map[bool]string{true: "success", false: "failed"}[result.Success],
		LogFilePath:      logFilePath,
		LogDate:          logDate,
		HasError:         !result.Success,
		WorkingDirectory: getWorkingDirectory(),
		CreatedAt:        time.Now(),
	}

	if err := historyManager.Record(record); err != nil {
		fmt.Fprintf(os.Stderr, "记录命令历史失败: %v\n", err)
		return
	}

	if err := statsManager.GenerateForDate(project.ID, logDate); err != nil {
		fmt.Fprintf(os.Stderr, "刷新统计缓存失败: %v\n", err)
	}
}

func buildCommandString(command string, args []string) string {
	parts := []string{}
	if command != "" {
		parts = append(parts, command)
	}
	if len(args) > 0 {
		parts = append(parts, args...)
	}
	return strings.Join(parts, " ")
}

func getWorkingDirectory() string {
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return ""
}

// writeHeader 写入日志头部信息
func (l *Logger) writeHeader(command string, args []string) {
	header := fmt.Sprintf(`
################################################################################
# LogCmd - 命令执行日志
# 时间: %s
# 命令: %s %v
################################################################################

`,
		time.Now().In(l.config.TimeZone).Format("2006-01-02 15:04:05"),
		command,
		args,
	)

	l.writer.WriteString(header)
}

// Close 关闭日志记录器
func (l *Logger) Close() error {
	if l.writer != nil {
		if err := l.writer.Flush(); err != nil {
			return err
		}
	}
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}
