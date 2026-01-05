package statspanel

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/aliancn/logcmd/internal/history"
	"github.com/aliancn/logcmd/internal/ui/common"
)

// Update 处理消息
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)

	case SetProjectMsg:
		m.SetProject(msg.Project)
		return m, m.LoadStatsCmd()

	case StatsLoadedMsg:
		if m.project == nil || msg.ProjectID != m.project.ID {
			break
		}
		m.commandDist = msg.CommandDist
		// 计算 Top 5 命令
		m.topCommands = calculateTopCommands(m.commandDist, 5)
	}

	return m, nil
}

// LoadStatsCmd 加载统计数据
func (m Model) LoadStatsCmd() tea.Cmd {
	if m.project == nil || m.historyMgr == nil {
		return nil
	}

	projectID := m.project.ID

	return func() tea.Msg {
		// 查询项目的历史记录
		histories, err := m.historyMgr.Query(history.QueryOptions{
			ProjectID: projectID,
			Limit:     500, // 取最近 500 条
		})
		if err != nil {
			return common.ErrorMsg{Err: fmt.Errorf("加载统计失败: %w", err)}
		}

		// 统计命令分布
		dist := make(map[string]int)
		for _, h := range histories {
			cmdName := strings.TrimSpace(h.CommandName)
			if cmdName == "" {
				// 如果没有 CommandName，使用完整命令
				cmdName = h.Command
				// 截取前 30 个字符作为命令名
				if len(cmdName) > 30 {
					cmdName = cmdName[:30]
				}
			}
			dist[cmdName]++
		}

		return StatsLoadedMsg{
			ProjectID:   projectID,
			CommandDist: dist,
		}
	}
}
