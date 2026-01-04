package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View 渲染当前界面。
func (m *Model) View() string {
	if !m.ready {
		return "TUI 初始化中..."
	}

	var body string
	switch m.state {
	case ProjectListView:
		body = m.renderProjectListWithStats()
	case HistoryListView:
		body = m.historyList.View()
	case LogViewerView:
		body = m.logViewer.View()
	case TaskListView:
		body = m.taskList.View()
	case SearchView:
		body = m.searchView.View()
	default:
		body = "未知视图"
	}

	var builder strings.Builder
	builder.WriteString(body)
	builder.WriteString("\n\n")
	builder.WriteString(m.renderStatusBar())

	if m.err != nil {
		builder.WriteString("\n")
		builder.WriteString(m.styles.Error.Render(fmt.Sprintf("错误: %v", m.err)))
	}

	return builder.String()
}

func (m *Model) renderStatusBar() string {
	status := fmt.Sprintf("当前视图: %s", m.stateLabel())
	help := "通用快捷键: tab 任务视图 · ctrl+f 搜索 · esc 返回 · ctrl+c 退出"
	return m.styles.StatusBar.Render(fmt.Sprintf("%s | %s", status, help))
}

func (m *Model) stateLabel() string {
	switch m.state {
	case ProjectListView:
		return "项目列表"
	case HistoryListView:
		return "历史记录"
	case LogViewerView:
		return "日志查看"
	case TaskListView:
		return "任务管理"
	case SearchView:
		return "全局搜索"
	default:
		return "未知"
	}
}

func (m *Model) renderProjectListWithStats() string {
	listView := m.projectList.View()
	statsView := m.statsPanel.View()
	if m.projectStatsCompact {
		compact := strings.TrimSpace(m.statsPanel.CompactView())
		if compact == "" {
			return listView
		}
		dividerWidth := m.width
		if dividerWidth <= 0 {
			dividerWidth = 60
		}
		divider := strings.Repeat("─", dividerWidth)
		return fmt.Sprintf("%s\n%s\n%s", compact, divider, listView)
	}
	if strings.TrimSpace(statsView) == "" {
		return listView
	}
	if m.projectSplitVertical {
		return lipgloss.JoinVertical(lipgloss.Left, listView, statsView)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, listView, statsView)
}
