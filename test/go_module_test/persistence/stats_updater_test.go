package persistence_test

import (
	"testing"
	"time"

	"github.com/aliancn/logcmd/internal/persistence"
)

func TestNewStatsUpdaterRequiresRegistry(t *testing.T) {
	if updater := persistence.NewStatsUpdater(nil); updater != nil {
		t.Fatal("传入 nil registry 时应该返回 nil")
	}
}

func TestStatsUpdaterUpdateProjectStats(t *testing.T) {
	reg := newTestRegistry(t)
	project, _ := registerTestProject(t, reg)

	updater := persistence.NewStatsUpdater(reg)
	if updater == nil {
		t.Fatal("NewStatsUpdater() 不应返回 nil")
	}

	duration := 1500 * time.Millisecond
	if err := updater.UpdateProjectStats(project.ID, "go test", true, duration); err != nil {
		t.Fatalf("UpdateProjectStats() 失败: %v", err)
	}

	row := reg.GetDB().QueryRow(`
		SELECT total_commands, success_commands, failed_commands, total_duration_ms, last_command, last_command_status
		FROM projects WHERE id = ?`, project.ID)
	var total, success, failed int
	var totalDuration int64
	var lastCommand, lastStatus string
	if err := row.Scan(&total, &success, &failed, &totalDuration, &lastCommand, &lastStatus); err != nil {
		t.Fatalf("读取项目记录失败: %v", err)
	}

	if total != 1 || success != 1 || failed != 0 {
		t.Fatalf("统计字段不正确: total=%d success=%d failed=%d", total, success, failed)
	}
	if totalDuration != duration.Milliseconds() {
		t.Fatalf("total_duration_ms = %d, want %d", totalDuration, duration.Milliseconds())
	}
	if lastCommand != "go test" {
		t.Fatalf("last_command = %s, want go test", lastCommand)
	}
	if lastStatus != "success" {
		t.Fatalf("last_command_status = %s, want success", lastStatus)
	}
}

func TestStatsUpdaterUpdateRequiresRegistry(t *testing.T) {
	var updater *persistence.StatsUpdater
	if err := updater.UpdateProjectStats(1, "echo", true, time.Second); err == nil {
		t.Fatal("未初始化的 StatsUpdater 应返回错误")
	}
}
