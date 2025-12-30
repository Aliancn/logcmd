package services

import (
	"fmt"
	"os"
	"strings"

	"github.com/aliancn/logcmd/internal/model"
	"github.com/aliancn/logcmd/internal/registry"
	"github.com/aliancn/logcmd/internal/stats"
	"github.com/aliancn/logcmd/internal/template"
)

// StatsService 负责聚合统计策略（优先数据库缓存，失败时回退到日志扫描）。
type StatsService struct {
	registry *registry.Registry
	cache    *stats.CacheManager
}

// NewStatsService 创建统计服务。
func NewStatsService(reg *registry.Registry) *StatsService {
	var cache *stats.CacheManager
	if reg != nil {
		cache = stats.NewCacheManager(reg.GetDB())
	}
	return &StatsService{
		registry: reg,
		cache:    cache,
	}
}

// StatsForProject 返回单个项目的统计数据。
func (s *StatsService) StatsForProject(project *model.Project) (*stats.Stats, error) {
	if project == nil {
		return nil, fmt.Errorf("project 不能为空")
	}

	report, err := s.statsFromCache(project)
	if err == nil && report != nil {
		return report, nil
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "警告: 数据库统计失败，回退到日志扫描: %v\n", err)
	}

	report, logErr := s.statsFromLogs(project.Path, s.displayName(project))
	if logErr != nil {
		if err != nil {
			return nil, fmt.Errorf("日志扫描失败: %w; 数据库统计失败: %v", logErr, err)
		}
		return nil, fmt.Errorf("日志扫描失败: %w", logErr)
	}
	return report, nil
}

// StatsForPath 根据目录返回统计。若注册表可用则自动注册。
func (s *StatsService) StatsForPath(path string) (*stats.Stats, error) {
	if s.registry == nil {
		return s.statsFromLogs(path, template.GetProjectName(path))
	}

	project, err := s.ProjectByPath(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "警告: 注册项目失败，回退到日志扫描: %v\n", err)
		return s.statsFromLogs(path, template.GetProjectName(path))
	}

	return s.StatsForProject(project)
}

// ProjectByPath 查找（或注册）指定目录对应的项目。
func (s *StatsService) ProjectByPath(path string) (*model.Project, error) {
	if s.registry == nil {
		return nil, fmt.Errorf("项目注册表未初始化")
	}

	project, err := s.registry.Get(path)
	if err == nil {
		return project, nil
	}

	return s.registry.Register(path)
}

func (s *StatsService) statsFromCache(project *model.Project) (*stats.Stats, error) {
	if s.cache == nil {
		return nil, fmt.Errorf("统计缓存不可用")
	}

	if err := s.cache.Sync(project.ID); err != nil {
		return nil, fmt.Errorf("同步统计缓存失败: %w", err)
	}

	summary, err := s.cache.GetProjectSummary(project.ID)
	if err != nil {
		return nil, fmt.Errorf("获取统计缓存失败: %w", err)
	}

	if summary == nil {
		if err := s.cache.GenerateForProject(project.ID); err != nil {
			return nil, fmt.Errorf("生成统计缓存失败: %w", err)
		}
		summary, err = s.cache.GetProjectSummary(project.ID)
		if err != nil {
			return nil, fmt.Errorf("获取统计缓存失败: %w", err)
		}
		if summary == nil {
			return nil, fmt.Errorf("统计缓存不存在")
		}
	}

	report := stats.FromCache(summary, s.displayName(project))
	if report == nil {
		return nil, fmt.Errorf("统计缓存无效")
	}
	return report, nil
}

func (s *StatsService) statsFromLogs(path, displayName string) (*stats.Stats, error) {
	analyzer := stats.New(path)
	report, err := analyzer.Analyze()
	if err != nil {
		return nil, err
	}
	if displayName != "" {
		report.ProjectName = displayName
	}
	return report, nil
}

func (s *StatsService) displayName(project *model.Project) string {
	if project == nil {
		return ""
	}
	if name := strings.TrimSpace(project.Name); name != "" {
		return name
	}
	return template.GetProjectName(project.Path)
}
