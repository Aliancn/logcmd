package cmd

import (
	"fmt"
	"strings"

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
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return newExitErrorf(1, "请使用 \"logcmd run %s\" 执行命令", strings.Join(args, " "))
	},
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
