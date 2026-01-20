package application

import (
	"fmt"

	"github.com/aliancn/logcmd/internal/platform/history"
	"github.com/aliancn/logcmd/internal/platform/registry"
	"github.com/aliancn/logcmd/internal/platform/tasks"
)

// Container 负责集中管理 CLI 运行期需要的依赖。
// 通过统一的入口，命令实现只需要关注业务逻辑，符合 SOLID 中的依赖倒置与单一职责原则。
type Container struct {
	registry       *registry.Registry
	taskManager    *tasks.Manager
	historyManager *history.Manager
}

// NewContainer 构建依赖容器。
func NewContainer() (*Container, error) {
	reg, err := registry.New()
	if err != nil {
		return nil, fmt.Errorf("初始化项目注册表失败: %w", err)
	}
	return &Container{registry: reg}, nil
}

// Registry 暴露底层注册表实例。
func (c *Container) Registry() *registry.Registry {
	if c == nil {
		return nil
	}
	return c.registry
}

// TaskManager 以延迟初始化的方式提供任务管理器。
func (c *Container) TaskManager() (*tasks.Manager, error) {
	if c == nil {
		return nil, fmt.Errorf("依赖容器未初始化")
	}
	if c.taskManager != nil {
		return c.taskManager, nil
	}
	manager := tasks.NewManager(c.registry.GetDB())
	if manager == nil {
		return nil, fmt.Errorf("任务管理器初始化失败")
	}
	c.taskManager = manager
	return c.taskManager, nil
}

// HistoryManager 同样采用延迟初始化，避免无谓的数据库句柄创建。
func (c *Container) HistoryManager() *history.Manager {
	if c == nil || c.registry == nil {
		return nil
	}
	if c.historyManager != nil {
		return c.historyManager
	}
	c.historyManager = history.NewManager(c.registry.GetDB())
	return c.historyManager
}

// Close 归还底层资源。
func (c *Container) Close() {
	if c == nil || c.registry == nil {
		return
	}
	c.registry.Close()
}
