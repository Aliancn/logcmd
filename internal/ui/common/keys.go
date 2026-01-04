package common

import "github.com/charmbracelet/bubbles/key"

// GlobalKeyMap 定义全局快捷键。
type GlobalKeyMap struct {
	Quit key.Binding
	Back key.Binding
	Task key.Binding
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
		Task: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "任务视图"),
		),
	}
}

// ShortHelp 返回展示用快捷键列表。
func (k GlobalKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Back, k.Task, k.Quit}
}
