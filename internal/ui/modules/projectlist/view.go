package projectlist

import (
	"fmt"
	"strings"

	"github.com/aliancn/logcmd/internal/model"
)

// View 渲染列表
func (m Model) View() string {
	if len(m.projects) == 0 {
		return m.panel.RenderEmpty("暂无项目\n使用 'logcmd config add' 添加项目")
	}
	return m.panel.Render(m.list.View())
}

// projectItem 实现 list.Item 接口
type projectItem struct {
	project *model.Project
}

func (i projectItem) Title() string {
	p := i.project
	icon := getStatusIcon(p.GetSuccessRate())
	successRate := fmt.Sprintf("%.1f%%", p.GetSuccessRate())

	return fmt.Sprintf("%s [%d] %-20s  (%s)",
		icon,
		p.ID,
		truncateString(p.Name, 20),
		successRate,
	)
}

func (i projectItem) Description() string {
	p := i.project
	var parts []string

	// 项目路径
	parts = append(parts, p.Path)

	// 最后执行时间
	lastExec := "-"
	if p.LastCommandTime.Valid {
		lastExec = p.LastCommandTime.Time.Format("2006-01-02 15:04:05")
	}
	parts = append(parts, fmt.Sprintf("最后执行: %s", lastExec))

	// 总命令数
	parts = append(parts, fmt.Sprintf("总命令: %d", p.TotalCommands))

	return strings.Join(parts, " | ")
}

func (i projectItem) FilterValue() string {
	return i.project.Name + " " + i.project.Path
}

// getStatusIcon 根据成功率获取图标
func getStatusIcon(successRate float64) string {
	if successRate >= 80.0 {
		return "✅"
	} else if successRate >= 50.0 {
		return "⚠️"
	}
	return "❌"
}

// truncateString 截断字符串，如果超过最大长度则添加 ...
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return "..."
	}
	return s[:maxLen-3] + "..."
}
