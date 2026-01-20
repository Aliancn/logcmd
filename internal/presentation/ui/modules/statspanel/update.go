package statspanel

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/aliancn/logcmd/internal/platform/history"
)

// Update 处理消息
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)

	case SetProjectMsg:
		m.SetProject(msg.Project)
		return m, m.Refresh()

	case StatsLoadedMsg:
		if msg.ProjectID != m.currentProjectID() {
			break
		}
		m.commandDist = msg.CommandDist
		// 计算 Top 5 命令
		m.topCommands = calculateTopCommands(m.commandDist, 5)
		m.summary = msg.Summary
		m.failures = msg.RecentFailures
		m.lastUpdated = msg.GeneratedAt
		m.loading = false
		m.err = nil
	case statsFailedMsg:
		if msg.ProjectID != m.currentProjectID() {
			break
		}
		m.loading = false
		m.err = msg.Err
	}

	return m, nil
}

// LoadStatsCmd 加载统计数据
func (m Model) LoadStatsCmd() tea.Cmd {
	if m.historyMgr == nil {
		return nil
	}

	projectID := m.currentProjectID()

	return func() tea.Msg {
		// 查询项目的历史记录
		histories, err := m.historyMgr.Query(history.QueryOptions{
			ProjectID: projectID,
			Limit:     500, // 取最近 500 条
		})
		if err != nil {
			return statsFailedMsg{ProjectID: projectID, Err: fmt.Errorf("加载统计失败: %w", err)}
		}

		// 统计命令分布
		dist := make(map[string]int)
		var stats statsSummary
		var failures []recentFailure
		var durationTotal time.Duration

		for _, h := range histories {
			cmdName := normalizeCommandName(h.CommandName, h.Command)
			dist[cmdName]++

			stats.Total++
			if strings.EqualFold(h.Status, "success") {
				stats.Success++
			} else {
				stats.Failed++
				if len(failures) < 4 {
					failures = append(failures, recentFailure{
						ProjectID: h.ProjectID,
						Command:   cmdName,
						Status:    h.Status,
						ExitCode:  h.ExitCode,
						StartedAt: h.StartTime,
						Duration:  h.GetDuration(),
					})
				}
			}
			durationTotal += h.GetDuration()
		}

		if stats.Total > 0 {
			stats.AvgDuration = time.Duration(int64(durationTotal) / int64(stats.Total))
		}
		if len(histories) > 0 {
			stats.LastRun = histories[0].StartTime
		}

		return StatsLoadedMsg{
			ProjectID:      projectID,
			CommandDist:    dist,
			Summary:        stats,
			RecentFailures: failures,
			GeneratedAt:    time.Now(),
		}
	}
}

func normalizeCommandName(name, fallback string) string {
	cmdName := strings.TrimSpace(name)
	if cmdName == "" {
		cmdName = strings.TrimSpace(fallback)
		if len(cmdName) > 30 {
			cmdName = cmdName[:30]
		}
	}
	if cmdName == "" {
		return "未知命令"
	}
	return cmdName
}
