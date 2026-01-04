package statspanel

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aliancn/logcmd/internal/history"
	"github.com/aliancn/logcmd/internal/model"
	"github.com/aliancn/logcmd/internal/ui/common"
	"github.com/aliancn/logcmd/internal/ui/modules/projectlist"
)

const maxCommandRows = 5

type commandStat struct {
	Name  string
	Count int
}

type commandStatsLoadedMsg struct {
	ProjectID int
	Stats     []commandStat
}

type commandStatsErrorMsg struct {
	ProjectID int
	Err       error
}

// Model 负责展示项目统计信息。
type Model struct {
	history *history.Manager
	styles  common.Styles
	width   int
	height  int

	project      *model.Project
	loading      bool
	err          error
	commandStats []commandStat
}

// New 创建统计面板。
func New(historyMgr *history.Manager) Model {
	return Model{
		history: historyMgr,
		styles:  common.DefaultStyles(),
	}
}

// Init 实现 tea.Model 接口。
func (m Model) Init() tea.Cmd {
	return nil
}

// SetSize 更新视图尺寸。
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Update 响应消息。
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch typed := msg.(type) {
	case projectlist.ProjectHighlightedMsg:
		if m.setProject(typed.Project) && typed.Project != nil {
			return m, m.loadCommandStatsCmd(typed.Project.ID)
		}
	case commandStatsLoadedMsg:
		if m.project == nil || typed.ProjectID != m.project.ID {
			break
		}
		m.loading = false
		m.err = nil
		m.commandStats = typed.Stats
	case commandStatsErrorMsg:
		if m.project == nil || typed.ProjectID != m.project.ID {
			break
		}
		m.loading = false
		m.err = typed.Err
		m.commandStats = nil
	}
	return m, nil
}

// View 渲染统计面板。
func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}
	if m.project == nil {
		return m.styles.Frame.Render("选择项目以查看统计信息")
	}

	var builder strings.Builder
	builder.WriteString(m.styles.Title.Render("统计概览"))
	builder.WriteString("\n")
	builder.WriteString(m.renderSummary())
	builder.WriteString("\n\n")
	builder.WriteString(m.renderCommandDistribution())

	return m.styles.Frame.Copy().
		Width(m.width).
		Render(builder.String())
}

func (m *Model) setProject(project *model.Project) bool {
	if project == nil {
		m.project = nil
		m.commandStats = nil
		m.loading = false
		m.err = nil
		return false
	}
	if m.project != nil && project.ID == m.project.ID {
		m.project = project
		return false
	}
	m.project = project
	m.loading = true
	m.err = nil
	m.commandStats = nil
	return true
}

func (m Model) loadCommandStatsCmd(projectID int) tea.Cmd {
	if m.history == nil || projectID == 0 {
		return nil
	}
	return func() tea.Msg {
		var zero time.Time
		statsMap, err := m.history.GetCommandStats(projectID, zero, zero)
		if err != nil {
			return commandStatsErrorMsg{ProjectID: projectID, Err: err}
		}
		stats := toCommandStats(statsMap)
		return commandStatsLoadedMsg{ProjectID: projectID, Stats: stats}
	}
}

func (m Model) renderSummary() string {
	if m.project == nil {
		return ""
	}
	total := m.project.TotalCommands
	success := m.project.SuccessCommands
	failed := m.project.FailedCommands
	avg := m.project.GetAvgDuration()

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("项目: %s\n", displayName(m.project)))
	builder.WriteString(fmt.Sprintf("总运行: %d  成功: %d  失败: %d\n", total, success, failed))
	if avg > 0 {
		builder.WriteString(fmt.Sprintf("平均耗时: %s\n", avg.Truncate(time.Millisecond)))
	}
	builder.WriteString(m.renderSuccessBar())
	return builder.String()
}

func (m Model) renderSuccessBar() string {
	if m.project == nil || m.project.TotalCommands == 0 {
		return "成功率: 暂无执行数据"
	}
	width := m.barWidth()
	if width < 1 {
		width = 1
	}
	successRate := m.project.GetSuccessRate()
	bar := m.renderBar(width, successRate/100)
	return fmt.Sprintf("成功率: %5.1f%% %s", successRate, bar)
}

func (m Model) renderCommandDistribution() string {
	var builder strings.Builder
	builder.WriteString("命令分布:\n")

	switch {
	case m.loading:
		builder.WriteString("  正在加载命令统计...\n")
	case m.err != nil:
		builder.WriteString(fmt.Sprintf("  加载失败: %v\n", m.err))
	case len(m.commandStats) == 0:
		builder.WriteString("  暂无历史数据\n")
	default:
		maxCount := m.commandStats[0].Count
		if maxCount <= 0 {
			maxCount = 1
		}
		barWidth := m.barWidth()
		nameWidth := clamp(barWidth/2, 12, 24)
		for _, stat := range m.commandStats {
			portion := 0.0
			if maxCount > 0 {
				portion = float64(stat.Count) / float64(maxCount)
			}
			bar := m.renderBar(barWidth, portion)
			builder.WriteString(fmt.Sprintf("  %-*s %4d %s\n",
				nameWidth,
				truncate(stat.Name, nameWidth),
				stat.Count,
				bar,
			))
		}
	}

	return builder.String()
}

func (m Model) barWidth() int {
	width := m.width - 12
	if width < 10 {
		width = 10
	}
	if width > 40 {
		width = 40
	}
	return width
}

func (m Model) renderBar(width int, ratio float64) string {
	if width <= 0 {
		width = 1
	}
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	filled := int(math.Round(ratio * float64(width)))
	if filled > width {
		filled = width
	}
	empty := width - filled
	if filled == 0 && ratio > 0 {
		filled = 1
		empty--
	}
	filledPart := lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Render(strings.Repeat("█", filled))
	if empty < 0 {
		empty = 0
	}
	emptyPart := lipgloss.NewStyle().
		Foreground(lipgloss.Color("236")).
		Render(strings.Repeat("░", empty))
	return lipgloss.NewStyle().
		Width(width).
		Render(filledPart + emptyPart)
}

// CompactView 返回文本简报。
func (m Model) CompactView() string {
	if m.loading {
		return "统计: 正在加载..."
	}
	if m.err != nil {
		return fmt.Sprintf("统计: 加载失败 - %v", m.err)
	}
	if m.project == nil {
		return ""
	}
	lines := []string{
		fmt.Sprintf("项目: %s (#%d)", displayName(m.project), m.project.ID),
		fmt.Sprintf("总运行 %d · 成功 %d · 失败 %d · 成功率 %.1f%%",
			m.project.TotalCommands,
			m.project.SuccessCommands,
			m.project.FailedCommands,
			m.project.GetSuccessRate(),
		),
	}
	if len(m.commandStats) > 0 {
		lines = append(lines, "常用命令:")
		for _, stat := range m.commandStats {
			lines = append(lines, fmt.Sprintf("  - %s × %d", stat.Name, stat.Count))
		}
	}
	return strings.Join(lines, "\n")
}

func toCommandStats(stats map[string]int) []commandStat {
	if len(stats) == 0 {
		return nil
	}
	list := make([]commandStat, 0, len(stats))
	for name, count := range stats {
		label := strings.TrimSpace(name)
		if label == "" {
			label = "(未命名命令)"
		}
		list = append(list, commandStat{Name: label, Count: count})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Count == list[j].Count {
			return list[i].Name < list[j].Name
		}
		return list[i].Count > list[j].Count
	})
	if len(list) > maxCommandRows {
		list = list[:maxCommandRows]
	}
	return list
}

func displayName(project *model.Project) string {
	if project == nil {
		return "-"
	}
	if name := strings.TrimSpace(project.Name); name != "" {
		return name
	}
	return project.Path
}

func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= width {
		return s
	}
	runes := []rune(s)
	if width == 1 {
		return string(runes[:1])
	}
	return string(runes[:width-1]) + "…"
}

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
