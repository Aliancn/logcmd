package history_test

import (
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/aliancn/logcmd/internal/history"
	"github.com/aliancn/logcmd/internal/migration"
	"github.com/aliancn/logcmd/internal/model"
)

// setupTestDB 创建测试数据库
func setupTestDB(t *testing.T) *sql.DB {
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)

	dbPath := tmpHome + "/test.db"
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("打开数据库失败: %v", err)
	}

	// 执行数据库迁移
	migrator := migration.NewMigration(db)
	if err := migrator.Migrate(); err != nil {
		db.Close()
		t.Fatalf("数据库迁移失败: %v", err)
	}

	return db
}

func TestNewManager(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	manager := history.NewManager(db)
	if manager == nil {
		t.Fatal("NewManager() 返回了 nil")
	}
}

func TestRecord(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	manager := history.NewManager(db)

	cmd := &model.CommandHistory{
		ProjectID:        1,
		Command:          "echo test",
		CommandArgs:      []string{"echo", "test"},
		StartTime:        time.Now(),
		EndTime:          time.Now().Add(time.Second),
		DurationMs:       1000,
		ExitCode:         0,
		LogFilePath:      "/path/to/log.log",
		LogDate:          "2024-01-01",
		StdoutPreview:    "test",
		WorkingDirectory: "/home/user",
		CreatedAt:        time.Now(),
	}

	err := manager.Record(cmd)
	if err != nil {
		t.Fatalf("Record() 失败: %v", err)
	}
}

func TestQuery(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	manager := history.NewManager(db)

	// 记录几条历史
	commands := []*model.CommandHistory{
		{
			ProjectID:   1,
			Command:     "echo test1",
			CommandArgs: []string{"test1"},
			StartTime:   time.Now(),
			EndTime:     time.Now(),
			DurationMs:  100,
			ExitCode:    0,
			Status:      "success",
			LogDate:     "2024-01-01",
			CreatedAt:   time.Now(),
		},
		{
			ProjectID:   1,
			Command:     "echo test2",
			CommandArgs: []string{"test2"},
			StartTime:   time.Now(),
			EndTime:     time.Now(),
			DurationMs:  200,
			ExitCode:    1,
			Status:      "failed",
			LogDate:     "2024-01-01",
			CreatedAt:   time.Now(),
		},
	}

	for _, cmd := range commands {
		if err := manager.Record(cmd); err != nil {
			t.Fatalf("Record() 失败: %v", err)
		}
	}

	// 查询所有记录
	results, err := manager.Query(history.QueryOptions{
		ProjectID: 1,
	})
	if err != nil {
		t.Fatalf("Query() 失败: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Query() 返回 %d 条记录, want 2", len(results))
	}

	// 查询成功的记录
	successResults, err := manager.Query(history.QueryOptions{
		ProjectID: 1,
		Status:    "success",
	})
	if err != nil {
		t.Fatalf("Query() 失败: %v", err)
	}

	if len(successResults) != 1 {
		t.Errorf("成功记录数 = %d, want 1", len(successResults))
	}

	// 查询失败的记录
	failedResults, err := manager.Query(history.QueryOptions{
		ProjectID: 1,
		Status:    "failed",
	})
	if err != nil {
		t.Fatalf("Query() 失败: %v", err)
	}

	if len(failedResults) != 1 {
		t.Errorf("失败记录数 = %d, want 1", len(failedResults))
	}
}

func TestQueryWithLimit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	manager := history.NewManager(db)

	// 记录多条历史
	for i := 0; i < 10; i++ {
		cmd := &model.CommandHistory{
			ProjectID:  1,
			Command:    "test",
			StartTime:  time.Now(),
			EndTime:    time.Now(),
			DurationMs: 100,
			ExitCode:   0,
			Status:     "success",
			LogDate:    "2024-01-01",
			CreatedAt:  time.Now(),
		}
		if err := manager.Record(cmd); err != nil {
			t.Fatalf("Record() 失败: %v", err)
		}
	}

	// 查询限制数量
	results, err := manager.Query(history.QueryOptions{
		ProjectID: 1,
		Limit:     5,
	})
	if err != nil {
		t.Fatalf("Query() 失败: %v", err)
	}

	if len(results) != 5 {
		t.Errorf("Query() 返回 %d 条记录, want 5", len(results))
	}
}

func TestGetRecent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	manager := history.NewManager(db)

	// 记录几条历史
	for i := 0; i < 5; i++ {
		cmd := &model.CommandHistory{
			ProjectID:  1,
			Command:    "test",
			StartTime:  time.Now(),
			EndTime:    time.Now(),
			DurationMs: 100,
			ExitCode:   0,
			Status:     "success",
			LogDate:    "2024-01-01",
			CreatedAt:  time.Now(),
		}
		if err := manager.Record(cmd); err != nil {
			t.Fatalf("Record() 失败: %v", err)
		}
		time.Sleep(10 * time.Millisecond) // 确保时间不同
	}

	results, err := manager.GetRecent(1, 3)
	if err != nil {
		t.Fatalf("GetRecent() 失败: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("GetRecent() 返回 %d 条记录, want 3", len(results))
	}

	// 验证按时间降序
	for i := 1; i < len(results); i++ {
		if results[i].StartTime.After(results[i-1].StartTime) {
			t.Error("结果应该按时间降序排列")
		}
	}
}

func TestGetFailed(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	manager := history.NewManager(db)

	// 记录成功和失败的命令
	commands := []struct {
		exitCode int
		status   string
	}{
		{0, "success"},
		{1, "failed"},
		{0, "success"},
		{1, "failed"},
	}

	for _, cmd := range commands {
		histCmd := &model.CommandHistory{
			ProjectID:  1,
			Command:    "test",
			StartTime:  time.Now(),
			EndTime:    time.Now(),
			DurationMs: 100,
			ExitCode:   cmd.exitCode,
			Status:     cmd.status,
			LogDate:    "2024-01-01",
			CreatedAt:  time.Now(),
		}
		if err := manager.Record(histCmd); err != nil {
			t.Fatalf("Record() 失败: %v", err)
		}
	}

	results, err := manager.GetFailed(1, 10)
	if err != nil {
		t.Fatalf("GetFailed() 失败: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("GetFailed() 返回 %d 条记录, want 2", len(results))
	}

	// 验证都是失败的
	for _, result := range results {
		if result.Status != "failed" {
			t.Errorf("GetFailed() 应该只返回失败的记录, got status: %s", result.Status)
		}
	}
}

func TestCount(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	manager := history.NewManager(db)

	// 初始数量应该为 0
	count, err := manager.Count(1)
	if err != nil {
		t.Fatalf("Count() 失败: %v", err)
	}
	if count != 0 {
		t.Errorf("初始 Count = %d, want 0", count)
	}

	// 记录几条历史
	for i := 0; i < 3; i++ {
		cmd := &model.CommandHistory{
			ProjectID:  1,
			Command:    "test",
			StartTime:  time.Now(),
			EndTime:    time.Now(),
			DurationMs: 100,
			ExitCode:   0,
			LogDate:    "2024-01-01",
			CreatedAt:  time.Now(),
		}
		if err := manager.Record(cmd); err != nil {
			t.Fatalf("Record() 失败: %v", err)
		}
	}

	// 验证数量
	count, err = manager.Count(1)
	if err != nil {
		t.Fatalf("Count() 失败: %v", err)
	}
	if count != 3 {
		t.Errorf("Count = %d, want 3", count)
	}
}

func TestDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	manager := history.NewManager(db)

	// 记录一条历史
	cmd := &model.CommandHistory{
		ProjectID:  1,
		Command:    "test",
		StartTime:  time.Now(),
		EndTime:    time.Now(),
		DurationMs: 100,
		ExitCode:   0,
		LogDate:    "2024-01-01",
		CreatedAt:  time.Now(),
	}
	if err := manager.Record(cmd); err != nil {
		t.Fatalf("Record() 失败: %v", err)
	}

	// 查询获取 ID
	results, err := manager.Query(history.QueryOptions{ProjectID: 1})
	if err != nil || len(results) == 0 {
		t.Fatalf("Query() 失败或无结果")
	}

	recordID := results[0].ID

	// 删除记录
	err = manager.Delete(recordID)
	if err != nil {
		t.Fatalf("Delete() 失败: %v", err)
	}

	// 验证已删除
	count, err := manager.Count(1)
	if err != nil {
		t.Fatalf("Count() 失败: %v", err)
	}
	if count != 0 {
		t.Errorf("删除后 Count = %d, want 0", count)
	}
}

func TestDeleteByProject(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	manager := history.NewManager(db)

	// 为项目 1 记录几条历史
	for i := 0; i < 3; i++ {
		cmd := &model.CommandHistory{
			ProjectID:  1,
			Command:    "test",
			StartTime:  time.Now(),
			EndTime:    time.Now(),
			DurationMs: 100,
			ExitCode:   0,
			LogDate:    "2024-01-01",
			CreatedAt:  time.Now(),
		}
		if err := manager.Record(cmd); err != nil {
			t.Fatalf("Record() 失败: %v", err)
		}
	}

	// 删除项目的所有历史
	err := manager.DeleteByProject(1)
	if err != nil {
		t.Fatalf("DeleteByProject() 失败: %v", err)
	}

	// 验证已全部删除
	count, err := manager.Count(1)
	if err != nil {
		t.Fatalf("Count() 失败: %v", err)
	}
	if count != 0 {
		t.Errorf("DeleteByProject 后 Count = %d, want 0", count)
	}
}
