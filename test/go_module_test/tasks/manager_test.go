package tasks_test

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/aliancn/logcmd/internal/migration"
	"github.com/aliancn/logcmd/internal/model"
	"github.com/aliancn/logcmd/internal/tasks"
)

func setupTaskManager(t *testing.T) (*tasks.Manager, *sql.DB) {
	t.Helper()

	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)

	dbPath := filepath.Join(tmpDir, "tasks.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("打开数据库失败: %v", err)
	}

	migrator := migration.NewMigration(db)
	if err := migrator.Migrate(); err != nil {
		db.Close()
		t.Fatalf("数据库迁移失败: %v", err)
	}

	manager := tasks.NewManager(db)
	if manager == nil {
		db.Close()
		t.Fatal("tasks.NewManager() 返回 nil")
	}

	return manager, db
}

func createTask(t *testing.T, manager *tasks.Manager, workingDir string, logDir string, command string, args []string) *model.Task {
	t.Helper()
	task := &model.Task{
		Command:     command,
		CommandArgs: args,
		WorkingDir:  workingDir,
		LogDir:      logDir,
	}
	created, err := manager.Create(task)
	if err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}
	return created
}

func TestManager_CreateAndGet(t *testing.T) {
	manager, db := setupTaskManager(t)
	defer db.Close()

	workingDir := t.TempDir()
	logDir := t.TempDir()

	created := createTask(t, manager, workingDir, logDir, "echo", []string{"hello"})
	if created.ID == 0 {
		t.Fatal("新建任务的 ID 不应为 0")
	}
	if created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
		t.Fatal("任务的时间戳应该被设置")
	}
	if created.ArgsJSON == "" {
		t.Fatal("任务应该序列化命令参数")
	}

	loaded, err := manager.Get(created.ID)
	if err != nil {
		t.Fatalf("Get() 失败: %v", err)
	}

	if loaded.Command != "echo" {
		t.Errorf("Command = %s, want echo", loaded.Command)
	}
	if len(loaded.CommandArgs) != 1 || loaded.CommandArgs[0] != "hello" {
		t.Errorf("CommandArgs = %v, want [hello]", loaded.CommandArgs)
	}
	if loaded.WorkingDir != workingDir {
		t.Errorf("WorkingDir = %s, want %s", loaded.WorkingDir, workingDir)
	}
	if loaded.LogDir != logDir {
		t.Errorf("LogDir = %s, want %s", loaded.LogDir, logDir)
	}
	if loaded.Status != model.TaskStatusPending {
		t.Errorf("Status = %s, want %s", loaded.Status, model.TaskStatusPending)
	}
}

func TestManager_ListActive(t *testing.T) {
	manager, db := setupTaskManager(t)
	defer db.Close()

	workingDir := t.TempDir()
	logDir := t.TempDir()

	pending := createTask(t, manager, workingDir, logDir, "echo", []string{"pending"})
	running := createTask(t, manager, workingDir, logDir, "echo", []string{"running"})
	completed := createTask(t, manager, workingDir, logDir, "echo", []string{"done"})

	if err := manager.MarkRunning(running.ID, 2001); err != nil {
		t.Fatalf("MarkRunning() 失败: %v", err)
	}
	if err := manager.MarkCompletion(completed.ID, model.TaskStatusSuccess, 0, "/tmp/log.log", ""); err != nil {
		t.Fatalf("MarkCompletion() 失败: %v", err)
	}

	list, err := manager.ListActive()
	if err != nil {
		t.Fatalf("ListActive() 失败: %v", err)
	}

	if len(list) != 2 {
		t.Fatalf("ListActive() 返回 %d 条记录, want 2", len(list))
	}

	if list[0].ID != pending.ID {
		t.Errorf("第一条任务 ID = %d, want %d", list[0].ID, pending.ID)
	}
	if list[1].ID != running.ID {
		t.Errorf("第二条任务 ID = %d, want %d", list[1].ID, running.ID)
	}
	if list[1].Status != model.TaskStatusRunning {
		t.Errorf("运行中任务状态 = %s, want %s", list[1].Status, model.TaskStatusRunning)
	}
}

func TestManager_UpdatePID(t *testing.T) {
	manager, db := setupTaskManager(t)
	defer db.Close()

	workingDir := t.TempDir()
	logDir := t.TempDir()

	task := createTask(t, manager, workingDir, logDir, "sleep", []string{"1"})
	if err := manager.UpdatePID(task.ID, 4321); err != nil {
		t.Fatalf("UpdatePID() 失败: %v", err)
	}

	loaded, err := manager.Get(task.ID)
	if err != nil {
		t.Fatalf("Get() 失败: %v", err)
	}

	if loaded.PID == nil || *loaded.PID != 4321 {
		t.Errorf("PID 更新失败: %+v", loaded.PID)
	}
}

func TestManager_MarkRunningAndCompletion(t *testing.T) {
	manager, db := setupTaskManager(t)
	defer db.Close()

	workingDir := t.TempDir()
	logDir := t.TempDir()

	task := createTask(t, manager, workingDir, logDir, "echo", []string{"task"})

	if err := manager.MarkRunning(task.ID, 1234); err != nil {
		t.Fatalf("MarkRunning() 失败: %v", err)
	}

	running, err := manager.Get(task.ID)
	if err != nil {
		t.Fatalf("Get() 失败: %v", err)
	}

	if running.Status != model.TaskStatusRunning {
		t.Errorf("运行中状态 = %s, want %s", running.Status, model.TaskStatusRunning)
	}
	if running.StartedAt == nil {
		t.Error("运行中任务的 StartedAt 应该被设置")
	}
	if running.PID == nil || *running.PID != 1234 {
		t.Errorf("PID = %+v, want 1234", running.PID)
	}

	if err := manager.MarkCompletion(task.ID, "", 0, "/tmp/task.log", ""); err != nil {
		t.Fatalf("MarkCompletion() 失败: %v", err)
	}

	completed, err := manager.Get(task.ID)
	if err != nil {
		t.Fatalf("Get() 失败: %v", err)
	}

	if completed.Status != model.TaskStatusSuccess {
		t.Errorf("完成状态 = %s, want %s", completed.Status, model.TaskStatusSuccess)
	}
	if completed.CompletedAt == nil {
		t.Error("CompletedAt 应该被设置")
	}
	if completed.PID != nil {
		t.Error("任务完成后 PID 应该被清空")
	}
	if completed.ExitCode == nil || *completed.ExitCode != 0 {
		t.Errorf("退出码 = %+v, want 0", completed.ExitCode)
	}
	if completed.LogFilePath != "/tmp/task.log" {
		t.Errorf("LogFilePath = %s, want /tmp/task.log", completed.LogFilePath)
	}
}

func TestManager_MarkRunningStateChanged(t *testing.T) {
	manager, db := setupTaskManager(t)
	defer db.Close()

	workingDir := t.TempDir()
	logDir := t.TempDir()

	task := createTask(t, manager, workingDir, logDir, "echo", []string{"done"})
	if err := manager.MarkCompletion(task.ID, model.TaskStatusSuccess, 0, "/tmp/log.log", ""); err != nil {
		t.Fatalf("预先完成任务失败: %v", err)
	}

	err := manager.MarkRunning(task.ID, 5678)
	if !errors.Is(err, tasks.ErrTaskStateChanged) {
		t.Fatalf("MarkRunning() 应该返回 ErrTaskStateChanged, got %v", err)
	}
}

func TestManager_MarkStopped(t *testing.T) {
	manager, db := setupTaskManager(t)
	defer db.Close()

	workingDir := t.TempDir()
	logDir := t.TempDir()

	task := createTask(t, manager, workingDir, logDir, "sleep", []string{"5"})
	if err := manager.MarkRunning(task.ID, 9876); err != nil {
		t.Fatalf("MarkRunning() 失败: %v", err)
	}

	if err := manager.MarkStopped(task.ID, "", "user requested stop"); err != nil {
		t.Fatalf("MarkStopped() 失败: %v", err)
	}

	stopped, err := manager.Get(task.ID)
	if err != nil {
		t.Fatalf("Get() 失败: %v", err)
	}

	if stopped.Status != model.TaskStatusStopped {
		t.Errorf("停止状态 = %s, want %s", stopped.Status, model.TaskStatusStopped)
	}
	if stopped.CompletedAt == nil {
		t.Error("停止的任务应该设置 CompletedAt")
	}
	if stopped.PID != nil {
		t.Error("停止后 PID 应被清空")
	}
	if stopped.ExitCode == nil || *stopped.ExitCode != -1 {
		t.Errorf("停止后的退出码 = %+v, want -1", stopped.ExitCode)
	}

	if err := manager.MarkStopped(task.ID, "", "stop again"); err == nil {
		t.Error("重复停止已结束的任务应该返回错误")
	}
}
