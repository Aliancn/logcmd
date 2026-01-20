package ui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aliancn/logcmd/internal/platform/history"
	"github.com/aliancn/logcmd/internal/domain/model"
	"github.com/aliancn/logcmd/internal/platform/registry"
	"github.com/aliancn/logcmd/internal/platform/tasks"
	"github.com/aliancn/logcmd/internal/presentation/ui/common"
	"github.com/aliancn/logcmd/internal/presentation/ui/modes"
)

// App 是新的极简根 Model
//
// 采用模式系统替代原有的 Tab 系统，每次只有一个模式处于激活状态。
// 用户通过快捷键在模式间快速切换，启动时默认进入搜索模式。
type App struct {
	// 核心依赖
	registry   *registry.Registry
	historyMgr *history.Manager
	taskMgr    *tasks.Manager

	// 模式管理
	currentMode Mode
	modes       map[string]Mode

	// 全局状态
	width  int
	height int
	ready  bool

	// 样式和主题
	theme  common.Theme
	styles common.Styles
}

// Mode 是 modes.Mode 的类型别名，简化代码
type Mode = modes.Mode

// NewApp 创建新的应用实例
func NewApp(reg *registry.Registry, historyMgr *history.Manager, taskMgr *tasks.Manager) *App {
	app := &App{
		registry:   reg,
		historyMgr: historyMgr,
		taskMgr:    taskMgr,
		theme:      common.DefaultTheme(),
		modes:      make(map[string]Mode),
	}

	app.styles = common.NewStyles(app.theme)

	// 注册所有模式
	app.modes["search"] = modes.NewSearchMode(reg, historyMgr, app.theme, app.styles)
	app.modes["project"] = modes.NewProjectMode(reg, historyMgr, app.theme, app.styles)
	app.modes["task"] = modes.NewTaskMode(taskMgr, app.theme, app.styles)
	app.modes["stats"] = modes.NewStatsMode(historyMgr, app.theme, app.styles)
	app.modes["command"] = modes.NewCommandMode(app.theme, app.styles)
	app.modes["logview"] = modes.NewLogViewMode(app.theme, app.styles)

	// 设置默认模式为搜索
	app.currentMode = app.modes["search"]

	return app
}

// Init 实现 tea.Model 接口
func (app *App) Init() tea.Cmd {
	if app.currentMode != nil {
		return app.currentMode.Activate()
	}
	return nil
}

// Update 实现 tea.Model 接口
func (app *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		app.width, app.height = msg.Width, msg.Height
		app.ready = true

	case tea.KeyMsg:
		key := msg.String()

		// 全局模式切换快捷键（优先级最高）
		// 使用 ctrl+ 组合键避免与搜索输入冲突
		switch key {
		case "/":
			return app, app.SwitchMode("search")
		case "ctrl+p":
			return app, app.SwitchMode("project")
		case "ctrl+t":
			return app, app.SwitchMode("task")
		case "ctrl+s":
			return app, app.SwitchMode("stats")
		case "ctrl+l":
			return app, app.SwitchMode("command")
		case "ctrl+c":
			return app, tea.Quit
		}

		// 让当前模式处理全局快捷键
		if app.currentMode != nil {
			if handled, cmd := app.currentMode.HandleKey(key); handled {
				return app, cmd
			}
		}

	case modes.SwitchModeMsg:
		// 处理模式内部发出的切换请求
		// 如果携带数据，先设置到目标模式
		if msg.Data != nil && msg.ModeName == "search" {
			if proj, ok := msg.Data.(*model.Project); ok {
				if searchMode, ok := app.modes["search"].(*modes.SearchMode); ok {
					searchMode.SetProject(proj)
				}
			}
		}
		if msg.Data != nil && msg.ModeName == "project" {
			if state, ok := msg.Data.(*modes.ProjectReturnState); ok {
				if projectMode, ok := app.modes["project"].(*modes.ProjectMode); ok {
					projectMode.SetReturnState(state)
				}
			}
		}
		if msg.Data != nil && msg.ModeName == "stats" {
			if proj, ok := msg.Data.(*model.Project); ok {
				if statsMode, ok := app.modes["stats"].(*modes.StatsMode); ok {
					statsMode.SetProject(proj)
				}
			}
		}
		return app, app.SwitchMode(msg.ModeName)

	case modes.OpenLogFileMsg:
		// 处理打开日志文件请求
		if logViewMode, ok := app.modes["logview"].(*modes.LogViewMode); ok {
			// 设置日志文件信息
			returnMode := msg.ReturnMode
			if returnMode == "" {
				returnMode = "search"
			}
			logViewMode.SetFile(msg.FilePath, msg.LineNum, msg.SearchQuery, returnMode, msg.Follow, msg.ReturnData)

			// 如果当前已经是 logview 模式，需要重新激活以加载新文件
			if app.currentMode == app.modes["logview"] {
				// 已经在 logview 模式，直接重新激活以加载新文件
				return app, logViewMode.Activate()
			}

			// 否则切换到日志查看模式
			return app, app.SwitchMode("logview")
		}

	case modes.SearchWithKeywordMsg:
		// 处理带关键词的搜索请求
		if searchMode, ok := app.modes["search"].(*modes.SearchMode); ok {
			searchMode.SetSearchKeyword(msg.Keyword)
		}
		// 切换到搜索模式
		return app, app.SwitchMode("search")
	}

	// 路由消息到当前模式
	if app.currentMode != nil {
		newMode, cmd := app.currentMode.Update(msg)
		app.currentMode = newMode
		return app, cmd
	}

	return app, nil
}

// View 实现 tea.Model 接口
func (app *App) View() string {
	if !app.ready {
		return "初始化中..."
	}

	if app.currentMode == nil {
		return app.renderError("错误: 未设置任何模式")
	}

	return app.currentMode.View()
}

// SwitchMode 切换到指定模式
func (app *App) SwitchMode(modeName string) tea.Cmd {
	newMode, ok := app.modes[modeName]
	if !ok {
		// 模式不存在，返回错误提示命令
		return func() tea.Msg {
			return common.ErrorMsg{Err: fmt.Errorf("模式 '%s' 不存在", modeName)}
		}
	}

	if newMode == app.currentMode {
		// 已经是当前模式，无需切换
		return nil
	}

	var cmds []tea.Cmd

	// 1. 停用当前模式
	if app.currentMode != nil {
		if cmd := app.currentMode.Deactivate(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// 2. 切换模式
	app.currentMode = newMode

	// 3. 如果已有尺寸信息，立即同步到新模式
	//    这样可以避免新模式显示"初始化中..."的闪烁
	if app.ready && app.width > 0 && app.height > 0 {
		sizeMsg := tea.WindowSizeMsg{
			Width:  app.width,
			Height: app.height,
		}
		// 立即更新新模式的尺寸
		updatedMode, cmd := app.currentMode.Update(sizeMsg)
		app.currentMode = updatedMode
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// 4. 激活新模式
	if cmd := app.currentMode.Activate(); cmd != nil {
		cmds = append(cmds, cmd)
	}

	return tea.Batch(cmds...)
}

// renderError 渲染错误信息
func (app *App) renderError(errMsg string) string {
	if app.width == 0 {
		return errMsg
	}

	// 简单的错误显示，居中
	padding := (app.height - 3) / 2
	if padding < 0 {
		padding = 0
	}

	var view string
	for i := 0; i < padding; i++ {
		view += "\n"
	}

	// 使用错误颜色渲染
	errorStyle := lipgloss.NewStyle().Foreground(app.theme.Error).Bold(true)
	view += errorStyle.Render(errMsg)
	return view
}

// GetRegistry 返回 registry 实例（用于模式访问）
func (app *App) GetRegistry() *registry.Registry {
	return app.registry
}

// GetHistoryManager 返回 history manager 实例（用于模式访问）
func (app *App) GetHistoryManager() *history.Manager {
	return app.historyMgr
}

// GetTaskManager 返回 task manager 实例（用于模式访问）
func (app *App) GetTaskManager() *tasks.Manager {
	return app.taskMgr
}

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

	// 使用新的 App 替代原有的 RootModel
	model := NewApp(reg, historyMgr, taskMgr)
	program := tea.NewProgram(model, tea.WithAltScreen(), tea.WithContext(ctx))
	_, err := program.Run()
	return err
}
