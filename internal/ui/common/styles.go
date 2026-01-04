package common

import "github.com/charmbracelet/lipgloss"

// Styles 统一管理 UI 样式。
type Styles struct {
	Title     lipgloss.Style
	Subtitle  lipgloss.Style
	Error     lipgloss.Style
	Frame     lipgloss.Style
	StatusBar lipgloss.Style
}

// DefaultStyles 返回默认样式。
func DefaultStyles() Styles {
	return Styles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("213")).
			PaddingBottom(1),
		Subtitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("103")).
			PaddingBottom(1),
		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff6b6b")).
			Bold(true),
		Frame: lipgloss.NewStyle().
			Margin(1, 2),
		StatusBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#d9d9d9")).
			Background(lipgloss.Color("#44475a")).
			Padding(0, 1),
	}
}
