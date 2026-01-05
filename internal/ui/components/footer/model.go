package footer

import "github.com/aliancn/logcmd/internal/ui/common"

// Model Footer组件的Model
type Model struct {
	keyHints []common.KeyHint // 快捷键提示列表
	width    int              // 组件宽度
	theme    common.Theme
	styles   common.Styles
}

// New 创建Footer Model
func New(theme common.Theme, styles common.Styles) Model {
	// 默认全局快捷键提示
	defaultHints := []common.KeyHint{
		{Key: "▲▼", Desc: "移动"},
		{Key: "Enter", Desc: "选择"},
		{Key: "1-4", Desc: "切换Tab"},
		{Key: "Ctrl+P", Desc: "命令"},
		{Key: "Ctrl+C", Desc: "退出"},
	}

	return Model{
		keyHints: defaultHints,
		theme:    theme,
		styles:   styles,
	}
}

// SetSize 设置Footer尺寸
func (m *Model) SetSize(width int) {
	m.width = width
}

// SetHints 设置快捷键提示
func (m *Model) SetHints(hints []common.KeyHint) {
	m.keyHints = hints
}

// AddHint 添加快捷键提示
func (m *Model) AddHint(key, desc string) {
	m.keyHints = append(m.keyHints, common.KeyHint{Key: key, Desc: desc})
}

// ClearHints 清空快捷键提示
func (m *Model) ClearHints() {
	m.keyHints = []common.KeyHint{}
}
