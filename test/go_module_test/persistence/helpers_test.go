package persistence_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aliancn/logcmd/internal/model"
	"github.com/aliancn/logcmd/internal/registry"
)

func newTestRegistry(t *testing.T) *registry.Registry {
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

	return reg
}

func registerTestProject(t *testing.T, reg *registry.Registry) (*model.Project, string) {
	t.Helper()

	projectRoot := filepath.Join(t.TempDir(), "project")
	logDir := filepath.Join(projectRoot, ".logcmd")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatalf("创建日志目录失败: %v", err)
	}

	project, err := reg.Register(logDir)
	if err != nil {
		t.Fatalf("注册项目失败: %v", err)
	}

	return project, logDir
}
