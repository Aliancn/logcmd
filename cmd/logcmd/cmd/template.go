package cmd

import (
	"fmt"

	"github.com/aliancn/logcmd/internal/template"
	"github.com/spf13/cobra"
)

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "配置模板",
}

var templateLogNameCmd = &cobra.Command{
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
	rootCmd.AddCommand(templateCmd)
	templateCmd.AddCommand(templateLogNameCmd)
}
