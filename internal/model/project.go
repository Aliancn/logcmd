package model

import (
	"database/sql"
	"encoding/json"
	"time"
)

// Project 表示一个项目的完整信息
type Project struct {
	ID          int       `db:"id"`
	Path        string    `db:"path"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	Category    string    `db:"category"`
	Tags        []string  `db:"-"` // 从 JSON 解析
	TagsJSON    string    `db:"tags"`

	// 统计信息
	TotalCommands   int   `db:"total_commands"`
	SuccessCommands int   `db:"success_commands"`
	FailedCommands  int   `db:"failed_commands"`
	TotalDurationMs int64 `db:"total_duration_ms"`

	// 最后执行信息
	LastCommand       string         `db:"last_command"`
	LastCommandStatus string         `db:"last_command_status"`
	LastCommandTime   sql.NullTime   `db:"last_command_time"`

	// 时间戳
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
	LastChecked time.Time `db:"last_checked"`

	// 配置（JSON 存储）
	TemplateJSON string `db:"template_config"`
	CustomJSON   string `db:"custom_config"`
}

// BeforeSave 在保存前序列化 JSON 字段
func (p *Project) BeforeSave() error {
	if p.Tags != nil {
		tagsJSON, err := json.Marshal(p.Tags)
		if err != nil {
			return err
		}
		p.TagsJSON = string(tagsJSON)
	}
	return nil
}

// AfterLoad 在加载后反序列化 JSON 字段
func (p *Project) AfterLoad() error {
	if p.TagsJSON != "" {
		if err := json.Unmarshal([]byte(p.TagsJSON), &p.Tags); err != nil {
			return err
		}
	}
	return nil
}

// GetSuccessRate 计算成功率
func (p *Project) GetSuccessRate() float64 {
	if p.TotalCommands == 0 {
		return 0
	}
	return float64(p.SuccessCommands) / float64(p.TotalCommands) * 100
}

// GetAvgDuration 计算平均执行时长
func (p *Project) GetAvgDuration() time.Duration {
	if p.TotalCommands == 0 {
		return 0
	}
	return time.Duration(p.TotalDurationMs/int64(p.TotalCommands)) * time.Millisecond
}

// UpdateStats 更新统计信息（增量）
func (p *Project) UpdateStats(success bool, duration time.Duration) {
	p.TotalCommands++
	if success {
		p.SuccessCommands++
	} else {
		p.FailedCommands++
	}
	p.TotalDurationMs += duration.Milliseconds()
	p.UpdatedAt = time.Now()
}
