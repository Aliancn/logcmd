package services_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aliancn/logcmd/internal/model"
	"github.com/aliancn/logcmd/internal/registry"
	"github.com/aliancn/logcmd/internal/services"
	"github.com/aliancn/logcmd/internal/stats"
)

func TestStatsServiceStatsForPathWithoutRegistry(t *testing.T) {
	projectDir := filepath.Join(t.TempDir(), "statspath")
	logDir := filepath.Join(projectDir, ".logcmd")
	createSampleLogFile(t, logDir)

	svc := services.NewStatsService(nil)
	report, err := svc.StatsForPath(context.Background(), logDir)
	if err != nil {
		t.Fatalf("StatsForPath() 失败: %v", err)
	}

	if report == nil {
		t.Fatal("report 不应为 nil")
	}

	if report.TotalCommands != 1 {
		t.Fatalf("TotalCommands = %d, want 1", report.TotalCommands)
	}

	if report.Source != stats.SourceLogFiles {
		t.Errorf("Source = %s, want %s", report.Source, stats.SourceLogFiles)
	}

	expectedName := filepath.Base(projectDir)
	if report.ProjectName != expectedName {
		t.Errorf("ProjectName = %s, want %s", report.ProjectName, expectedName)
	}
}

func TestStatsServiceStatsForProjectUsesCache(t *testing.T) {
	svc, reg, project, logDir := setupStatsServiceWithRegistry(t)
	insertCommandHistory(t, reg, project.ID, logDir)

	report, err := svc.StatsForProject(context.Background(), project)
	if err != nil {
		t.Fatalf("StatsForProject() 失败: %v", err)
	}

	if report == nil {
		t.Fatal("report 不应为 nil")
	}

	if report.Source != stats.SourceDatabase {
		t.Errorf("Source = %s, want %s", report.Source, stats.SourceDatabase)
	}

	if report.TotalCommands != 1 || report.SuccessCommands != 1 {
		t.Fatalf("统计结果不正确: %+v", report)
	}

	if report.ProjectName == "" {
		t.Fatal("ProjectName 不应为空")
	}
}

func TestStatsServiceStatsForProjectFallsBackToLogs(t *testing.T) {
	projectDir := filepath.Join(t.TempDir(), "fallback")
	logDir := filepath.Join(projectDir, ".logcmd")
	createSampleLogFile(t, logDir)

	project := &model.Project{
		Path: logDir,
		Name: "Manual Project",
	}

	svc := services.NewStatsService(nil)
	report, err := svc.StatsForProject(context.Background(), project)
	if err != nil {
		t.Fatalf("StatsForProject() 失败: %v", err)
	}

	if report.Source != stats.SourceLogFiles {
		t.Errorf("Source = %s, want %s", report.Source, stats.SourceLogFiles)
	}

	if report.ProjectName != project.Name {
		t.Errorf("ProjectName = %s, want %s", report.ProjectName, project.Name)
	}

	if report.TotalCommands != 1 {
		t.Fatalf("TotalCommands = %d, want 1", report.TotalCommands)
	}
}

func TestStatsServiceProjectByPathRegistersDirectory(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	logDir := filepath.Join(tmpHome, "project", ".logcmd")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("创建日志目录失败: %v", err)
	}

	reg, err := registry.New()
	if err != nil {
		t.Fatalf("创建 Registry 失败: %v", err)
	}
	t.Cleanup(func() {
		reg.Close()
	})

	svc := services.NewStatsService(reg)
	ctx := context.Background()

	project, err := svc.ProjectByPath(ctx, logDir)
	if err != nil {
		t.Fatalf("ProjectByPath() 失败: %v", err)
	}

	if project == nil {
		t.Fatal("project 不应为 nil")
	}

	stored, err := reg.Get(logDir)
	if err != nil {
		t.Fatalf("Get() 失败: %v", err)
	}

	if stored.ID != project.ID {
		t.Fatalf("注册的项目ID不匹配: got %d want %d", project.ID, stored.ID)
	}
}

func createSampleLogFile(t *testing.T, logDir string) {
	t.Helper()

	dateDir := filepath.Join(logDir, "2024-01-15")
	if err := os.MkdirAll(dateDir, 0755); err != nil {
		t.Fatalf("创建日志目录失败: %v", err)
	}

	content := `
################################################################################
# LogCmd - 命令执行日志
# 时间: 2024-01-15 10:00:00
# 命令: echo [test]
################################################################################

test output

================================================================================
命令: echo [test]
开始时间: 2024-01-15 10:00:00
结束时间: 2024-01-15 10:00:01
执行时长: 1s
退出码: 0
执行状态: 成功
================================================================================
`
	logPath := filepath.Join(dateDir, "sample.log")
	if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
		t.Fatalf("写入日志失败: %v", err)
	}
}

func setupStatsServiceWithRegistry(t *testing.T) (*services.StatsService, *registry.Registry, *model.Project, string) {
	t.Helper()

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	reg, err := registry.New()
	if err != nil {
		t.Fatalf("创建 Registry 失败: %v", err)
	}
	t.Cleanup(func() {
		reg.Close()
	})

	projectDir := filepath.Join(tmpHome, "stats-cache-project")
	logDir := filepath.Join(projectDir, ".logcmd")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("创建日志目录失败: %v", err)
	}

	project, err := reg.Register(logDir)
	if err != nil {
		t.Fatalf("注册项目失败: %v", err)
	}

	return services.NewStatsService(reg), reg, project, logDir
}

func insertCommandHistory(t *testing.T, reg *registry.Registry, projectID int, logDir string) {
	t.Helper()

	db := reg.GetDB()
	now := time.Now()
	logDate := now.Format("2006-01-02")

	logPath := filepath.Join(logDir, logDate, "cache.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		t.Fatalf("创建日志目录失败: %v", err)
	}
	if err := os.WriteFile(logPath, []byte("cache log"), 0644); err != nil {
		t.Fatalf("写入日志失败: %v", err)
	}

	_, err := db.Exec(`
		INSERT INTO command_history (
			project_id, command, command_name, command_args,
			start_time, end_time, duration_ms, exit_code, status,
			log_file_path, log_date, stdout_preview, stderr_preview,
			has_error, working_directory, environment_info, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		projectID,
		"echo cache",
		"echo",
		"cache",
		now,
		now.Add(time.Second),
		int64(1000),
		0,
		"success",
		logPath,
		logDate,
		"",
		"",
		0,
		logDir,
		"",
		now,
	)
	if err != nil {
		t.Fatalf("插入 command_history 失败: %v", err)
	}
}
