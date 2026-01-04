package cmd

import (
	"context"

	"github.com/aliancn/logcmd/internal/ui"
	"github.com/spf13/cobra"
)

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "启动交互式终端界面",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUI(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(uiCmd)
}

func runUI(ctx context.Context) error {
	services, err := newCLIServices()
	if err != nil {
		return err
	}
	defer services.Close()

	if ctx == nil {
		ctx = context.Background()
	}

	return ui.Start(ctx, services.Registry())
}
