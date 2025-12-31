package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/aliancn/logcmd/internal/config"
	"github.com/aliancn/logcmd/internal/logger"
	"github.com/aliancn/logcmd/internal/persistence"
	"github.com/aliancn/logcmd/internal/registry"
	"github.com/spf13/cobra"
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
	rootCmd.AddCommand(runCmd)
}

func runCommand(cmd *cobra.Command, args []string) error {
	cfg := config.DefaultConfig()
	if logDirFlag != "" {
		cfg.LogDir = logDirFlag
	}

	reg, err := registry.New()
	if err != nil {
		return fmt.Errorf("初始化项目注册表失败: %w", err)
	}
	defer reg.Close()

	repo := persistence.NewRunRepository(reg)
	statsUpdater := persistence.NewStatsUpdater(reg)

	log, err := logger.New(cfg, repo, statsUpdater)
	if err != nil {
		return fmt.Errorf("创建日志记录器失败: %w", err)
	}

	ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := log.Run(ctx, args[0], args[1:]...); err != nil {
		if ctx.Err() == context.Canceled {
			fmt.Println("\n命令已由用户中断")
			return newExitError(nil, 130)
		}
		return fmt.Errorf("执行失败: %w", err)
	}

	return nil
}
