package cmd

import (
	"fmt"

	"github.com/aliancn/logcmd/internal/registry"
	"github.com/aliancn/logcmd/internal/tasks"
)

// cliServices 提供 CLI 命令使用的共享依赖
type cliServices struct {
	registry    *registry.Registry
	taskManager *tasks.Manager
}

// newCLIServices 初始化基础依赖
func newCLIServices() (*cliServices, error) {
	reg, err := registry.New()
	if err != nil {
		return nil, fmt.Errorf("初始化项目注册表失败: %w", err)
	}
	return &cliServices{registry: reg}, nil
}

func (s *cliServices) Registry() *registry.Registry {
	if s == nil {
		return nil
	}
	return s.registry
}

func (s *cliServices) TaskManager() (*tasks.Manager, error) {
	if s == nil {
		return nil, fmt.Errorf("CLI 服务未初始化")
	}
	if s.taskManager != nil {
		return s.taskManager, nil
	}
	manager := tasks.NewManager(s.registry.GetDB())
	if manager == nil {
		return nil, fmt.Errorf("任务管理器初始化失败")
	}
	s.taskManager = manager
	return s.taskManager, nil
}

func (s *cliServices) Close() {
	if s == nil || s.registry == nil {
		return
	}
	s.registry.Close()
}
