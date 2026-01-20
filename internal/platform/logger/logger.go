package logger

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/aliancn/logcmd/internal/domain/config"
	"github.com/aliancn/logcmd/internal/platform/executor"
	"github.com/aliancn/logcmd/internal/domain/model"
)

// Logger 日志记录器
type Logger struct {
	config       *config.Config
	repo         RunRepository
	statsUpdater ProjectStatsUpdater
	file         *os.File
	writer       *bufio.Writer
	logPath      string // 预设的日志路径
	mu           sync.Mutex
	lastFlush    time.Time

	// Async Persistence
	persistCh chan func()
	wg        sync.WaitGroup
}

// RunRepository 抽象运行结果的持久化能力
type RunRepository interface {
	RegisterProject(path string) (*model.Project, error)
	RecordRun(project *model.Project, result *executor.Result, logFilePath string) error
}

// ProjectStatsUpdater 负责项目级别的统计更新
type ProjectStatsUpdater interface {
	UpdateProjectStats(projectID int, command string, success bool, duration time.Duration) error
}

// New 创建新的日志记录器
func New(cfg *config.Config, repo RunRepository, statsUpdater ProjectStatsUpdater) (*Logger, error) {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	l := &Logger{
		config:       cfg,
		repo:         repo,
		statsUpdater: statsUpdater,
		persistCh:    make(chan func(), 100), // Buffer size 100
	}

	l.wg.Add(1)
	go l.processPersistence()

	return l, nil
}

func (l *Logger) processPersistence() {
	defer l.wg.Done()
	for task := range l.persistCh {
		task()
	}
}

// SetLogPath 设置强制使用的日志路径
func (l *Logger) SetLogPath(path string) {
	l.logPath = path
}

// Run 执行命令并记录日志
func (l *Logger) Run(ctx context.Context, command string, args ...string) (*executor.Result, string, error) {
	// 检查白名单
	if len(l.config.Whitelist) > 0 {
		allowed := false
		for _, allow := range l.config.Whitelist {
			if allow == command {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, "", fmt.Errorf("命令 '%s' 不在白名单中，拒绝执行", command)
		}
	}

	// 设置命令信息（用于生成日志文件名）
	l.config.Command = command
	l.config.CommandArgs = args

	var logPath string
	var err error

	if l.logPath != "" {
		logPath = l.logPath
	} else {
		// 生成日志文件路径
		logPath, err = l.config.GetLogFilePath()
		if err != nil {
			return nil, "", fmt.Errorf("生成日志路径失败: %w", err)
		}
	}

	// 自动注册项目（如果尚未注册）
	var project *model.Project
	if l.repo != nil {
		project, err = l.repo.RegisterProject(l.config.LogDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "注册项目失败: %v\n", err)
		}
	}

	// 打开日志文件
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, "", fmt.Errorf("打开日志文件失败: %w", err)
	}
	defer file.Close()

	l.file = file
	l.writer = bufio.NewWriterSize(file, l.config.BufferSize)
	l.lastFlush = time.Now()

	// 确保最后刷新
	defer func() {
		l.mu.Lock()
		defer l.mu.Unlock()
		if l.writer != nil {
			l.writer.Flush()
		}
	}()

	// 显示日志文件路径
	fmt.Printf("正在记录日志到: %s\n", logPath)

	// 写入日志头部
	l.writeHeader(command, args)

	// 创建带锁的 writer
	sw := &syncedWriter{l: l}

	// 创建执行器并执行命令
	exec := executor.New(sw, os.Stdout, os.Stderr)
	result, err := exec.Execute(ctx, command, args...)

	// 写入元数据
	if result != nil {
		exec.WriteMetadata(result)

		if project != nil {
			// Async Update Stats
			if l.statsUpdater != nil {
				l.persistCh <- func() {
					if err := l.statsUpdater.UpdateProjectStats(project.ID, result.Command, result.Success, result.Duration); err != nil {
						fmt.Fprintf(os.Stderr, "更新项目统计失败: %v\n", err)
					}
				}
			}

			// Async Record History
			if l.repo != nil {
				l.persistCh <- func() {
					if err := l.repo.RecordRun(project, result, logPath); err != nil {
						fmt.Fprintf(os.Stderr, "记录命令历史失败: %v\n", err)
					}
				}
			}
		}
	}

	if err != nil {
		return result, logPath, fmt.Errorf("命令执行失败: %w", err)
	}

	if !result.Success {
		return result, logPath, fmt.Errorf("命令退出码: %d", result.ExitCode)
	}

	return result, logPath, nil
}

// writeHeader 写入日志头部信息
func (l *Logger) writeHeader(command string, args []string) {
	l.mu.Lock()
	defer l.mu.Unlock()

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
	l.writer.Flush()
	l.lastFlush = time.Now()
}

// Close 关闭日志记录器
func (l *Logger) Close() error {
	// 1. Close persistence channel and wait
	close(l.persistCh)
	l.wg.Wait()

	l.mu.Lock()
	defer l.mu.Unlock()

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

type syncedWriter struct {
	l *Logger
}

func (s *syncedWriter) Write(p []byte) (int, error) {
	s.l.mu.Lock()
	defer s.l.mu.Unlock()

	n, err := s.l.writer.Write(p)
	if err == nil {
		if time.Since(s.l.lastFlush) > s.l.config.FlushInterval {
			s.l.writer.Flush()
			s.l.lastFlush = time.Now()
		}
	}
	return n, err
}
