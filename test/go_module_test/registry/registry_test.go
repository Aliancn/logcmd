package registry_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aliancn/logcmd/internal/registry"
)

// setupTestRegistry 创建测试用的 Registry 实例
func setupTestRegistry(t *testing.T) *registry.Registry {
	// 使用临时目录作为 HOME
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)

	reg, err := registry.New()
	if err != nil {
		t.Fatalf("创建 Registry 失败: %v", err)
	}

	return reg
}

func TestNew(t *testing.T) {
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	defer os.Unsetenv("HOME")

	reg, err := registry.New()
	if err != nil {
		t.Fatalf("New() 失败: %v", err)
	}
	defer reg.Close()

	if reg == nil {
		t.Fatal("New() 返回了 nil")
	}

	// 验证数据库文件已创建
	dbPath := filepath.Join(tmpHome, ".logcmd", "data", "registry.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("数据库文件应该已创建")
	}
}

func TestRegister(t *testing.T) {
	reg := setupTestRegistry(t)
	defer reg.Close()

	// 创建测试目录
	testDir := t.TempDir()

	// 注册项目
	project, err := reg.Register(testDir)
	if err != nil {
		t.Fatalf("Register() 失败: %v", err)
	}

	if project == nil {
		t.Fatal("project 不应为 nil")
	}

	// 验证项目字段
	if project.ID == 0 {
		t.Error("project.ID 不应为 0")
	}

	absPath, _ := filepath.Abs(testDir)
	if project.Path != absPath {
		t.Errorf("project.Path = %s, want %s", project.Path, absPath)
	}

	if project.Name == "" {
		t.Error("project.Name 不应为空")
	}

	if project.CreatedAt.IsZero() {
		t.Error("project.CreatedAt 不应为零值")
	}
}

func TestRegisterDuplicate(t *testing.T) {
	reg := setupTestRegistry(t)
	defer reg.Close()

	testDir := t.TempDir()

	// 第一次注册
	project1, err := reg.Register(testDir)
	if err != nil {
		t.Fatalf("第一次 Register() 失败: %v", err)
	}

	// 第二次注册相同目录
	project2, err := reg.Register(testDir)
	if err != nil {
		t.Fatalf("第二次 Register() 失败: %v", err)
	}

	// 应该返回相同的项目 ID
	if project1.ID != project2.ID {
		t.Errorf("重复注册应该返回相同的项目: ID1=%d, ID2=%d", project1.ID, project2.ID)
	}

	// UpdatedAt 应该被更新
	if !project2.UpdatedAt.After(project1.UpdatedAt) && !project2.UpdatedAt.Equal(project1.UpdatedAt) {
		t.Error("重复注册应该更新 UpdatedAt")
	}
}

func TestRegisterNonExistentDirectory(t *testing.T) {
	reg := setupTestRegistry(t)
	defer reg.Close()

	// 尝试注册不存在的目录
	_, err := reg.Register("/nonexistent/directory/12345")
	if err == nil {
		t.Error("Register() 应该对不存在的目录返回错误")
	}
}

func TestList(t *testing.T) {
	reg := setupTestRegistry(t)
	defer reg.Close()

	// 注册多个项目
	dirs := []string{
		t.TempDir(),
		t.TempDir(),
		t.TempDir(),
	}

	for _, dir := range dirs {
		_, err := reg.Register(dir)
		if err != nil {
			t.Fatalf("Register() 失败: %v", err)
		}
	}

	// 列出所有项目
	projects, err := reg.List()
	if err != nil {
		t.Fatalf("List() 失败: %v", err)
	}

	if len(projects) != len(dirs) {
		t.Errorf("List() 返回 %d 个项目, want %d", len(projects), len(dirs))
	}

	// 验证项目按更新时间降序排列
	for i := 1; i < len(projects); i++ {
		if projects[i].UpdatedAt.After(projects[i-1].UpdatedAt) {
			t.Error("项目应该按 UpdatedAt 降序排列")
		}
	}
}

func TestGet(t *testing.T) {
	reg := setupTestRegistry(t)
	defer reg.Close()

	testDir := t.TempDir()

	// 注册项目
	originalProject, err := reg.Register(testDir)
	if err != nil {
		t.Fatalf("Register() 失败: %v", err)
	}

	// 通过 ID 获取
	t.Run("通过ID获取", func(t *testing.T) {
		idStr := fmt.Sprintf("%d", originalProject.ID)
		project, err := reg.Get(idStr)
		if err != nil {
			t.Fatalf("Get() by ID 失败: %v", err)
		}

		if project.ID != originalProject.ID {
			t.Errorf("project.ID = %d, want %d", project.ID, originalProject.ID)
		}
	})

	// 通过路径获取
	t.Run("通过路径获取", func(t *testing.T) {
		project, err := reg.Get(testDir)
		if err != nil {
			t.Fatalf("Get() by path 失败: %v", err)
		}

		if project.ID != originalProject.ID {
			t.Errorf("project.ID = %d, want %d", project.ID, originalProject.ID)
		}
	})
}

func TestGetNonExistent(t *testing.T) {
	reg := setupTestRegistry(t)
	defer reg.Close()

	// 获取不存在的项目
	_, err := reg.Get("99999")
	if err == nil {
		t.Error("Get() 应该对不存在的项目返回错误")
	}

	_, err = reg.Get("/nonexistent/path")
	if err == nil {
		t.Error("Get() 应该对不存在的路径返回错误")
	}
}

func TestUpdate(t *testing.T) {
	reg := setupTestRegistry(t)
	defer reg.Close()

	testDir := t.TempDir()

	// 注册项目
	project, err := reg.Register(testDir)
	if err != nil {
		t.Fatalf("Register() 失败: %v", err)
	}

	// 修改项目信息
	project.Description = "测试项目"
	project.Category = "test"
	project.Tags = []string{"tag1", "tag2"}

	err = reg.Update(project)
	if err != nil {
		t.Fatalf("Update() 失败: %v", err)
	}

	// 重新获取项目验证更新
	idStr := fmt.Sprintf("%d", project.ID)
	updatedProject, err := reg.Get(idStr)
	if err != nil {
		t.Fatalf("Get() 失败: %v", err)
	}

	if updatedProject.Description != "测试项目" {
		t.Errorf("Description = %s, want 测试项目", updatedProject.Description)
	}

	if updatedProject.Category != "test" {
		t.Errorf("Category = %s, want test", updatedProject.Category)
	}

	if len(updatedProject.Tags) != 2 {
		t.Errorf("Tags 长度 = %d, want 2", len(updatedProject.Tags))
	}
}

func TestUpdateStats(t *testing.T) {
	reg := setupTestRegistry(t)
	defer reg.Close()

	testDir := t.TempDir()

	// 注册项目
	project, err := reg.Register(testDir)
	if err != nil {
		t.Fatalf("Register() 失败: %v", err)
	}

	// 更新统计信息
	err = reg.UpdateStats(project.ID, "test command", true, 1*time.Second)
	if err != nil {
		t.Fatalf("UpdateStats() 失败: %v", err)
	}

	// 获取更新后的项目
	idStr := fmt.Sprintf("%d", project.ID)
	updatedProject, err := reg.Get(idStr)
	if err != nil {
		t.Fatalf("Get() 失败: %v", err)
	}

	// 验证统计信息
	if updatedProject.TotalCommands != 1 {
		t.Errorf("TotalCommands = %d, want 1", updatedProject.TotalCommands)
	}

	if updatedProject.SuccessCommands != 1 {
		t.Errorf("SuccessCommands = %d, want 1", updatedProject.SuccessCommands)
	}

	if updatedProject.LastCommand != "test command" {
		t.Errorf("LastCommand = %s, want test command", updatedProject.LastCommand)
	}

	if updatedProject.LastCommandStatus != "success" {
		t.Errorf("LastCommandStatus = %s, want success", updatedProject.LastCommandStatus)
	}

	// 再次更新（失败命令）
	err = reg.UpdateStats(project.ID, "failing command", false, 500*time.Millisecond)
	if err != nil {
		t.Fatalf("第二次 UpdateStats() 失败: %v", err)
	}

	updatedProject2, err := reg.Get(idStr)
	if err != nil {
		t.Fatalf("Get() 失败: %v", err)
	}

	if updatedProject2.TotalCommands != 2 {
		t.Errorf("TotalCommands = %d, want 2", updatedProject2.TotalCommands)
	}

	if updatedProject2.FailedCommands != 1 {
		t.Errorf("FailedCommands = %d, want 1", updatedProject2.FailedCommands)
	}
}

func TestDelete(t *testing.T) {
	reg := setupTestRegistry(t)
	defer reg.Close()

	testDir := t.TempDir()

	// 注册项目
	project, err := reg.Register(testDir)
	if err != nil {
		t.Fatalf("Register() 失败: %v", err)
	}

	// 删除项目
	idStr := fmt.Sprintf("%d", project.ID)
	err = reg.Delete(idStr)
	if err != nil {
		t.Fatalf("Delete() 失败: %v", err)
	}

	// 验证项目已删除
	_, err = reg.Get(idStr)
	if err == nil {
		t.Error("删除后 Get() 应该返回错误")
	}
}

func TestDeleteByPath(t *testing.T) {
	reg := setupTestRegistry(t)
	defer reg.Close()

	testDir := t.TempDir()

	// 注册项目
	_, err := reg.Register(testDir)
	if err != nil {
		t.Fatalf("Register() 失败: %v", err)
	}

	// 通过路径删除项目
	err = reg.Delete(testDir)
	if err != nil {
		t.Fatalf("Delete() 失败: %v", err)
	}

	// 验证项目已删除
	_, err = reg.Get(testDir)
	if err == nil {
		t.Error("删除后 Get() 应该返回错误")
	}
}

func TestDeleteNonExistent(t *testing.T) {
	reg := setupTestRegistry(t)
	defer reg.Close()

	// 删除不存在的项目
	err := reg.Delete("99999")
	if err == nil {
		t.Error("Delete() 应该对不存在的项目返回错误")
	}
}

func TestClose(t *testing.T) {
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	defer os.Unsetenv("HOME")

	reg, err := registry.New()
	if err != nil {
		t.Fatalf("New() 失败: %v", err)
	}

	err = reg.Close()
	if err != nil {
		t.Errorf("Close() 失败: %v", err)
	}

	// 关闭后再次关闭应该安全
	err = reg.Close()
	if err != nil {
		t.Errorf("二次 Close() 失败: %v", err)
	}
}
