package persistence

import (
	"fmt"
	"time"

	"github.com/aliancn/logcmd/internal/registry"
)

// StatsUpdater 实现 logger.ProjectStatsUpdater，通过共享 Registry 更新聚合信息。
type StatsUpdater struct {
	registry *registry.Registry
}

// NewStatsUpdater 构造 StatsUpdater。
func NewStatsUpdater(reg *registry.Registry) *StatsUpdater {
	if reg == nil {
		return nil
	}
	return &StatsUpdater{registry: reg}
}

// UpdateProjectStats 持久化项目统计。
func (s *StatsUpdater) UpdateProjectStats(projectID int, command string, success bool, duration time.Duration) error {
	if s == nil || s.registry == nil {
		return fmt.Errorf("registry 未初始化")
	}
	return s.registry.UpdateStats(projectID, command, success, duration)
}
