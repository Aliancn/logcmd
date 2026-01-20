package execution

import (
	"context"
	"fmt"

	"github.com/aliancn/logcmd/internal/domain/config"
	"github.com/aliancn/logcmd/internal/platform/executor"
	"github.com/aliancn/logcmd/internal/platform/logger"
)

// Options 控制命令执行时的可选行为。
type Options struct {
	// LogPathOverride 用于预先指定日志文件路径（例如后台任务需要提前写入数据库）。
	LogPathOverride string
}

// Runner 负责根据配置执行命令，是命令行与底层 logger、持久化层之间的防腐层。
// 通过集中处理依赖，它满足单一职责，同时通过接口依赖 logger.RunRepository/ProjectStatsUpdater 体现依赖倒置。
type Runner struct {
	repo  logger.RunRepository
	stats logger.ProjectStatsUpdater
}

// NewRunner 创建 Runner。
func NewRunner(repo logger.RunRepository, stats logger.ProjectStatsUpdater) *Runner {
	return &Runner{
		repo:  repo,
		stats: stats,
	}
}

// Execute 根据给定配置执行命令并返回执行结果与日志路径。
func (r *Runner) Execute(ctx context.Context, cfg *config.Config, command string, args []string, opts Options) (*executor.Result, string, error) {
	if cfg == nil {
		return nil, "", fmt.Errorf("config 不能为空")
	}

	log, err := logger.New(cfg, r.repo, r.stats)
	if err != nil {
		return nil, "", err
	}
	defer log.Close()

	if opts.LogPathOverride != "" {
		log.SetLogPath(opts.LogPathOverride)
	}

	return log.Run(ctx, command, args...)
}

// PrepareLogPath 根据命令上下文生成日志文件路径，供需要提前暴露日志路径的场景复用。
func (r *Runner) PrepareLogPath(cfg *config.Config, command string, args []string) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("config 不能为空")
	}
	cfg.Command = command
	cfg.CommandArgs = args
	return cfg.GetLogFilePath()
}
