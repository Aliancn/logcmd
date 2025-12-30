package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aliancn/logcmd/internal/config"
	"github.com/aliancn/logcmd/internal/registry"
	"github.com/aliancn/logcmd/internal/services"
	"github.com/aliancn/logcmd/internal/stats"
	"github.com/aliancn/logcmd/internal/template"
	"github.com/spf13/cobra"
)

var (
	statsAllFlag bool
	statsDirFlag string
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "统计日志数据",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStats()
	},
}

func init() {
	rootCmd.AddCommand(statsCmd)

	statsCmd.Flags().BoolVar(&statsAllFlag, "all", false, "统计所有已注册项目")
	statsCmd.Flags().StringVar(&statsDirFlag, "dir", "", "日志目录路径")
}

func runStats() error {
	cfg := config.DefaultConfig()
	logDirPath := filepath.Clean(cfg.LogDir)
	if statsDirFlag != "" {
		logDirPath = filepath.Clean(statsDirFlag)
	}

	reg, err := registry.New()
	var svc *services.StatsService
	if err == nil {
		svc = services.NewStatsService(reg)
		defer reg.Close()
	}

	if statsAllFlag {
		if svc == nil || reg == nil {
			return fmt.Errorf("初始化项目注册表失败: %w", err)
		}
		return analyzeAllProjects(reg, svc)
	}

	if svc == nil {
		fmt.Fprintf(os.Stderr, "警告: 统计服务未初始化，直接扫描日志目录\n")
		return analyzeLogDir(logDirPath)
	}

	report, statErr := svc.StatsForPath(logDirPath)
	if statErr != nil {
		return fmt.Errorf("统计分析失败: %w", statErr)
	}
	stats.PrintStats(report)
	return nil
}

func analyzeAllProjects(reg *registry.Registry, svc *services.StatsService) error {
	entries, err := reg.List()
	if err != nil {
		return fmt.Errorf("获取项目列表失败: %w", err)
	}

	if len(entries) == 0 {
		return fmt.Errorf("错误: 没有已注册的项目")
	}

	fmt.Printf("正在统计 %d 个项目...\n\n", len(entries))

	for i, entry := range entries {
		if _, err := os.Stat(entry.Path); os.IsNotExist(err) {
			fmt.Printf("[%d/%d] 跳过（已删除）: %s\n", i+1, len(entries), entry.Path)
			if err := reg.Delete(fmt.Sprintf("%d", entry.ID)); err != nil {
				fmt.Fprintf(os.Stderr, "  警告: 删除无效项目失败: %v\n", err)
			}
			continue
		}

		fmt.Printf("[%d/%d] 统计: %s\n", i+1, len(entries), entry.Path)

		report, statErr := svc.StatsForProject(entry)
		if statErr != nil {
			fmt.Fprintf(os.Stderr, "  警告: 统计失败: %v\n", statErr)
			continue
		}
		stats.PrintStats(report)
		fmt.Println()
		reg.UpdateLastChecked(fmt.Sprintf("%d", entry.ID))
	}

	fmt.Println("统计完成")
	return nil
}

func analyzeLogDir(logDirPath string) error {
	analyzer := stats.New(logDirPath)
	statistics, err := analyzer.Analyze()
	if err != nil {
		return fmt.Errorf("统计分析失败: %w", err)
	}
	statistics.ProjectName = template.GetProjectName(logDirPath)

	stats.PrintStats(statistics)
	return nil
}
