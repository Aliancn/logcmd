package persistence_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/aliancn/logcmd/internal/executor"
	"github.com/aliancn/logcmd/internal/persistence"
)

func TestNewRunRepositoryRequiresRegistry(t *testing.T) {
	if repo := persistence.NewRunRepository(nil); repo != nil {
		t.Fatal("传入 nil registry 时应该返回 nil")
	}
}

func TestRunRepositoryRegisterProject(t *testing.T) {
	reg := newTestRegistry(t)
	repo := persistence.NewRunRepository(reg)
	if repo == nil {
		t.Fatal("NewRunRepository() 不应返回 nil")
	}

	projectRoot := filepath.Join(t.TempDir(), "manual-project")
	logDir := filepath.Join(projectRoot, ".logcmd")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatalf("创建日志目录失败: %v", err)
	}

	project, err := repo.RegisterProject(logDir)
	if err != nil {
		t.Fatalf("RegisterProject() 失败: %v", err)
	}

	wantPath, err := filepath.Abs(logDir)
	if err != nil {
		t.Fatalf("获取绝对路径失败: %v", err)
	}
	if project.Path != wantPath {
		t.Fatalf("注册路径不匹配: got %s want %s", project.Path, wantPath)
	}
}

func TestRunRepositoryRecordRunPersistsHistoryAndCache(t *testing.T) {
	reg := newTestRegistry(t)
	repo := persistence.NewRunRepository(reg)
	project, logDir := registerTestProject(t, reg)

	startTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	endTime := startTime.Add(2 * time.Second)
	result := &executor.Result{
		Command:   "echo",
		Args:      []string{"hello", "world"},
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  endTime.Sub(startTime),
		ExitCode:  0,
		Success:   true,
	}

	logDate := startTime.Format("2006-01-02")
	logDirForDate := filepath.Join(logDir, logDate)
	if err := os.MkdirAll(logDirForDate, 0o755); err != nil {
		t.Fatalf("创建日期目录失败: %v", err)
	}
	logPath := filepath.Join(logDirForDate, "logcmd.log")
	if err := os.WriteFile(logPath, []byte("sample"), 0o644); err != nil {
		t.Fatalf("写入日志失败: %v", err)
	}

	if err := repo.RecordRun(project, result, logPath); err != nil {
		t.Fatalf("RecordRun() 失败: %v", err)
	}

	db := reg.GetDB()
	var (
		command    string
		argsJSON   string
		storedPath string
		storedDate string
		workingDir string
		hasError   bool
	)
	row := db.QueryRow(`
		SELECT command, command_args, log_file_path, log_date, working_directory, has_error
		FROM command_history WHERE project_id = ?`, project.ID)
	if err := row.Scan(&command, &argsJSON, &storedPath, &storedDate, &workingDir, &hasError); err != nil {
		t.Fatalf("读取 command_history 失败: %v", err)
	}

	if command != "echo hello world" {
		t.Fatalf("command = %s, want echo hello world", command)
	}

	var args []string
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		t.Fatalf("解析 args JSON 失败: %v", err)
	}
	if !reflect.DeepEqual(args, result.Args) {
		t.Fatalf("命令参数不匹配: got %v want %v", args, result.Args)
	}

	if storedPath != logPath {
		t.Fatalf("log_file_path = %s, want %s", storedPath, logPath)
	}

	if storedDate != logDate {
		t.Fatalf("log_date = %s, want %s", storedDate, logDate)
	}

	if hasError {
		t.Fatal("has_error 应为 false")
	}

	if wd, err := os.Getwd(); err == nil {
		if workingDir != wd {
			t.Fatalf("working_directory = %s, want %s", workingDir, wd)
		}
	}

	cacheRow := db.QueryRow(`
		SELECT total_commands, success_commands, failed_commands
		FROM project_stats_cache WHERE project_id = ? AND stat_date = ?`, project.ID, logDate)

	var total, success, failed int
	if err := cacheRow.Scan(&total, &success, &failed); err != nil {
		t.Fatalf("读取 project_stats_cache 失败: %v", err)
	}

	if total != 1 || success != 1 || failed != 0 {
		t.Fatalf("统计缓存不正确: total=%d success=%d failed=%d", total, success, failed)
	}
}

func TestRunRepositoryRecordRunRequiresDependencies(t *testing.T) {
	var repo *persistence.RunRepository
	err := repo.RecordRun(nil, nil, "")
	if err == nil {
		t.Fatal("未初始化的 RunRepository 应该返回错误")
	}
}
