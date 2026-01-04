package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/aliancn/logcmd/internal/config"
	"github.com/aliancn/logcmd/internal/model"
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
		return runStats(cmd)
	},
}

type statsJob struct {
	index int
	entry *model.Project
}

type statsJobResult struct {
	index  int
	entry  *model.Project
	report *stats.Stats
	err    error
}

func init() {
	rootCmd.AddCommand(statsCmd)

	statsCmd.Flags().BoolVar(&statsAllFlag, "all", false, "统计所有已注册项目")
	statsCmd.Flags().StringVar(&statsDirFlag, "dir", "", "日志目录路径")
}

func runStats(cmd *cobra.Command) error {
	cfg := config.DefaultConfig()
	logDirPath := filepath.Clean(cfg.LogDir)
	if statsDirFlag != "" {
		logDirPath = filepath.Clean(statsDirFlag)
	}

	ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cliServices, svcErr := newCLIServices()
	var statsSvc *services.StatsService
	if svcErr == nil {
		statsSvc = services.NewStatsService(cliServices.Registry())
		defer cliServices.Close()
	}

	if statsAllFlag {
		if statsSvc == nil || cliServices == nil {
			if svcErr != nil {
				return svcErr
			}
			return fmt.Errorf("统计服务未初始化")
		}
		return analyzeAllProjects(ctx, cliServices, statsSvc)
	}

	if statsSvc == nil {
		fmt.Fprintf(os.Stderr, "警告: 统计服务未初始化，直接扫描日志目录\n")
		return analyzeLogDir(ctx, logDirPath)
	}

	report, statErr := statsSvc.StatsForPath(ctx, logDirPath)
	if statErr != nil {
		return fmt.Errorf("统计分析失败: %w", statErr)
	}
	stats.PrintStats(report)
	return nil
}

func analyzeAllProjects(ctx context.Context, cliSvc *cliServices, svc *services.StatsService) error {
	reg := cliSvc.Registry()
	entries, err := reg.List()
	if err != nil {
		return fmt.Errorf("获取项目列表失败: %w", err)
	}

	if len(entries) == 0 {
		return fmt.Errorf("错误: 没有已注册的项目")
	}

	fmt.Printf("正在统计 %d 个项目...\n\n", len(entries))

	jobs := make(chan statsJob)
	resultsCh := make(chan statsJobResult, workerCount(len(entries)))
	var wg sync.WaitGroup

	workerTotal := workerCount(len(entries))
	for i := 0; i < workerTotal; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				if ctx.Err() != nil {
					resultsCh <- statsJobResult{index: job.index, entry: job.entry, err: ctx.Err()}
					continue
				}
				report, err := svc.StatsForProject(ctx, job.entry)
				resultsCh <- statsJobResult{index: job.index, entry: job.entry, report: report, err: err}
			}
		}()
	}

	scheduledEntries := make([]*model.Project, 0, len(entries))

dispatchLoop:
	for _, entry := range entries {
		if _, statErr := os.Stat(entry.Path); os.IsNotExist(statErr) {
			fmt.Printf("[%d/%d] 跳过（已删除）: %s\n", len(scheduledEntries)+1, len(entries), entry.Path)
			if err := reg.Delete(fmt.Sprintf("%d", entry.ID)); err != nil {
				fmt.Fprintf(os.Stderr, "  警告: 删除无效项目失败: %v\n", err)
			}
			continue
		}

		jobIndex := len(scheduledEntries)
		scheduledEntries = append(scheduledEntries, entry)

		select {
		case <-ctx.Done():
			scheduledEntries = scheduledEntries[:jobIndex]
			break dispatchLoop
		case jobs <- statsJob{index: jobIndex, entry: entry}:
		}
	}

	close(jobs)
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	results := make([]statsJobResult, len(scheduledEntries))
	for res := range resultsCh {
		results[res.index] = res
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	for i, entry := range scheduledEntries {
		fmt.Printf("[%d/%d] 统计: %s\n", i+1, len(entries), entry.Path)
		result := results[i]
		if result.err != nil {
			fmt.Fprintf(os.Stderr, "  警告: 统计失败: %v\n", result.err)
			continue
		}
		stats.PrintStats(result.report)
		fmt.Println()
		reg.UpdateLastChecked(fmt.Sprintf("%d", entry.ID))
	}

	fmt.Println("统计完成")
	return nil
}

func analyzeLogDir(ctx context.Context, logDirPath string) error {
	analyzer := stats.New(logDirPath)
	statistics, err := analyzer.Analyze(ctx)
	if err != nil {
		return fmt.Errorf("统计分析失败: %w", err)
	}
	statistics.ProjectName = template.GetProjectName(logDirPath)

	stats.PrintStats(statistics)
	return nil
}
