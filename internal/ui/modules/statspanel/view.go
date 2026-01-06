package statspanel

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/aliancn/logcmd/internal/ui/common"
)

// View 渲染视图
func (m Model) View() string {
	m.panel.SetHeader(m.renderHeader())

	switch {
	case m.historyMgr == nil:
		return m.panel.RenderEmpty("历史记录不可用")
	case m.loading:
		return m.panel.Render(m.styles.Muted.Render("正在加载统计数据..."))
	case m.err != nil:
		return m.panel.Render(m.styles.Error.Render(fmt.Sprintf("加载失败: %v", m.err)))
	}

	var sections []string

	if m.summary.Total == 0 && len(m.topCommands) == 0 {
		return m.panel.RenderEmpty("暂无统计数据")
	}

	if summary := m.renderSummarySection(); summary != "" {
		sections = append(sections, summary)
	}

	if failures := m.renderFailuresSection(); failures != "" {
		sections = append(sections, failures)
	}

	if commands := m.renderCommandSection(); commands != "" {
		sections = append(sections, commands)
	}

	content := strings.Join(sections, "\n\n")
	return m.panel.Render(content)
}

func (m Model) renderSummarySection() string {
	title := lipgloss.NewStyle().
		Foreground(m.theme.Primary).
		Bold(true).
		Render(m.summaryTitle())

	if m.summary.Total == 0 {
		return title + "\n" + m.styles.Muted.Render("暂无执行记录")
	}

	metrics := []struct {
		label string
		value string
		style lipgloss.Style
	}{
		{"总执行", fmt.Sprintf("%d", m.summary.Total), lipgloss.NewStyle().Foreground(m.theme.Foreground)},
		{"成功", fmt.Sprintf("%d", m.summary.Success), lipgloss.NewStyle().Foreground(m.theme.Success)},
		{"失败", fmt.Sprintf("%d", m.summary.Failed), lipgloss.NewStyle().Foreground(m.theme.Error)},
		{"成功率", fmt.Sprintf("%.1f%%", m.summary.successRate()), lipgloss.NewStyle().Foreground(m.theme.Primary)},
		{"平均耗时", formatDurationShort(m.summary.AvgDuration), lipgloss.NewStyle().Foreground(m.theme.TextMuted)},
		{"最近执行", formatTimeShort(m.summary.LastRun), lipgloss.NewStyle().Foreground(m.theme.TextMuted)},
	}

	var lines []string
	for i := 0; i < len(metrics); i += 3 {
		end := i + 3
		if end > len(metrics) {
			end = len(metrics)
		}
		chunk := metrics[i:end]
		var parts []string
		for _, metric := range chunk {
			label := lipgloss.NewStyle().Foreground(m.theme.TextMuted).Render(metric.label + ":")
			value := metric.style.Bold(true).Render(metric.value)
			parts = append(parts, lipgloss.NewStyle().Width(20).Render(fmt.Sprintf("%s %s", label, value)))
		}
		lines = append(lines, strings.Join(parts, "  "))
	}

	body := strings.Join(lines, "\n")
	return fmt.Sprintf("%s\n%s", title, body)
}

func (m Model) renderFailuresSection() string {
	if len(m.failures) == 0 {
		return ""
	}

	title := lipgloss.NewStyle().
		Foreground(m.theme.Warning).
		Bold(true).
		Render("最近失败")

	var rows []string
	for _, f := range m.failures {
		project := fmt.Sprintf("项目#%d", f.ProjectID)
		cmd := lipgloss.NewStyle().Bold(true).Render(f.Command)
		status := lipgloss.NewStyle().Foreground(m.theme.Error).Render(strings.ToUpper(f.Status))
		meta := fmt.Sprintf("%s · %s · 耗时 %s · 退出码 %d",
			project,
			formatTimeShort(f.StartedAt),
			formatDurationShort(f.Duration),
			f.ExitCode,
		)
		rows = append(rows, fmt.Sprintf("%s  %s\n%s", status, cmd, lipgloss.NewStyle().Foreground(m.theme.TextMuted).Render(meta)))
	}

	return fmt.Sprintf("%s\n%s", title, strings.Join(rows, "\n\n"))
}

func (m Model) renderCommandSection() string {
	if len(m.topCommands) == 0 {
		return ""
	}

	header := lipgloss.NewStyle().
		Foreground(m.theme.Primary).
		Bold(true).
		Render("常用命令")

	chartWidth := m.width - 8
	if chartWidth < 20 {
		chartWidth = 20
	}
	chart := renderBarChart(m.topCommands, chartWidth, m.theme)

	return fmt.Sprintf("%s\n\n%s", header, chart)
}

func (m Model) summaryTitle() string {
	if m.project != nil {
		return fmt.Sprintf("项目 #%d 运行状态", m.project.ID)
	}
	return "全局运行状态"
}

func (m statsSummary) successRate() float64 {
	if m.Total == 0 {
		return 0
	}
	return float64(m.Success) / float64(m.Total) * 100
}

func (m Model) renderHeader() string {
	var title string
	if m.project != nil {
		title = fmt.Sprintf("统计 · 项目 #%d", m.project.ID)
	} else {
		title = "统计 · 全局"
	}
	if !m.lastUpdated.IsZero() {
		title = fmt.Sprintf("%s · 更新 %s", title, m.lastUpdated.Format("15:04:05"))
	}
	return title
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

func formatDurationShort(d time.Duration) string {
	if d <= 0 {
		return "-"
	}
	if d >= time.Second {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%dms", d.Milliseconds())
}

func formatTimeShort(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("01-02 15:04")
}
