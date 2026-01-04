package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/aliancn/logcmd/internal/config"
	"github.com/aliancn/logcmd/internal/logger"
	"github.com/aliancn/logcmd/internal/model"
	"github.com/aliancn/logcmd/internal/persistence"
	"github.com/spf13/cobra"
)

var (
	runDetached bool
)

var runCmd = &cobra.Command{
	Use:   "run <command> [args...]",
	Short: "执行命令并记录日志",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return newExitErrorf(1, "run 子命令需要指定要执行的命令")
		}
		return runCommand(cmd, args)
	},
}

func init() {
	runCmd.Flags().SetInterspersed(false)
	runCmd.Flags().BoolVarP(&runDetached, "detached", "d", false, "后台运行命令并交由 task 管理")
	rootCmd.AddCommand(runCmd)
}

func runCommand(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}
	if logDirFlag != "" {
		cfg.LogDir = logDirFlag
	}

	services, err := newCLIServices()
	if err != nil {
		return err
	}
	defer services.Close()

	if runDetached {
		return startDetachedTask(cfg, services, args)
	}

	reg := services.Registry()
	repo := persistence.NewRunRepository(reg)
	statsUpdater := persistence.NewStatsUpdater(reg)

	log, err := logger.New(cfg, repo, statsUpdater)
	if err != nil {
		return fmt.Errorf("创建日志记录器失败: %w", err)
	}

	ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if _, _, err := log.Run(ctx, args[0], args[1:]...); err != nil {
		if ctx.Err() == context.Canceled {
			fmt.Println("\n命令已由用户中断")
			return newExitError(nil, 130)
		}
		return fmt.Errorf("执行失败: %w", err)
	}

	return nil
}

func startDetachedTask(cfg *config.Config, services *cliServices, args []string) error {
	manager, err := services.TaskManager()
	if err != nil {
		return err
	}

	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取工作目录失败: %w", err)
	}

	task := &model.Task{
		Command:     args[0],
		CommandArgs: args[1:],
		WorkingDir:  workingDir,
		LogDir:      cfg.LogDir,
		Status:      model.TaskStatusPending,
	}

	task, err = manager.Create(task)
	if err != nil {
		return fmt.Errorf("创建后台任务失败: %w", err)
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取可执行文件失败: %w", err)
	}

	workerCmd := exec.Command(exe, "task", "worker", strconv.Itoa(task.ID))
	// 使用 nil (默认值) 将 stdout/stderr 连接到 /dev/null
	// 不要使用 io.Discard，因为它会创建 pipe，导致父进程退出后子进程写入 stdout 时收到 SIGPIPE 并终止
	workerCmd.Stdout = nil
	workerCmd.Stderr = nil
	workerCmd.Stdin = nil
	workerCmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	if err := workerCmd.Start(); err != nil {
		_ = manager.MarkStopped(task.ID, model.TaskStatusFailed, fmt.Sprintf("启动失败: %v", err))
		return fmt.Errorf("启动后台任务失败: %w", err)
	}

	if err := manager.UpdatePID(task.ID, workerCmd.Process.Pid); err != nil {
		fmt.Fprintf(os.Stderr, "更新任务 PID 失败: %v\n", err)
	}

	if err := workerCmd.Process.Release(); err != nil {
		fmt.Fprintf(os.Stderr, "释放后台进程失败: %v\n", err)
	}

	fmt.Printf("后台任务 #%d 已启动: %s\n", task.ID, strings.Join(args, " "))
	fmt.Printf("日志目录: %s\n", cfg.LogDir)

	return nil
}
