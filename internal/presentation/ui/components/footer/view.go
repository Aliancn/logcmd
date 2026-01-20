package footer

import (
	"fmt"
	"strings"

	"github.com/aliancn/logcmd/internal/presentation/ui/common"
)

// View 渲染Footer
func (m Model) View() string {
	if m.width == 0 {
		return ""
	}

	// 构建快捷键提示字符串
	var hints []string
	for _, hint := range m.keyHints {
		// 统一使用黑字，不使用 Highlight/Normal 区分以保持简约
		hints = append(hints, fmt.Sprintf("%s %s", hint.Key, hint.Desc))
	}

	content := strings.Join(hints, "  ·  ")

	// 使用StatusBar样式，确保宽度占满，去掉多余填充
	return m.styles.StatusBar.Copy().
		Width(m.width).
		Render(content)
}

// RenderWithCustomHints 使用自定义提示渲染Footer
func (m Model) RenderWithCustomHints(hints []common.KeyHint) string {
	tempModel := m
	tempModel.keyHints = hints
	return tempModel.View()
}
