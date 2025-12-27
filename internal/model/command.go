package model

import (
	"encoding/json"
	"strings"
	"time"
)

// CommandHistory 记录单条命令的执行历史
type CommandHistory struct {
	ID        int       `db:"id"`
	ProjectID int       `db:"project_id"`

	// 命令信息
	Command     string   `db:"command"`
	CommandName string   `db:"command_name"`
	CommandArgs []string `db:"-"`
	ArgsJSON    string   `db:"command_args"`

	// 执行信息
	StartTime  time.Time `db:"start_time"`
	EndTime    time.Time `db:"end_time"`
	DurationMs int64     `db:"duration_ms"`
	ExitCode   int       `db:"exit_code"`
	Status     string    `db:"status"` // "success" or "failed"

	// 日志文件关联
	LogFilePath string `db:"log_file_path"`
	LogDate     string `db:"log_date"` // YYYY-MM-DD

	// 输出预览
	StdoutPreview string `db:"stdout_preview"`
	StderrPreview string `db:"stderr_preview"`
	HasError      bool   `db:"has_error"`

	// 元数据
	WorkingDirectory string `db:"working_directory"`
	EnvironmentJSON  string `db:"environment_info"`

	// 时间戳
	CreatedAt time.Time `db:"created_at"`
}

// BeforeSave 在保存前序列化 JSON 字段
func (c *CommandHistory) BeforeSave() error {
	if c.CommandArgs != nil {
		argsJSON, err := json.Marshal(c.CommandArgs)
		if err != nil {
			return err
		}
		c.ArgsJSON = string(argsJSON)
	}

	// 提取命令名称
	if c.Command != "" && c.CommandName == "" {
		parts := strings.Fields(c.Command)
		if len(parts) > 0 {
			c.CommandName = parts[0]
		}
	}

	// 设置状态
	if c.Status == "" {
		if c.ExitCode == 0 {
			c.Status = "success"
		} else {
			c.Status = "failed"
		}
	}

	return nil
}

// AfterLoad 在加载后反序列化 JSON 字段
func (c *CommandHistory) AfterLoad() error {
	if c.ArgsJSON != "" {
		if err := json.Unmarshal([]byte(c.ArgsJSON), &c.CommandArgs); err != nil {
			return err
		}
	}
	return nil
}

// GetDuration 获取执行时长
func (c *CommandHistory) GetDuration() time.Duration {
	return time.Duration(c.DurationMs) * time.Millisecond
}

// IsSuccess 判断是否执行成功
func (c *CommandHistory) IsSuccess() bool {
	return c.Status == "success"
}

// TruncateOutput 截断输出到指定长度
func TruncateOutput(output string, maxLen int) string {
	if len(output) <= maxLen {
		return output
	}
	return output[:maxLen] + "..."
}
