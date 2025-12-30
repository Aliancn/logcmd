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

const version = "1.0.0"

var (
	logDirFlag  string
	versionFlag bool
)

var rootCmd = &cobra.Command{
	Use:           "logcmd",
	Short:         "高性能命令日志记录工具",
	Long:          "LogCmd - 执行命令并记录日志、搜索与统计的高性能 CLI 工具。",
	SilenceUsage:  true,
	SilenceErrors: true,
	Args:          cobra.ArbitraryArgs,
	RunE:          runCommand,
}

// Execute 启动 CLI。
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&logDirFlag, "dir", "", "日志目录路径（默认：自动选择）")
	rootCmd.PersistentFlags().BoolVar(&versionFlag, "version", false, "显示版本信息")
	rootCmd.PersistentFlags().SetInterspersed(false)
	rootCmd.Flags().SetInterspersed(false)

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if versionFlag {
			fmt.Printf("logcmd version %s\n", version)
			return newExitError(nil, 0)
		}
		return nil
	}
}

func runCommand(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		_ = cmd.Help()
		return nil
	}

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
