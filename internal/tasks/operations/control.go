package operations

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/aliancn/logcmd/internal/model"
	"github.com/aliancn/logcmd/internal/tasks"
)

// CheckProcessAlive 检查给定 PID 对应的进程是否仍存在。
func CheckProcessAlive(pid *int64) bool {
	if pid == nil || *pid <= 0 {
		return false
	}
	process, err := os.FindProcess(int(*pid))
	if err != nil {
		return false
	}
	// 信号 0 不会影响进程，仅用于探测。
	if err = process.Signal(syscall.Signal(0)); err == nil {
		return true
	}
	if errors.Is(err, os.ErrProcessDone) {
		return false
	}
	if errno, ok := err.(syscall.Errno); ok && errno == syscall.ESRCH {
		return false
	}
	// 其他错误（如 EPERM）代表进程仍存在。
	return true
}

// StopTask 请求停止或终止任务，并更新数据库状态。
func StopTask(manager *tasks.Manager, task *model.Task, force bool) (string, error) {
	if manager == nil {
		return "", fmt.Errorf("任务管理器未初始化")
	}
	if task == nil {
		return "", fmt.Errorf("任务不存在")
	}

	if !task.IsActive() {
		return "已结束", nil
	}

	if task.PID != nil && *task.PID > 0 {
		if proc, err := os.FindProcess(int(*task.PID)); err == nil {
			if force {
				_ = proc.Kill()
			} else {
				if termErr := proc.Signal(os.Interrupt); termErr != nil && !errors.Is(termErr, os.ErrProcessDone) {
					_ = proc.Kill()
				}
			}
		}
	}

	action := "停止"
	status := model.TaskStatusStopped
	if force {
		action = "终止"
		status = model.TaskStatusFailed
	}

	if err := manager.MarkStopped(task.ID, status, fmt.Sprintf("用户请求%s任务", action)); err != nil {
		return "", err
	}

	task.Status = status
	task.PID = nil
	return action, nil
}
