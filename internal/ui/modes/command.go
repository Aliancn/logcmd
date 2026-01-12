package modes

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aliancn/logcmd/internal/ui/common"
)

// CommandMode 命令模式（类似vim的:命令模式）
//
// 核心功能：
//   - 提供vim风格的命令输入
//   - 支持基本命令：help, quit, search, set等
//   - 命令自动补全和历史
type CommandMode struct {
	// UI 组件
	input textinput.Model

	// 状态
	output       string // 命令执行结果
	history      []string
	historyIndex int
	error        string

	// 布局
	width  int
	height int

	// 样式
	theme  common.Theme
	styles common.Styles
}

// NewCommandMode 创建命令模式
func NewCommandMode(theme common.Theme, styles common.Styles) *CommandMode {
	ti := textinput.New()
	ti.Placeholder = "输入命令 (如 :help)"
	ti.Focus()
	ti.CharLimit = 200
	ti.Prompt = ":"

	return &CommandMode{
		input:   ti,
		history: make([]string, 0, 50),
		theme:   theme,
		styles:  styles,
	}
}

// Name 实现 Mode 接口
func (m *CommandMode) Name() string {
	return "command"
}

// Activate 实现 Mode 接口
func (m *CommandMode) Activate() tea.Cmd {
	m.input.Focus()
	m.input.SetValue("")
	m.output = ""
	m.error = ""
	return nil
}

// Deactivate 实现 Mode 接口
func (m *CommandMode) Deactivate() tea.Cmd {
	m.input.Blur()
	return nil
}

// Update 实现 Mode 接口
func (m *CommandMode) Update(msg tea.Msg) (Mode, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.Width = msg.Width - 4
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			// 执行命令
			command := strings.TrimSpace(m.input.Value())
			if command != "" {
				m.history = append(m.history, command)
				m.historyIndex = len(m.history)
				return m, m.executeCommand(command)
			}

		case "esc":
			// 返回搜索模式
			return m, func() tea.Msg {
				return SwitchModeMsg{ModeName: "search"}
			}

		case "up":
			// 历史命令上翻
			if len(m.history) > 0 && m.historyIndex > 0 {
				m.historyIndex--
				m.input.SetValue(m.history[m.historyIndex])
			}
			return m, nil

		case "down":
			// 历史命令下翻
			if len(m.history) > 0 && m.historyIndex < len(m.history)-1 {
				m.historyIndex++
				m.input.SetValue(m.history[m.historyIndex])
			} else {
				m.historyIndex = len(m.history)
				m.input.SetValue("")
			}
			return m, nil
		}
	}

	// 更新输入框
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// View 实现 Mode 接口
func (m *CommandMode) View() string {
	if m.width == 0 {
		return "初始化中..."
	}

	// 状态栏
	statusBar := m.renderStatusBar()

	// 中间空白区域 - 显示输出或帮助
	contentHeight := m.height - 6 // 减去状态栏和输入区域
	if contentHeight < 1 {
		contentHeight = 1
	}

	var content string
	if m.error != "" {
		// 显示错误
		errorStyle := lipgloss.NewStyle().
			Foreground(m.theme.Error).
			Bold(true)
		content = errorStyle.Render(m.error)
	} else if m.output != "" {
		// 显示输出
		content = m.output
	} else {
		// 显示默认提示
		content = m.renderHelp()
	}

	// 上部空白填充
	padding := strings.Repeat("\n", contentHeight/2)

	// 输入区域
	inputArea := m.renderInputArea()

	// 底部提示
	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left,
		statusBar,
		padding,
		content,
		padding,
		inputArea,
		footer,
	)
}

// HandleKey 实现 Mode 接口
func (m *CommandMode) HandleKey(key string) (bool, tea.Cmd) {
	// 命令模式的快捷键在Update中处理
	return false, nil
}

// executeCommand 执行命令
func (m *CommandMode) executeCommand(command string) tea.Cmd {
	// 移除开头的冒号（如果有）
	command = strings.TrimPrefix(command, ":")
	parts := strings.Fields(command)

	if len(parts) == 0 {
		return nil
	}

	cmdName := parts[0]

	switch cmdName {
	case "help", "?":
		m.output = m.renderCommandList()
		m.error = ""

	case "q", "quit":
		// 退出程序
		return tea.Quit

	case "search":
		if len(parts) > 1 {
			keyword := strings.Join(parts[1:], " ")
			// 发送搜索消息并切换到搜索模式
			return func() tea.Msg {
				return SearchWithKeywordMsg{Keyword: keyword}
			}
		}
		// 无参数则直接切换到搜索模式
		return func() tea.Msg {
			return SwitchModeMsg{ModeName: "search"}
		}

	case "project":
		if len(parts) > 1 {
			// TODO: 切换到指定项目
			m.output = fmt.Sprintf("切换到项目 %s (功能待实现)", parts[1])
		}
		// 切换到项目模式
		return func() tea.Msg {
			return SwitchModeMsg{ModeName: "project"}
		}

	case "set":
		if len(parts) < 2 {
			m.error = "错误: set 命令需要参数"
			return nil
		}
		m.handleSetCommand(parts[1:])

	default:
		m.error = fmt.Sprintf("未知命令: %s (输入 :help 查看可用命令)", cmdName)
	}

	return nil
}

// handleSetCommand 处理 set 命令
func (m *CommandMode) handleSetCommand(args []string) {
	if len(args) == 0 {
		m.error = "错误: set 命令需要参数"
		return
	}

	setting := args[0]
	switch setting {
	case "case":
		if len(args) < 2 {
			m.error = "错误: 需要指定 on 或 off"
			return
		}
		value := args[1]
		if value == "on" || value == "off" {
			m.output = fmt.Sprintf("设置大小写敏感: %s (功能待实现)", value)
			m.error = ""
		} else {
			m.error = "错误: case 参数必须是 on 或 off"
		}

	case "context":
		if len(args) < 2 {
			m.error = "错误: 需要指定上下文行数"
			return
		}
		if n, err := strconv.Atoi(args[1]); err == nil && n >= 0 {
			m.output = fmt.Sprintf("设置上下文行数: %d (功能待实现)", n)
			m.error = ""
		} else {
			m.error = "错误: context 参数必须是非负整数"
		}

	default:
		m.error = fmt.Sprintf("未知设置: %s", setting)
	}
}

// renderStatusBar 渲染状态栏
func (m *CommandMode) renderStatusBar() string {
	status := "[命令模式] 输入命令执行操作"

	statusStyle := lipgloss.NewStyle().
		Foreground(m.theme.Foreground).
		Background(m.theme.StatusBar).
		Padding(0, 1).
		Width(m.width)

	return statusStyle.Render(status)
}

// renderInputArea 渲染输入区域
func (m *CommandMode) renderInputArea() string {
	return "\n" + m.input.View() + "\n"
}

// renderFooter 渲染底部提示
func (m *CommandMode) renderFooter() string {
	hints := "Enter 执行 · ↑↓ 历史 · Esc 返回搜索"

	footerStyle := lipgloss.NewStyle().
		Foreground(m.theme.TextMuted).
		Padding(0, 1)

	return footerStyle.Render(hints)
}

// renderHelp 渲染帮助信息
func (m *CommandMode) renderHelp() string {
	helpStyle := lipgloss.NewStyle().
		Foreground(m.theme.TextMuted).
		Align(lipgloss.Center)

	help := `输入 :help 查看所有可用命令

示例: :search error, :project 1, :q`

	return helpStyle.Render(help)
}

// renderCommandList 渲染命令列表
func (m *CommandMode) renderCommandList() string {
	commands := []string{
		"可用命令:",
		"",
		"  :help, :?           显示此帮助",
		"  :q, :quit           退出程序",
		"  :search <keyword>   搜索关键词",
		"  :project <id>       切换到项目",
		"  :set case on/off    设置大小写敏感",
		"  :set context <n>    设置上下文行数",
		"",
		"快捷键:",
		"  Enter    执行命令",
		"  Esc      返回搜索模式",
		"  ↑/↓      浏览历史命令",
	}

	contentStyle := lipgloss.NewStyle().
		Foreground(m.theme.Foreground).
		Padding(1, 2)

	return contentStyle.Render(strings.Join(commands, "\n"))
}
