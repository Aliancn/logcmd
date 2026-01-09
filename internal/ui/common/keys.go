package common

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
)

// GlobalKeyMap 定义全局快捷键。
type GlobalKeyMap struct {
	Quit   key.Binding
	Back   key.Binding
	Search key.Binding
}

// NewGlobalKeyMap 初始化默认快捷键。
func NewGlobalKeyMap() GlobalKeyMap {
	return GlobalKeyMap{
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c", "q"),
			key.WithHelp("ctrl+c/q", "退出"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "返回"),
		),
		Search: key.NewBinding(
			key.WithKeys("ctrl+f"),
			key.WithHelp("ctrl+f", "全局搜索"),
		),
	}
}

// ShortHelp 返回展示用快捷键列表。
func (k GlobalKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Back, k.Search, k.Quit}
}

// FormatKeyHelp 返回 binding 的“键 描述”字符串。
func FormatKeyHelp(binding key.Binding) string {
	help := binding.Help()
	keyStr := strings.TrimSpace(help.Key)
	desc := strings.TrimSpace(help.Desc)
	switch {
	case keyStr == "" && desc == "":
		return ""
	case keyStr == "":
		return desc
	case desc == "":
		return keyStr
	default:
		return fmt.Sprintf("%s %s", keyStr, desc)
	}
}

// JoinKeyHelps 拼接多个快捷键描述，自动跳过空项。
func JoinKeyHelps(hints ...string) string {
	var filtered []string
	for _, h := range hints {
		if strings.TrimSpace(h) != "" {
			filtered = append(filtered, h)
		}
	}
	return strings.Join(filtered, " · ")
}

// DefaultFooterHints 返回默认的全局快捷键提示。
func DefaultFooterHints() []KeyHint {
	return GlobalFooterHints(NewGlobalKeyMap())
}

// GlobalFooterHints 根据给定的全局按键映射返回Footer提示。
func GlobalFooterHints(keys GlobalKeyMap) []KeyHint {
	hints := KeyHintsFromBindings(keys.Back, keys.Search)
	hints = append(hints,
		KeyHint{Key: "1-4", Desc: "切换Tab"},
		KeyHint{Key: "Ctrl+P", Desc: "命令面板"},
	)
	hints = append(hints, KeyHintsFromBindings(keys.Quit)...)
	return hints
}

// KeyHintFromBinding 将 key.Binding 的帮助信息转换为 KeyHint。
func KeyHintFromBinding(binding key.Binding) (KeyHint, bool) {
	help := binding.Help()
	keyStr := strings.TrimSpace(help.Key)
	desc := strings.TrimSpace(help.Desc)
	if keyStr == "" && desc == "" {
		return KeyHint{}, false
	}
	return KeyHint{Key: keyStr, Desc: desc}, true
}

// KeyHintsFromBindings 批量转换 key.Binding 为 KeyHint。
func KeyHintsFromBindings(bindings ...key.Binding) []KeyHint {
	var hints []KeyHint
	for _, b := range bindings {
		if hint, ok := KeyHintFromBinding(b); ok {
			hints = append(hints, hint)
		}
	}
	return hints
}