package cmd

import (
	"fmt"

	"github.com/aliancn/logcmd/internal/template"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "配置日志相关选项",
}

var configLogNameCmd = &cobra.Command{
	Use:   "logname",
	Short: "交互式配置日志命名模板",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := template.ConfigureInteractive(); err != nil {
			return fmt.Errorf("配置模板失败: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configLogNameCmd)
}
