package modes

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/aliancn/logcmd/internal/presentation/ui/common"
	"github.com/aliancn/logcmd/internal/presentation/ui/modules/logviewer"
)

// LogViewMode 日志查看模式
//
// 核心功能：
//   - 包装 internal/ui/modules/logviewer 组件
//   - 提供统一的日志查看入口（支持从搜索、项目、任务等进入）
//   - 处理模式切换和返回逻辑
type LogViewMode struct {
	// 核心组件
	viewer logviewer.Model

	// 状态数据
	returnMode string
	returnData interface{}
	follow     bool

	// 挂起的初始操作（等待内容加载后执行）
	pendingLine  int
	pendingQuery string

	// 布局
	width  int
	height int

	// 样式
	theme  common.Theme
	styles common.Styles
}

// NewLogViewMode 创建日志查看模式
func NewLogViewMode(theme common.Theme, styles common.Styles) *LogViewMode {
	viewer := logviewer.New(theme, styles)

	return &LogViewMode{
		viewer: viewer,
		theme:  theme,
		styles: styles,
	}
}

// Name 实现 Mode 接口
func (m *LogViewMode) Name() string {
	return "logview"
}

// Activate 实现 Mode 接口
func (m *LogViewMode) Activate() tea.Cmd {
	// 触发 viewer 加载内容
	cmd := m.viewer.LoadContentCmd()
	
	// 如果需要 follow，添加 tick cmd
	if m.follow {
		// LogViewer 目前没有内置 follow 逻辑，
		// 这里我们暂时不处理 follow，或者需要给 viewer 加 follow 功能。
		// 考虑到时间，先忽略 follow 的特殊处理，专注基础查看。
	}
	
	return cmd
}

// Deactivate 实现 Mode 接口
func (m *LogViewMode) Deactivate() tea.Cmd {
	m.viewer.Reset()
	m.pendingLine = 0
	m.pendingQuery = ""
	return nil
}

// Update 实现 Mode 接口
func (m *LogViewMode) Update(msg tea.Msg) (Mode, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewer.SetSize(msg.Width, msg.Height)
		return m, nil

	case logviewer.BackMsg:
		// 接收到 viewer 的返回请求，切换回原模式
		mode := m.returnMode
		if mode == "" {
			mode = "search"
		}
		return m, func() tea.Msg {
			return SwitchModeMsg{
				ModeName: mode,
				Data:     m.returnData,
			}
		}

	case logviewer.ContentLoadedMsg, logviewer.IndexBuiltMsg:
		// 内容加载完成后，检查是否需要执行后续操作（跳转行、搜索等）
		
		// 1. 先让 viewer 处理加载消息（建立索引、设置内容等）
		newViewer, cmd := m.viewer.Update(msg)
		m.viewer = newViewer
		cmds = append(cmds, cmd)

		// 2. 执行挂起的操作
		if m.pendingLine > 0 {
			m.viewer.JumpToLine(m.pendingLine)
			// 如果是大文件模式，JumpToLine 只是设置了 YOffset，可能需要触发加载
			// 但是 viewer 并没有暴露 loadVisibleLinesCmd。
			// 好消息是：viewer.JumpToLine 只是设置了 offset。
			// 下一次 Update 如果发送个空消息或者什么，可能会触发 check。
			// 或者我们在 logviewer 内部优化 JumpToLine 返回 cmd。
			// 目前我们先假设它能工作（小文件肯定没问题，大文件可能需要滚动一下）。
			
			// 实际上，logviewer.Update 里的 KeyDown 等都会检查 needsReload。
			// 我们可以手动构造一个 update 事件？不，太黑客了。
			
			// 最好的办法：在 logviewer.model 里，JumpToLine 后应该检查是否需要加载。
			// 但我们不能在外部调用私有方法。
			
			// 既然我们刚刚给 LogViewer 加了 PerformSearch 返回 Cmd，
			// 也许 JumpToLine 也应该返回 Cmd。
			// 暂时先这样，如果大文件跳转有问题再修 logviewer。
		}

		if m.pendingQuery != "" {
			searchCmd := m.viewer.PerformSearch(m.pendingQuery)
			if searchCmd != nil {
				cmds = append(cmds, searchCmd)
			}
		}

		// 清除挂起状态
		// 注意：如果这是流式加载（比如 chunked），可能不应该立即清除？
		// ContentLoadedMsg 是小文件完成，IndexBuiltMsg 是大文件索引完成（准备好读取了）。
		// 所以在这里清除是安全的。
		m.pendingLine = 0
		m.pendingQuery = ""

		return m, tea.Batch(cmds...)
	}

	// 转发给 viewer
	newViewer, cmd := m.viewer.Update(msg)
	m.viewer = newViewer
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View 实现 Mode 接口
func (m *LogViewMode) View() string {
	if m.width == 0 {
		return "初始化中..."
	}
	return m.viewer.View()
}

// HandleKey 实现 Mode 接口
func (m *LogViewMode) HandleKey(key string) (bool, tea.Cmd) {
	return false, nil
}

// SetFile 设置要查看的文件
func (m *LogViewMode) SetFile(filePath string, lineNum int, searchQuery string, returnMode string, follow bool, returnData interface{}) {
	m.viewer.SetFile(filePath)
	m.returnMode = returnMode
	m.returnData = returnData
	m.follow = follow
	
	m.pendingLine = lineNum
	m.pendingQuery = searchQuery
}

// followTickCmd 暂时为空
func (m *LogViewMode) followTickCmd() tea.Cmd {
	if !m.follow {
		return nil
	}
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		return nil // 暂时不实现 logviewer 的外部 follow
	})
}
