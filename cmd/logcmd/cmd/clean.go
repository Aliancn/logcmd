package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	cleanDays  int
	cleanKeep  int
	cleanForce bool
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "清理旧日志文件和历史记录",
	Long:  `根据条件清理过期的日志文件和数据库记录。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runClean()
	},
}

func init() {
	cleanCmd.Flags().IntVar(&cleanDays, "days", 0, "删除超过指定天数的日志")
	cleanCmd.Flags().IntVar(&cleanKeep, "keep", 0, "仅保留最近的 N 条记录")
	cleanCmd.Flags().BoolVarP(&cleanForce, "force", "f", false, "跳过确认")
	rootCmd.AddCommand(cleanCmd)
}

func runClean() error {
	if cleanDays <= 0 && cleanKeep <= 0 {
		return fmt.Errorf("请指定清理条件: --days 或 --keep")
	}

	services, err := newCLIServices()
	if err != nil {
		return err
	}
	defer services.Close()

	histMgr := services.HistoryManager()

	// Confirmation
	if !cleanForce {
		fmt.Println("⚠️  警告: 此操作将永久删除日志文件和数据库记录。")
		if cleanDays > 0 {
			fmt.Printf("- 删除 %d 天前的记录\n", cleanDays)
		}
		if cleanKeep > 0 {
			fmt.Printf("- 仅保留最近 %d 条记录\n", cleanKeep)
		}
		fmt.Print("确认执行? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(input)) != "y" {
			fmt.Println("操作已取消")
			return nil
		}
	}

	if cleanDays > 0 {
		if err := histMgr.DeleteOldRecords(cleanDays); err != nil {
			return err
		}
	}

	if cleanKeep > 0 {
		if err := histMgr.DeleteExcessRecords(cleanKeep); err != nil {
			return err
		}
	}

	return nil
}
