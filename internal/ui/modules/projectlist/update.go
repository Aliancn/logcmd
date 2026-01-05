package projectlist

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aliancn/logcmd/internal/ui/common"
)

// Update 处理消息
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)

	case tea.KeyMsg:
		if m.confirmingDelete {
			// 处理删除确认
			return m.handleDeleteConfirmation(msg)
		}

		// 正常键盘处理
		switch {
		case key.Matches(msg, m.keys.Select):
			if item, ok := m.list.SelectedItem().(projectItem); ok {
				cmds = append(cmds, func() tea.Msg {
					return ProjectSelectedMsg{Project: item.project}
				})
			}

		case key.Matches(msg, m.keys.Delete):
			if item, ok := m.list.SelectedItem().(projectItem); ok {
				m.confirmingDelete = true
				m.deleteTargetID = item.project.ID
				m.statusMsg = fmt.Sprintf("确认删除项目 #%d \"%s\"? [y/n]",
					item.project.ID, item.project.Name)
				m.SetSize(m.width, m.height) // 更新 footer
			}

		case key.Matches(msg, m.keys.Refresh):
			cmds = append(cmds, m.LoadProjectsCmd())
		}

	case ProjectsLoadedMsg:
		m.projects = msg.Projects
		items := make([]list.Item, len(msg.Projects))
		for i, proj := range msg.Projects {
			items[i] = projectItem{project: proj}
		}
		m.list.SetItems(items)
		m.statusMsg = fmt.Sprintf("已加载 %d 个项目", len(msg.Projects))
		m.SetSize(m.width, m.height)
	}

	// 更新 list
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// handleDeleteConfirmation 处理删除确认
func (m Model) handleDeleteConfirmation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Confirm):
		// 确认删除
		m.confirmingDelete = false
		m.statusMsg = fmt.Sprintf("正在删除项目 #%d...", m.deleteTargetID)
		m.SetSize(m.width, m.height)
		return m, m.deleteProjectCmd(m.deleteTargetID)

	case key.Matches(msg, m.keys.Cancel):
		// 取消删除
		m.confirmingDelete = false
		m.deleteTargetID = 0
		m.statusMsg = "已取消删除"
		m.SetSize(m.width, m.height)
	}

	return m, nil
}

// LoadProjectsCmd 触发项目列表加载
func (m Model) LoadProjectsCmd() tea.Cmd {
	if m.registry == nil {
		return nil
	}

	return func() tea.Msg {
		projects, err := m.registry.List()
		if err != nil {
			return common.ErrorMsg{Err: fmt.Errorf("加载项目列表失败: %w", err)}
		}
		return ProjectsLoadedMsg{Projects: projects}
	}
}

// deleteProjectCmd 执行项目删除
func (m Model) deleteProjectCmd(projectID int) tea.Cmd {
	return func() tea.Msg {
		err := m.registry.Delete(fmt.Sprintf("%d", projectID))
		if err != nil {
			return common.ErrorMsg{Err: fmt.Errorf("删除项目失败: %w", err)}
		}
		return ProjectDeletedMsg{ProjectID: projectID}
	}
}
