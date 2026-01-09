package tabbar

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aliancn/logcmd/internal/ui/common"
)

// Tab 单个Tab的定义
type Tab struct {
	Index int    // Tab索引（0-3，共4个Tab）
	Key   string // 快捷键（"1", "2", "3", "4"）
	Label string // 显示标签
}

// Model TabBar组件的Model
type Model struct {
	tabs        []Tab // Tab列表
	activeIndex int   // 当前激活的Tab索引
	width       int   // 组件宽度
	theme       common.Theme
	styles      common.Styles
}

// New 创建TabBar Model
func New(theme common.Theme, styles common.Styles) Model {
	// 定义4个Tab
	tabs := []Tab{
		{Index: 0, Key: "1", Label: "项目"},
		{Index: 1, Key: "2", Label: "任务"},
		{Index: 2, Key: "3", Label: "搜索"},
		{Index: 3, Key: "4", Label: "统计"},
	}

	return Model{
		tabs:        tabs,
		activeIndex: 0, // 默认激活第一个Tab
		theme:       theme,
		styles:      styles,
	}
}

// SetSize 设置TabBar尺寸
func (m *Model) SetSize(width int) {
	m.width = width
}

// SetActive 设置当前激活的Tab
func (m *Model) SetActive(index int) {
	if index >= 0 && index < len(m.tabs) {
		m.activeIndex = index
	}
}

// GetActive 获取当前激活的Tab索引
func (m Model) GetActive() int {
	return m.activeIndex
}

// GetActiveTab 获取当前激活的Tab
func (m Model) GetActiveTab() Tab {
	if m.activeIndex >= 0 && m.activeIndex < len(m.tabs) {
		return m.tabs[m.activeIndex]
	}
	return m.tabs[0]
}

// Update TabBar的Update方法（用于处理数字键切换）
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.KeyMsg:
		// 处理数字键1-4切换Tab
		switch typed.String() {
		case "1":
			m.activeIndex = 0
			return m, nil
		case "2":
			m.activeIndex = 1
			return m, nil
		case "3":
			m.activeIndex = 2
			return m, nil
		case "4":
			m.activeIndex = 3
			return m, nil
		}
	}

	return m, nil
}
