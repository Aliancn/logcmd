package statspanel

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/aliancn/logcmd/internal/ui/common"
)

// View 渲染视图
func (m Model) View() string {
	if m.project == nil {
		return m.panel.RenderEmpty("选择一个项目查看统计")
	}

	var content strings.Builder

	// 标题
	titleStyle := lipgloss.NewStyle().
		Foreground(m.theme.Primary).
		Bold(true).
		PaddingBottom(1)
	content.WriteString(titleStyle.Render(fmt.Sprintf("项目 #%d 统计", m.project.ID)))
	content.WriteString("\n\n")

	// 基本统计
	successRate := m.project.GetSuccessRate()
	successRateStyle := m.getSuccessRateStyle(successRate)

	stats := []struct {
		Label string
		Value string
		Style lipgloss.Style
	}{
		{"成功率", fmt.Sprintf("%.1f%%", successRate), successRateStyle},
		{"总执行", fmt.Sprintf("%d", m.project.TotalCommands), lipgloss.NewStyle().Foreground(m.theme.Foreground)},
		{"成功", fmt.Sprintf("%d", m.project.SuccessCommands), lipgloss.NewStyle().Foreground(m.theme.Success)},
		{"失败", fmt.Sprintf("%d", m.project.FailedCommands), lipgloss.NewStyle().Foreground(m.theme.Error)},
	}

	for _, stat := range stats {
		labelStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted)
		line := fmt.Sprintf("%s: %s",
			labelStyle.Render(stat.Label),
			stat.Style.Render(stat.Value),
		)
		content.WriteString(line)
		content.WriteString("\n")
	}

	// 命令分布
	if len(m.topCommands) > 0 {
		content.WriteString("\n")
		divider := lipgloss.NewStyle().
			Foreground(m.theme.Border).
			Render(strings.Repeat("─", m.width-4))
		content.WriteString(divider)
		content.WriteString("\n\n")

		headerStyle := lipgloss.NewStyle().
			Foreground(m.theme.Primary).
			Bold(true)
		content.WriteString(headerStyle.Render("常用命令"))
		content.WriteString("\n\n")

		// ASCII bar chart
		chartWidth := m.width - 8 // 减去 padding 和边框
		if chartWidth < 20 {
			chartWidth = 20
		}
		chart := renderBarChart(m.topCommands, chartWidth, m.theme)
		content.WriteString(chart)
	}

	return m.panel.Render(content.String())
}

// getSuccessRateStyle 根据成功率返回样式
func (m Model) getSuccessRateStyle(rate float64) lipgloss.Style {
	if rate >= 80.0 {
		return lipgloss.NewStyle().Foreground(m.theme.Success)
	} else if rate >= 50.0 {
		return lipgloss.NewStyle().Foreground(m.theme.Warning)
	}
	return lipgloss.NewStyle().Foreground(m.theme.Error)
}

// renderBarChart 渲染 ASCII bar chart
func renderBarChart(stats []CommandStat, maxWidth int, theme common.Theme) string {
	if len(stats) == 0 {
		return ""
	}

	// 找到最大值用于归一化
	maxCount := 0
	for _, stat := range stats {
		if stat.Count > maxCount {
			maxCount = stat.Count
		}
	}

	var chart strings.Builder
	maxLabelLen := 20                          // 命令名最大长度
	barMaxWidth := maxWidth - maxLabelLen - 10 // 减去标签和计数显示

	if barMaxWidth < 10 {
		barMaxWidth = 10
	}

	for _, stat := range stats {
		// 截断命令名
		cmd := stat.Command
		if len(cmd) > maxLabelLen {
			cmd = cmd[:maxLabelLen-3] + "..."
		}

		// 计算 bar 长度
		barLen := 0
		if maxCount > 0 {
			barLen = (stat.Count * barMaxWidth) / maxCount
		}
		if barLen < 1 && stat.Count > 0 {
			barLen = 1
		}

		// 渲染行
		labelStyle := lipgloss.NewStyle().
			Width(maxLabelLen).
			Foreground(theme.Foreground)

		barStyle := lipgloss.NewStyle().
			Foreground(theme.Success)
		bar := barStyle.Render(strings.Repeat("█", barLen))

		countStyle := lipgloss.NewStyle().
			Foreground(theme.TextMuted)

		line := fmt.Sprintf("%s %s %s",
			labelStyle.Render(cmd),
			bar,
			countStyle.Render(fmt.Sprintf("(%d)", stat.Count)),
		)

		chart.WriteString(line)
		chart.WriteString("\n")
	}

	return chart.String()
}

// calculateTopCommands 计算 Top N 命令
func calculateTopCommands(dist map[string]int, n int) []CommandStat {
	stats := make([]CommandStat, 0, len(dist))
	for cmd, count := range dist {
		stats = append(stats, CommandStat{
			Command: cmd,
			Count:   count,
		})
	}

	// 按计数降序排序
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Count > stats[j].Count
	})

	// 取前 N 个
	if len(stats) > n {
		stats = stats[:n]
	}

	return stats
}
