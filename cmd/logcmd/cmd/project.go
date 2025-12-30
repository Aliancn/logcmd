package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/aliancn/logcmd/internal/model"
	"github.com/aliancn/logcmd/internal/registry"
	"github.com/aliancn/logcmd/internal/template"
	"github.com/spf13/cobra"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "管理已注册的项目",
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有项目",
	RunE: func(cmd *cobra.Command, args []string) error {
		return listProjects()
	},
}

var projectCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "清理不存在的项目",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cleanProjects()
	},
}

var projectDeleteCmd = &cobra.Command{
	Use:   "delete <id|path>",
	Short: "删除指定项目",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return deleteProject(args[0])
	},
}

var projectDeleteForce bool

func init() {
	rootCmd.AddCommand(projectCmd)
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectCleanCmd)
	projectCmd.AddCommand(projectDeleteCmd)

	projectDeleteCmd.Flags().BoolVar(&projectDeleteForce, "force", false, "跳过确认直接删除项目及日志目录")
}

func listProjects() error {
	reg, err := registry.New()
	if err != nil {
		return fmt.Errorf("初始化项目注册表失败: %w", err)
	}
	defer reg.Close()

	entries, err := reg.List()
	if err != nil {
		return fmt.Errorf("列出项目失败: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("没有已注册的项目")
		return nil
	}

	fmt.Printf("已注册的项目 (共%d个):\n\n", len(entries))
	fmt.Printf("%-5s %-20s %-45s %-19s %-8s %-10s %s\n", "ID", "项目名称", "路径", "最后执行", "成功率", "命令数", "存在")
	fmt.Println(strings.Repeat("-", 130))
	for _, entry := range entries {
		exists := "✓"
		if _, err := os.Stat(entry.Path); os.IsNotExist(err) {
			exists = "✗"
		}
		lastRun := "-"
		if entry.LastCommandTime.Valid {
			lastRun = entry.LastCommandTime.Time.Format("2006-01-02 15:04:05")
		}
		name := entry.Name
		if name == "" {
			name = template.GetProjectName(entry.Path)
		}
		successRate := fmt.Sprintf("%.1f%%", entry.GetSuccessRate())

		fmt.Printf("%-5d %-20s %-45s %-19s %-8s %-10d %s\n",
			entry.ID, name, entry.Path, lastRun, successRate, entry.TotalCommands, exists)
	}

	return nil
}

func cleanProjects() error {
	reg, err := registry.New()
	if err != nil {
		return fmt.Errorf("初始化项目注册表失败: %w", err)
	}
	defer reg.Close()

	if err := reg.CheckAndCleanup(); err != nil {
		return fmt.Errorf("清理失败: %w", err)
	}

	fmt.Println("清理完成")
	return nil
}

func deleteProject(target string) error {
	reg, err := registry.New()
	if err != nil {
		return fmt.Errorf("初始化项目注册表失败: %w", err)
	}
	defer reg.Close()

	project, err := reg.Get(target)
	if err != nil {
		return fmt.Errorf("查询项目失败: %w", err)
	}

	if !projectDeleteForce {
		confirmed, confirmErr := confirmProjectDeletion(project)
		if confirmErr != nil {
			return fmt.Errorf("读取用户输入失败: %w", confirmErr)
		}
		if !confirmed {
			fmt.Println("已取消删除操作")
			return nil
		}
	}

	if err := reg.Delete(target); err != nil {
		return fmt.Errorf("删除项目失败: %w", err)
	}

	if err := os.RemoveAll(project.Path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除项目日志目录失败: %w", err)
	}

	fmt.Printf("成功删除项目: %s\n", target)
	return nil
}

func confirmProjectDeletion(project *model.Project) (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	displayName := project.Name
	if strings.TrimSpace(displayName) == "" {
		displayName = template.GetProjectName(project.Path)
	}

	fmt.Printf("⚠️ 即将删除项目 ID=%d 名称=\"%s\"\n", project.ID, displayName)
	fmt.Printf("日志目录: %s\n", project.Path)
	fmt.Print("请输入 yes/确认 继续删除，其他输入取消: ")

	input, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	normalized := strings.ToLower(strings.TrimSpace(input))
	return normalized == "yes" || normalized == "确认" || normalized == "y", nil
}
