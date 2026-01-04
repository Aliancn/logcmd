package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/spf13/cobra"
)

var (
	tailFollow bool
	tailLines  int
)

var tailCmd = &cobra.Command{
	Use:   "tail <taskID>",
	Short: "查看任务日志",
	Long:  "查看指定任务的日志输出。支持查看最后几行以及实时跟踪日志。",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTail(args[0])
	},
}

func init() {
	tailCmd.Flags().BoolVarP(&tailFollow, "follow", "f", false, "实时跟踪日志输出")
	tailCmd.Flags().IntVarP(&tailLines, "lines", "n", 20, "显示最后几行日志")
	rootCmd.AddCommand(tailCmd)
}

func runTail(idArg string) error {
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

	if task.LogFilePath == "" {
		return fmt.Errorf("任务 #%d 尚未生成日志文件 (状态: %s)", task.ID, task.Status)
	}

	if _, err := os.Stat(task.LogFilePath); os.IsNotExist(err) {
		return fmt.Errorf("日志文件不存在: %s", task.LogFilePath)
	}

	tailArgs := []string{"-n", strconv.Itoa(tailLines)}
	if tailFollow {
		tailArgs = append(tailArgs, "-f")
	}
	tailArgs = append(tailArgs, task.LogFilePath)

	// 使用系统的 tail 命令
	c := exec.Command("tail", tailArgs...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin

	// 处理中断信号，确保退出时不影响后台任务
	// 实际上，logcmd tail 只是一个查看器，kill 掉它不会影响产生日志的 task 进程。
	// 这里直接运行 tail 命令即可，用户 Ctrl+C 会终止 logcmd tail 和其子进程 tail。

	if err := c.Run(); err != nil {
		// 如果是用户中断 (Ctrl+C)，通常返回 exit status 130 或类似，视作正常退出
		if exitErr, ok := err.(*exec.ExitError); ok {
			// 忽略中断信号导致的错误
			if exitErr.ExitCode() == 130 { // SIGINT usually results in 130
				return nil
			}
			// tail -f 被 kill 也是正常
			if tailFollow {
				return nil
			}
		}
		return fmt.Errorf("执行 tail 命令失败: %w", err)
	}

	return nil
}
