package model

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	TaskStatusPending = "pending"
	TaskStatusRunning = "running"
	TaskStatusSuccess = "success"
	TaskStatusFailed  = "failed"
	TaskStatusStopped = "stopped"
)

// Task 描述一个后台运行的命令
type Task struct {
	ID           int
	Command      string
	CommandArgs  []string
	ArgsJSON     string
	WorkingDir   string
	LogDir       string
	Status       string
	PID          *int64
	LogFilePath  string
	ExitCode     *int64
	ErrorMessage string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	StartedAt    *time.Time
	CompletedAt  *time.Time
}

// BeforeSave 在持久化前准备字段
func (t *Task) BeforeSave() error {
	if t.Command == "" {
		return fmt.Errorf("command 不能为空")
	}

	if t.CommandArgs != nil {
		argsJSON, err := json.Marshal(t.CommandArgs)
		if err != nil {
			return err
		}
		t.ArgsJSON = string(argsJSON)
	}

	if t.Status == "" {
		t.Status = TaskStatusPending
	}

	return nil
}

// AfterLoad 在读取后恢复字段
func (t *Task) AfterLoad() error {
	if t.ArgsJSON != "" {
		if err := json.Unmarshal([]byte(t.ArgsJSON), &t.CommandArgs); err != nil {
			return err
		}
	}
	return nil
}

// IsActive 判断任务是否仍在运行或等待
func (t *Task) IsActive() bool {
	return t.Status == TaskStatusPending || t.Status == TaskStatusRunning
}
