package projectlist

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aliancn/logcmd/internal/ui/common"
)

// Update 处理消息
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.isAdding {
			return m.handleAddProject(msg)
		}

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

		case key.Matches(msg, m.keys.Add):
			m.isAdding = true
			m.addStep = 0
			m.pathInput.Focus()
			m.nameInput.Blur()
			m.pathInput.SetValue("")
			m.nameInput.SetValue("")
			m.SetSize(m.width, m.height) // 更新 footer
			return m, textinput.Blink

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
		if m.statusMsg == "" || strings.HasPrefix(m.statusMsg, "已加载") {
			m.statusMsg = fmt.Sprintf("已加载 %d 个项目", len(msg.Projects))
		}
		m.SetSize(m.width, m.height)

	case ProjectCreatedMsg:
		m.isAdding = false
		m.statusMsg = fmt.Sprintf("项目已创建: %s", msg.Name)
		cmds = append(cmds, m.LoadProjectsCmd())
	}

	// 仅在非添加模式下更新 list
	if !m.isAdding {
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		// 添加模式下更新 inputs
		m.pathInput, cmd = m.pathInput.Update(msg)
		cmds = append(cmds, cmd)
		m.nameInput, cmd = m.nameInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// handleAddProject 处理添加项目逻辑
func (m Model) handleAddProject(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.isAdding = false
		m.statusMsg = "已取消添加"
		m.SetSize(m.width, m.height)
		return m, nil

	case msg.Type == tea.KeyTab:
		// 切换焦点
		if m.addStep == 0 {
			m.addStep = 1
			m.pathInput.Blur()
			m.nameInput.Focus()
		} else {
			m.addStep = 0
			m.nameInput.Blur()
			m.pathInput.Focus()
		}
		return m, nil

	case key.Matches(msg, m.keys.Confirm):
		if m.addStep == 0 {
			// 如果在路径输入框按回车，跳到名称输入框
			if m.pathInput.Value() != "" {
				m.addStep = 1
				m.pathInput.Blur()
				m.nameInput.Focus()
				return m, nil
			}
		}
		// 提交
		return m.submitAddProject()
	}

	// 更新 inputs
	var cmd tea.Cmd
	var cmds []tea.Cmd
	
m.pathInput, cmd = m.pathInput.Update(msg)
	cmds = append(cmds, cmd)
	m.nameInput, cmd = m.nameInput.Update(msg)
	cmds = append(cmds, cmd)
	
	return m, tea.Batch(cmds...)
}

// submitAddProject 提交新项目
func (m Model) submitAddProject() (Model, tea.Cmd) {
	path := strings.TrimSpace(m.pathInput.Value())
	name := strings.TrimSpace(m.nameInput.Value())

	if path == "" {
		m.statusMsg = "路径不能为空"
		m.pathInput.Focus()
		m.addStep = 0
		return m, nil
	}

	// 执行添加命令
	return m, func() tea.Msg {
		if m.registry == nil {
			return common.ErrorMsg{Err: fmt.Errorf("registry unavailable")}
		}
		
		proj, err := m.registry.Register(path)
		if err != nil {
			return common.ErrorMsg{Err: fmt.Errorf("添加项目失败: %w", err)}
		}
		
		// 如果指定了名称，更新名称
		if name != "" && name != proj.Name {
			proj.Name = name
			if err := m.registry.Update(proj); err != nil {
				return common.ErrorMsg{Err: fmt.Errorf("更新项目名称失败: %w", err)}
			}
		}

		return ProjectCreatedMsg{ID: proj.ID, Name: proj.Name}
	}
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