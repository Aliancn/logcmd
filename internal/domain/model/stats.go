package model

import (
	"encoding/json"
	"time"
)

// ProjectStatsCache 项目统计数据缓存（按日期）
type ProjectStatsCache struct {
	ID        int    `db:"id"`
	ProjectID int    `db:"project_id"`
	StatDate  string `db:"stat_date"` // YYYY-MM-DD

	// 每日统计
	TotalCommands   int   `db:"total_commands"`
	SuccessCommands int   `db:"success_commands"`
	FailedCommands  int   `db:"failed_commands"`
	TotalDurationMs int64 `db:"total_duration_ms"`
	AvgDurationMs   int64 `db:"avg_duration_ms"`
	MaxDurationMs   int64 `db:"max_duration_ms"`
	MinDurationMs   int64 `db:"min_duration_ms"`

	// 分布统计（JSON 存储）
	CommandDistribution  map[string]int `db:"-"`
	CommandDistJSON      string         `db:"command_distribution"`
	ExitCodeDistribution map[int]int    `db:"-"`
	ExitCodeDistJSON     string         `db:"exit_code_distribution"`

	// 时间戳
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// BeforeSave 在保存前序列化 JSON 字段
func (s *ProjectStatsCache) BeforeSave() error {
	if s.CommandDistribution != nil {
		cmdJSON, err := json.Marshal(s.CommandDistribution)
		if err != nil {
			return err
		}
		s.CommandDistJSON = string(cmdJSON)
	}

	if s.ExitCodeDistribution != nil {
		exitJSON, err := json.Marshal(s.ExitCodeDistribution)
		if err != nil {
			return err
		}
		s.ExitCodeDistJSON = string(exitJSON)
	}

	return nil
}

// AfterLoad 在加载后反序列化 JSON 字段
func (s *ProjectStatsCache) AfterLoad() error {
	if s.CommandDistJSON != "" {
		if err := json.Unmarshal([]byte(s.CommandDistJSON), &s.CommandDistribution); err != nil {
			return err
		}
	}

	if s.ExitCodeDistJSON != "" {
		if err := json.Unmarshal([]byte(s.ExitCodeDistJSON), &s.ExitCodeDistribution); err != nil {
			return err
		}
	}

	return nil
}

// GetSuccessRate 计算成功率
func (s *ProjectStatsCache) GetSuccessRate() float64 {
	if s.TotalCommands == 0 {
		return 0
	}
	return float64(s.SuccessCommands) / float64(s.TotalCommands) * 100
}

// GetAvgDuration 获取平均执行时长
func (s *ProjectStatsCache) GetAvgDuration() time.Duration {
	return time.Duration(s.AvgDurationMs) * time.Millisecond
}

// GetMaxDuration 获取最长执行时长
func (s *ProjectStatsCache) GetMaxDuration() time.Duration {
	return time.Duration(s.MaxDurationMs) * time.Millisecond
}

// GetMinDuration 获取最短执行时长
func (s *ProjectStatsCache) GetMinDuration() time.Duration {
	return time.Duration(s.MinDurationMs) * time.Millisecond
}
