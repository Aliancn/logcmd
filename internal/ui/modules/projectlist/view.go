package projectlist

import (
	"fmt"
	"strings"
	"time"

	"github.com/aliancn/logcmd/internal/model"
)

// View 渲染列表
func (m Model) View() string {
	if m.isAdding {
		return m.panel.Render(m.renderAddForm())
	}

	if len(m.projects) == 0 {
		return m.panel.RenderEmpty("暂无项目\n按 'a' 添加项目")
	}
	return m.panel.Render(m.list.View())
}

// renderAddForm 渲染添加项目表单
func (m Model) renderAddForm() string {
	var b strings.Builder

	b.WriteString("添加新项目\n\n")

	// 路径输入
	b.WriteString("项目路径 (绝对路径):\n")
	b.WriteString(m.pathInput.View())
	b.WriteString("\n\n")

	// 名称输入
	b.WriteString("项目名称 (可选):\n")
	b.WriteString(m.nameInput.View())
	b.WriteString("\n\n")

	b.WriteString("Enter 确认 · Esc 取消")

	return b.String()
}

// projectItem 实现 list.Item 接口
type projectItem struct {
	project *model.Project
}

func (i projectItem) Title() string {
	p := i.project

	category := ""
	if p.Category != "" {
		category = fmt.Sprintf(" (%s)", p.Category)
	}

	return fmt.Sprintf("%s%s",
		p.Name,
		category,
	)
}

func (i projectItem) Description() string {
	p := i.project
	var parts []string

	// 1. 路径
	parts = append(parts, p.Path)

	// 2. 最后执行 (相对时间)
	lastExec := "从不"
	if p.LastCommandTime.Valid {
		lastExec = formatRelativeTime(p.LastCommandTime.Time)
	}
	parts = append(parts, fmt.Sprintf("Last: %s", lastExec))

	// 3. 统计
	parts = append(parts, fmt.Sprintf("Run: %d (%.0f%%)", p.TotalCommands, p.GetSuccessRate()))

	// 4. Tags
	if len(p.Tags) > 0 {
		tags := strings.Join(p.Tags, ",")
		if len(tags) > 15 {
			tags = tags[:12] + "..."
		}
		parts = append(parts, fmt.Sprintf("[%s]", tags))
	}

	return strings.Join(parts, " | ")
}

func (i projectItem) FilterValue() string {
	return i.project.Name + " " + i.project.Path + " " + i.project.Category + " " + strings.Join(i.project.Tags, " ")
}

// formatRelativeTime 格式化相对时间
func formatRelativeTime(t time.Time) string {
	diff := time.Since(t)
	if diff < time.Minute {
		return "刚刚"
	} else if diff < time.Hour {
		return fmt.Sprintf("%d分前", int(diff.Minutes()))
	} else if diff < 24*time.Hour {
		return fmt.Sprintf("%d小时前", int(diff.Hours()))
	} else if diff < 48*time.Hour {
		return "昨天"
	}
	return t.Format("01-02")
}
