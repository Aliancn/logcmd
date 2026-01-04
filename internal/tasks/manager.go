package tasks

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aliancn/logcmd/internal/model"
)

// ErrTaskStateChanged 代表任务状态已被修改
var ErrTaskStateChanged = errors.New("task state changed")

// Manager 提供后台任务的增删改查能力
type Manager struct {
	db *sql.DB
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

// NewManager 创建任务管理器
func NewManager(db *sql.DB) *Manager {
	if db == nil {
		return nil
	}
	return &Manager{db: db}
}

// Create 新建任务
func (m *Manager) Create(task *model.Task) (*model.Task, error) {
	if m == nil || m.db == nil {
		return nil, fmt.Errorf("任务管理器未初始化")
	}
	if task == nil {
		return nil, fmt.Errorf("任务不能为空")
	}

	if err := task.BeforeSave(); err != nil {
		return nil, err
	}

	now := time.Now()
	task.CreatedAt = now
	task.UpdatedAt = now

	result, err := m.db.Exec(`
		INSERT INTO tasks (command, command_args, working_dir, log_dir, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, task.Command, task.ArgsJSON, task.WorkingDir, task.LogDir, task.Status, task.CreatedAt, task.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("创建任务失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("获取任务ID失败: %w", err)
	}
	task.ID = int(id)

	return task, nil
}

// Get 根据ID获取任务
func (m *Manager) Get(id int) (*model.Task, error) {
	if m == nil || m.db == nil {
		return nil, fmt.Errorf("任务管理器未初始化")
	}

	row := m.db.QueryRow(`
		SELECT id, command, command_args, working_dir, log_dir, status,
		       pid, IFNULL(log_file_path, ''), exit_code, IFNULL(error_message, ''), created_at, updated_at,
		       started_at, completed_at
		FROM tasks
		WHERE id = ?
	`, id)

	task, err := scanTask(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("任务不存在: %d", id)
		}
		return nil, fmt.Errorf("查询任务失败: %w", err)
	}

	return task, nil
}

// ListActive 列出仍在运行或准备运行的任务
func (m *Manager) ListActive() ([]*model.Task, error) {
	if m == nil || m.db == nil {
		return nil, fmt.Errorf("任务管理器未初始化")
	}

	rows, err := m.db.Query(`
		SELECT id, command, command_args, working_dir, log_dir, status,
		       pid, IFNULL(log_file_path, ''), exit_code, IFNULL(error_message, ''), created_at, updated_at,
		       started_at, completed_at
		FROM tasks
		WHERE status IN (?, ?)
		ORDER BY created_at ASC
	`, model.TaskStatusPending, model.TaskStatusRunning)
	if err != nil {
		return nil, fmt.Errorf("查询任务失败: %w", err)
	}
	defer rows.Close()

	var tasksList []*model.Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("读取任务失败: %w", err)
		}
		tasksList = append(tasksList, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("读取任务失败: %w", err)
	}

	return tasksList, nil
}

func scanTask(scanner rowScanner) (*model.Task, error) {
	var (
		pid         sql.NullInt64
		exitCode    sql.NullInt64
		startedAt   sql.NullTime
		completedAt sql.NullTime
	)

	task := &model.Task{}
	if err := scanner.Scan(
		&task.ID,
		&task.Command,
		&task.ArgsJSON,
		&task.WorkingDir,
		&task.LogDir,
		&task.Status,
		&pid,
		&task.LogFilePath,
		&exitCode,
		&task.ErrorMessage,
		&task.CreatedAt,
		&task.UpdatedAt,
		&startedAt,
		&completedAt,
	); err != nil {
		return nil, err
	}

	task.PID = nullInt64Ptr(pid)
	task.ExitCode = nullInt64Ptr(exitCode)
	task.StartedAt = nullTimePtr(startedAt)
	task.CompletedAt = nullTimePtr(completedAt)

	if err := task.AfterLoad(); err != nil {
		return nil, err
	}

	return task, nil
}

func nullInt64Ptr(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	v := value.Int64
	return &v
}

func nullTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time
	return &t
}

// UpdatePID 更新任务的进程信息
func (m *Manager) UpdatePID(id int, pid int) error {
	if m == nil || m.db == nil {
		return fmt.Errorf("任务管理器未初始化")
	}
	now := time.Now()
	_, err := m.db.Exec(`UPDATE tasks SET pid = ?, updated_at = ? WHERE id = ?`, pid, now, id)
	return err
}

// UpdateLogFilePath 更新任务的日志文件路径
func (m *Manager) UpdateLogFilePath(id int, path string) error {
	if m == nil || m.db == nil {
		return fmt.Errorf("任务管理器未初始化")
	}
	now := time.Now()
	_, err := m.db.Exec(`UPDATE tasks SET log_file_path = ?, updated_at = ? WHERE id = ?`, path, now, id)
	return err
}

// MarkRunning 将任务标记为运行中
func (m *Manager) MarkRunning(id int, pid int) error {
	if m == nil || m.db == nil {
		return fmt.Errorf("任务管理器未初始化")
	}
	now := time.Now()
	result, err := m.db.Exec(`
		UPDATE tasks SET status = ?, pid = ?, started_at = ?, updated_at = ?
		WHERE id = ? AND status IN (?, ?)
	`, model.TaskStatusRunning, pid, now, now, id, model.TaskStatusPending, model.TaskStatusRunning)
	if err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}
	rows, err := result.RowsAffected()
	if err == nil && rows == 0 {
		return ErrTaskStateChanged
	}
	return nil
}

// MarkCompletion 完成任务并记录结果
func (m *Manager) MarkCompletion(id int, status string, exitCode int, logFilePath string, errMsg string) error {
	if m == nil || m.db == nil {
		return fmt.Errorf("任务管理器未初始化")
	}

	now := time.Now()
	if strings.TrimSpace(status) == "" {
		status = model.TaskStatusSuccess
	}

	_, err := m.db.Exec(`
		UPDATE tasks SET
			status = ?,
			exit_code = ?,
			log_file_path = ?,
			error_message = ?,
			completed_at = ?,
			updated_at = ?,
			pid = NULL
		WHERE id = ?
	`, status, exitCode, logFilePath, errMsg, now, now, id)
	if err != nil {
		return fmt.Errorf("记录任务结果失败: %w", err)
	}

	return nil
}

// MarkStopped 标记任务被终止
func (m *Manager) MarkStopped(id int, status string, errMsg string) error {
	if m == nil || m.db == nil {
		return fmt.Errorf("任务管理器未初始化")
	}
	if status == "" {
		status = model.TaskStatusStopped
	}
	now := time.Now()
	result, err := m.db.Exec(`
		UPDATE tasks SET
			status = ?,
			exit_code = -1,
			error_message = ?,
			completed_at = ?,
			updated_at = ?,
			pid = NULL
		WHERE id = ? AND status IN (?, ?)
	`, status, errMsg, now, now, id, model.TaskStatusPending, model.TaskStatusRunning)
	if err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}
	rows, err := result.RowsAffected()
	if err == nil && rows == 0 {
		return fmt.Errorf("任务已结束或不存在")
	}
	return nil
}
