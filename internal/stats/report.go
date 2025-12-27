package stats

import (
	"time"

	"github.com/aliancn/logcmd/internal/model"
)

// FromCache 构造数据库统计报告
func FromCache(cache *model.ProjectStatsCache, projectName string) *Stats {
	if cache == nil {
		return nil
	}

	commandDist := cache.CommandDistribution
	if commandDist == nil {
		commandDist = make(map[string]int)
	}

	exitDist := cache.ExitCodeDistribution
	if exitDist == nil {
		exitDist = make(map[int]int)
	}

	report := &Stats{
		ProjectName:     projectName,
		RangeLabel:      cache.StatDate,
		Source:          SourceDatabase,
		TotalCommands:   cache.TotalCommands,
		SuccessCommands: cache.SuccessCommands,
		FailedCommands:  cache.FailedCommands,
		TotalDuration:   time.Duration(cache.TotalDurationMs) * time.Millisecond,
		AvgDuration:     time.Duration(cache.AvgDurationMs) * time.Millisecond,
		MaxDuration:     time.Duration(cache.MaxDurationMs) * time.Millisecond,
		MinDuration:     time.Duration(cache.MinDurationMs) * time.Millisecond,
		CommandCounts:   commandDist,
		ExitCodes:       exitDist,
		DailyStats:      make(map[string]*DayStats),
	}

	if report.AvgDuration == 0 && report.TotalCommands > 0 {
		report.AvgDuration = report.TotalDuration / time.Duration(report.TotalCommands)
	}

	return report
}
