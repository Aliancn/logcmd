package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/aliancn/logcmd/internal/config"
	"github.com/aliancn/logcmd/internal/logger"
	"github.com/aliancn/logcmd/internal/model"
	"github.com/aliancn/logcmd/internal/persistence"
	"github.com/aliancn/logcmd/internal/tasks"
	"github.com/spf13/cobra"
)

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "管理后台运行的命令任务",
}

var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出仍在运行的后台任务",
	RunE: func(cmd *cobra.Command, args []string) error {
		return listTasks()
	},
}

var taskKillCmd = &cobra.Command{
	Use:   "kill <id>",
	Short: "强制终止后台任务",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return stopTask(args[0], true)
	},
}

var taskStopCmd = &cobra.Command{
	Use:   "stop <id>",
	Short: "尝试优雅停止后台任务",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return stopTask(args[0], false)
	},
}

var taskWorkerCmd = &cobra.Command{
	Use:    "worker <taskID>",
	Short:  "内部使用：后台执行任务",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTaskWorker(args[0])
	},
}

func init() {
	rootCmd.AddCommand(taskCmd)
	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskStopCmd)
	taskCmd.AddCommand(taskKillCmd)
	taskCmd.AddCommand(taskWorkerCmd)
}

func listTasks() error {
	services, err := newCLIServices()
	if err != nil {
		return err
	}
	defer services.Close()

	manager, err := services.TaskManager()
	if err != nil {
		return err
	}

	activeTasks, err := manager.ListActive()
	if err != nil {
		return err
	}

	if len(activeTasks) == 0 {
		fmt.Println("暂无运行中的后台任务")
		return nil
	}

	fmt.Printf("后台任务列表 (共%d个):\n\n", len(activeTasks))
	fmt.Printf("%-4s %-8s %-6s %-19s %-19s %s\n", "ID", "状态", "PID", "创建时间", "开始时间", "命令")
	fmt.Println(strings.Repeat("-", 80))

	for _, task := range activeTasks {
		// 检查进程是否存活
		alive := checkProcessAlive(task.PID)
		if !alive {
			// 如果进程不在了，更新数据库状态
			_ = manager.MarkStopped(task.ID, model.TaskStatusFailed, "进程异常退出 (检测到 PID 失效)")
			// 虽已更新为停止，但在本次列表显示中，可以标记为失效或直接不显示
			// 这里选择显示但标记状态变化，或者简单点，直接修改 task 对象的状态并在 UI 上体现
			task.Status = "lost" // 临时状态用于显示
		}

		started := "-"
		if task.StartedAt != nil {
			started = task.StartedAt.Format("2006-01-02 15:04:05")
		}

		statusStr := task.Status
		if !alive {
			statusStr = "dead/lost"
		}

		fmt.Printf("%-4d %-8s %-6s %-19s %-19s %s\n",
			task.ID,
			statusStr,
			formatPID(task),
			task.CreatedAt.Format("2006-01-02 15:04:05"),
			started,
			formatTaskCommand(task),
		)
		if task.LogFilePath != "" {
			fmt.Printf("      日志: %s\n", task.LogFilePath)
		}
	}

	return nil
}

func checkProcessAlive(pid *int64) bool {
	if pid == nil || *pid <= 0 {
		return false
	}
	process, err := os.FindProcess(int(*pid))
	if err != nil {
		return false
	}
	// 发送信号 0 检查进程是否存在
	err = process.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}

	// 检查特定错误
	if errors.Is(err, os.ErrProcessDone) {
		return false
	}

	// 检查 errno
	if errno, ok := err.(syscall.Errno); ok {
		if errno == syscall.ESRCH {
			return false
		}
	}

	// 其他错误（如 syscall.EPERM 无权限）说明进程存在
	return true
}

func formatPID(task *model.Task) string {
	if task.PID != nil && *task.PID > 0 {
		return strconv.FormatInt(*task.PID, 10)
	}
	return "-"
}

func formatTaskCommand(task *model.Task) string {
	parts := append([]string{task.Command}, task.CommandArgs...)
	return strings.Join(parts, " ")
}

func stopTask(idArg string, force bool) error {
	taskID, err := strconv.Atoi(idArg)
	if err != nil {
		return newExitErrorf(1, "任务ID必须为数字: %s", idArg)
	}

	services, err := newCLIServices()
	if err != nil {
		return err
	}
	defer services.Close()

	manager, err := services.TaskManager()
	if err != nil {
		return err
	}

	task, err := manager.Get(taskID)
	if err != nil {
		return err
	}

	if !task.IsActive() {
		fmt.Printf("任务 #%d 已结束，当前状态：%s\n", task.ID, task.Status)
		return nil
	}

	if task.PID != nil && *task.PID > 0 {
		proc, findErr := os.FindProcess(int(*task.PID))
		if findErr == nil {
			if force {
				_ = proc.Kill()
			} else {
				termErr := proc.Signal(os.Interrupt)
				if termErr != nil && !errors.Is(termErr, os.ErrProcessDone) {
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
		return err
	}

	fmt.Printf("任务 #%d 已%s\n", task.ID, action)
	return nil
}

func runTaskWorker(idArg string) (retErr error) {
	taskID, err := strconv.Atoi(idArg)
	if err != nil {
		return fmt.Errorf("无效的任务ID: %s", idArg)
	}

	services, err := newCLIServices()
	if err != nil {
		return err
	}
	defer services.Close()

	manager, err := services.TaskManager()
	if err != nil {
		return err
	}

	task, err := manager.Get(taskID)
	if err != nil {
		return err
	}
	if task == nil {
		return fmt.Errorf("任务不存在: %d", taskID)
	}

	if task.WorkingDir != "" {
		if err := os.Chdir(task.WorkingDir); err != nil {
			return fmt.Errorf("切换工作目录失败: %w", err)
		}
	}

	if err := manager.MarkRunning(task.ID, os.Getpid()); err != nil {
		if errors.Is(err, tasks.ErrTaskStateChanged) {
			return nil
		}
		return err
	}

	// 确保任务最终会被标记为完成/失败
	var (
		status   = model.TaskStatusFailed
		exitCode = -1
		logPath  = ""
		errMsg   = ""
	)

	defer func() {
		if retErr != nil && errMsg == "" {
			errMsg = retErr.Error()
		}
		_ = manager.MarkCompletion(task.ID, status, exitCode, logPath, errMsg)
	}()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}
	if task.LogDir != "" {
		cfg.LogDir = task.LogDir
	}

	// 预先生成并记录日志路径，以便 tail 命令可以立即查看
	cfg.Command = task.Command
	cfg.CommandArgs = task.CommandArgs
	preLogPath, err := cfg.GetLogFilePath()
	if err != nil {
		return fmt.Errorf("生成日志路径失败: %w", err)
	}
	if err := manager.UpdateLogFilePath(task.ID, preLogPath); err != nil {
		fmt.Fprintf(os.Stderr, "警告: 更新日志路径失败: %v\n", err)
	}

	reg := services.Registry()
	repo := persistence.NewRunRepository(reg)
	statsUpdater := persistence.NewStatsUpdater(reg)

	log, err := logger.New(cfg, repo, statsUpdater)
	if err != nil {
		return fmt.Errorf("创建日志记录器失败: %w", err)
	}
	log.SetLogPath(preLogPath)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	result, path, runErr := log.Run(ctx, task.Command, task.CommandArgs...)
	logPath = path
	if result != nil {
		exitCode = result.ExitCode
	}

	if runErr != nil {
		if ctx.Err() == context.Canceled {
			status = model.TaskStatusStopped
			errMsg = "任务已被终止"
		} else {
			status = model.TaskStatusFailed
			errMsg = runErr.Error()
		}
		return runErr
	}

	status = model.TaskStatusSuccess
	return nil
}
