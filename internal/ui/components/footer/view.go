package footer

import (
	"fmt"
	"strings"

	"github.com/aliancn/logcmd/internal/ui/common"
)

// View 渲染Footer
func (m Model) View() string {
	if m.width == 0 {
		return ""
	}

	// 构建快捷键提示字符串
	var hints []string
	for _, hint := range m.keyHints {
		// 格式：快捷键 描述
		keyStyle := m.styles.Highlight.Render(hint.Key)
		descStyle := m.styles.Normal.Render(hint.Desc)
		hints = append(hints, fmt.Sprintf("%s %s", keyStyle, descStyle))
	}

	content := strings.Join(hints, " · ")

	// 使用StatusBar样式，确保宽度占满
	footerStyle := m.styles.StatusBar.Width(m.width - 2) // 减去padding

	return footerStyle.Render(content)
}

// RenderWithCustomHints 使用自定义提示渲染Footer
func (m Model) RenderWithCustomHints(hints []common.KeyHint) string {
	tempModel := m
	tempModel.keyHints = hints
	return tempModel.View()
}
