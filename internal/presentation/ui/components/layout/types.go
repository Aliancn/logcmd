package layout

import tea "github.com/charmbracelet/bubbletea"

// Resizable 接口定义了支持动态调整大小的组件
type Resizable interface {
	tea.Model
	SetSize(width, height int)
}
