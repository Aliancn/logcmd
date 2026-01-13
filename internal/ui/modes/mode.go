package modes

import tea "github.com/charmbracelet/bubbletea"

// Mode 表示一个 UI 操作模式
//
// LogCmd TUI 采用模式系统而非传统的 Tab 系统，类似 vim 的设计理念。
// 用户通过快捷键在不同模式间切换：
//   - /      : 搜索模式（默认）
//   - Ctrl+P : 项目模式
//   - Ctrl+T : 任务模式
//   - Ctrl+S : 统计模式
//   - Ctrl+L : 命令模式
//
// 每个模式都是独立的、自包含的 UI 单元，负责渲染完整的界面和处理用户交互。
type Mode interface {
	// Name 返回模式的唯一标识符
	// 用于模式注册和切换，例如: "search", "project", "task", "stats", "command"
	Name() string

	// Activate 在模式被激活时调用
	// 返回的 tea.Cmd 用于执行初始化操作，如加载数据、启动定时器等
	// 如果无需初始化，可返回 nil
	Activate() tea.Cmd

	// Deactivate 在模式被停用（切换到其他模式）时调用
	// 返回的 tea.Cmd 用于执行清理操作，如停止定时器、释放资源等
	// 如果无需清理，可返回 nil
	Deactivate() tea.Cmd

	// Update 处理来自 bubbletea 的消息
	//
	// 注意: 返回 Mode 接口而非具体类型，这允许模式间的动态切换。
	// 例如，ProjectMode 可以在用户选择项目后返回切换到 SearchMode：
	//   return app.modes["search"], switchToSearchCmd
	//
	// 大多数情况下，模式只处理自己的消息并返回 self：
	//   return m, someCmd
	Update(msg tea.Msg) (Mode, tea.Cmd)

	// View 渲染模式的完整视图
	//
	// 每个模式负责渲染自己的完整界面，包括：
	//   - 状态栏（显示模式名称和关键信息）
	//   - 主内容区域
	//   - 底部快捷键提示
	//
	// 不需要考虑全局布局，App 会直接显示 Mode.View() 的返回值。
	View() string

	// HandleKey 处理全局快捷键
	//
	// 在 Update() 之前被调用，用于处理模式切换等全局操作。
	// 返回值：
	//   - handled: true 表示该快捷键已被处理，不再传递给 Update()
	//   - cmd: 要执行的命令，如模式切换命令
	//
	// 典型用法：
	//   func (m *SearchMode) HandleKey(key string) (bool, tea.Cmd) {
	//       switch key {
	//       case "ctrl+a":
	//           m.toggleSearchScope()
	//           return true, nil
	//       default:
	//           return false, nil
	//       }
	//   }
	HandleKey(key string) (bool, tea.Cmd)
}

// SwitchModeMsg 是模式切换消息
// 当某个模式需要切换到另一个模式时，发送此消息给 App
type SwitchModeMsg struct {
	ModeName string
	Data     interface{} // 可选的数据传递，如切换到搜索模式时指定项目
}

// OpenLogFileMsg 是打开日志文件消息
// 用于从SearchMode打开日志文件到LogViewMode
type OpenLogFileMsg struct {
	FilePath    string
	LineNum     int
	SearchQuery string
	ReturnMode  string
	Follow      bool
}

// SearchWithKeywordMsg 是带关键词的搜索消息
// 用于从CommandMode触发搜索
type SearchWithKeywordMsg struct {
	Keyword string
}
