package persistence

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aliancn/logcmd/internal/executor"
	"github.com/aliancn/logcmd/internal/history"
	"github.com/aliancn/logcmd/internal/model"
	"github.com/aliancn/logcmd/internal/registry"
	"github.com/aliancn/logcmd/internal/stats"
)

// RunRepository 提供命令执行结果的持久化实现，依赖共享的 Registry/DB。
type RunRepository struct {
	registry *registry.Registry
	history  *history.Manager
	cache    *stats.CacheManager
}

// NewRunRepository 创建 RunRepository。
func NewRunRepository(reg *registry.Registry) *RunRepository {
	if reg == nil {
		return nil
	}
	return &RunRepository{
		registry: reg,
		history:  history.NewManager(reg.GetDB()),
		cache:    stats.NewCacheManager(reg.GetDB()),
	}
}

// RegisterProject 注册项目。
func (r *RunRepository) RegisterProject(path string) (*model.Project, error) {
	if r == nil || r.registry == nil {
		return nil, fmt.Errorf("registry 未初始化")
	}
	return r.registry.Register(path)
}

// RecordRun 保存命令历史并刷新统计缓存。
func (r *RunRepository) RecordRun(project *model.Project, result *executor.Result, logFilePath string) error {
	if r == nil || r.history == nil || r.cache == nil {
		return fmt.Errorf("存储依赖未初始化")
	}
	if project == nil || result == nil {
		return fmt.Errorf("项目或结果不能为空")
	}

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

	if err := r.history.Record(record); err != nil {
		return err
	}

	if err := r.cache.GenerateForDate(project.ID, logDate); err != nil {
		return err
	}

	return nil
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
