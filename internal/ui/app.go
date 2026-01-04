package ui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/aliancn/logcmd/internal/history"
	"github.com/aliancn/logcmd/internal/registry"
	"github.com/aliancn/logcmd/internal/tasks"
)

// Start 使用默认依赖启动 TUI。
func Start(ctx context.Context, reg *registry.Registry) error {
	if reg == nil {
		return fmt.Errorf("registry 未初始化")
	}
	historyMgr := history.NewManager(reg.GetDB())
	taskMgr := tasks.NewManager(reg.GetDB())
	if taskMgr == nil {
		return fmt.Errorf("任务管理器未初始化")
	}
	return StartWithDependencies(ctx, reg, historyMgr, taskMgr)
}

// StartWithDependencies 使用注入的依赖启动 TUI，方便测试。
func StartWithDependencies(ctx context.Context, reg *registry.Registry, historyMgr *history.Manager, taskMgr *tasks.Manager) error {
	if reg == nil {
		return fmt.Errorf("registry 未初始化")
	}
	if historyMgr == nil {
		return fmt.Errorf("history 管理器未初始化")
	}
	if taskMgr == nil {
		return fmt.Errorf("任务管理器未初始化")
	}

	model := NewRootModel(reg, historyMgr, taskMgr)
	program := tea.NewProgram(model, tea.WithAltScreen(), tea.WithContext(ctx))
	_, err := program.Run()
	return err
}
