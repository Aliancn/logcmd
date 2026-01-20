package footer

import "github.com/aliancn/logcmd/internal/presentation/ui/common"

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
	defaultHints := common.DefaultFooterHints()

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
