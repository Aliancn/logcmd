package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aliancn/logcmd/internal/ui/common"
	"github.com/aliancn/logcmd/internal/ui/modules/historylist"
	"github.com/aliancn/logcmd/internal/ui/modules/logviewer"
	"github.com/aliancn/logcmd/internal/ui/modules/projectlist"
)

// Update 是根 Model 的 Update 实现。
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = typed.Width, typed.Height
		m.ready = true
		listWidth, statsWidth, vertical, compact := m.projectLayout()
		m.projectSplitVertical = vertical
		m.projectStatsCompact = compact
		if compact {
			m.projectList.SetSize(m.width, m.height)
			m.statsPanel.SetSize(m.width, 0)
		} else if vertical {
			half := m.height / 2
			if half < 1 {
				half = m.height
			}
			m.projectList.SetSize(m.width, half)
			m.statsPanel.SetSize(m.width, half)
		} else {
			m.projectList.SetSize(listWidth, m.height)
			m.statsPanel.SetSize(statsWidth, m.height)
		}
		m.historyList.SetSize(m.width, m.height)
		m.logViewer.SetSize(m.width, m.height)
		m.taskList.SetSize(m.width, m.height)
		m.searchView.SetSize(m.width, m.height)
	case tea.KeyMsg:
		if m.handleGlobalKey(typed, &cmds) {
			break
		}
	case projectlist.ProjectSelectedMsg:
		m.state = HistoryListView
		m.historyList.SetProject(typed.Project)
		cmds = append(cmds, m.historyList.LoadHistoryCmd())
	case historylist.BackToProjectsMsg:
		m.state = ProjectListView
		m.logViewer.Reset()
	case historylist.OpenLogMsg:
		m.state = LogViewerView
		m.logViewer.SetHistory(typed.History)
		cmds = append(cmds, m.logViewer.LoadContentCmd())
	case logviewer.BackMsg:
		m.state = HistoryListView
	case logviewer.ContentLoadedMsg:
		// 由 logviewer 自身处理
	case projectlist.ProjectDeletedMsg:
		if m.historyList.CurrentProjectID() == typed.ProjectID {
			m.state = ProjectListView
			m.historyList.SetProject(nil)
			m.logViewer.Reset()
		}
	case common.ErrorMsg:
		m.err = typed.Err
	}

	var cmd tea.Cmd
	switch m.state {
	case ProjectListView:
		m.projectList, cmd = m.projectList.Update(msg)
	case HistoryListView:
		m.historyList, cmd = m.historyList.Update(msg)
	case LogViewerView:
		m.logViewer, cmd = m.logViewer.Update(msg)
	case TaskListView:
		m.taskList, cmd = m.taskList.Update(msg)
	case SearchView:
		m.searchView, cmd = m.searchView.Update(msg)
	}
	cmds = append(cmds, cmd)

	m.statsPanel, cmd = m.statsPanel.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *Model) handleGlobalKey(keyMsg tea.KeyMsg, cmds *[]tea.Cmd) bool {
	switch {
	case key.Matches(keyMsg, m.globalKeys.Quit):
		*cmds = append(*cmds, tea.Quit)
		return true
	case key.Matches(keyMsg, m.globalKeys.Back):
		switch m.state {
		case HistoryListView:
			*cmds = append(*cmds, func() tea.Msg { return historylist.BackToProjectsMsg{} })
		case LogViewerView:
			*cmds = append(*cmds, func() tea.Msg { return logviewer.BackMsg{} })
		case TaskListView:
			if cmd := m.exitTaskView(); cmd != nil {
				*cmds = append(*cmds, cmd)
			}
		case SearchView:
			if cmd := m.exitSearchView(); cmd != nil {
				*cmds = append(*cmds, cmd)
			}
		default:
			// 在项目列表中忽略
		}
		return true
	case key.Matches(keyMsg, m.globalKeys.Task):
		if m.state == TaskListView {
			if cmd := m.exitTaskView(); cmd != nil {
				*cmds = append(*cmds, cmd)
			}
		} else {
			if cmd := m.enterTaskView(); cmd != nil {
				*cmds = append(*cmds, cmd)
			}
		}
		return true
	case key.Matches(keyMsg, m.globalKeys.Search):
		if m.state == LogViewerView {
			return false
		}
		if m.state == SearchView {
			if cmd := m.exitSearchView(); cmd != nil {
				*cmds = append(*cmds, cmd)
			}
		} else {
			if cmd := m.enterSearchView(); cmd != nil {
				*cmds = append(*cmds, cmd)
			}
		}
		return true
	}
	return false
}

func (m *Model) enterTaskView() tea.Cmd {
	m.prevState = m.state
	m.state = TaskListView
	return m.taskList.SetActive(true)
}

func (m *Model) exitTaskView() tea.Cmd {
	if m.state != TaskListView {
		return nil
	}
	target := m.prevState
	if target == TaskListView {
		target = ProjectListView
	}
	m.state = target
	return m.taskList.SetActive(false)
}

func (m *Model) enterSearchView() tea.Cmd {
	m.prevState = m.state
	m.searchView.SetCurrentProject(m.projectList.SelectedProject())
	m.state = SearchView
	return m.searchView.Activate()
}

func (m *Model) exitSearchView() tea.Cmd {
	if m.state != SearchView {
		return nil
	}
	target := m.prevState
	if target == TaskListView || target == SearchView {
		target = ProjectListView
	}
	m.state = target
	return m.searchView.Deactivate()
}

func (m *Model) projectLayout() (listWidth, statsWidth int, vertical, compact bool) {
	if m.width <= 0 {
		return 0, 0, false, false
	}
	if m.width < 90 {
		return m.width, 0, false, true
	}
	if m.width < 110 {
		return m.width, m.width, true, false
	}
	statsWidth = m.width / 3
	if statsWidth < 30 {
		statsWidth = 30
	}
	if statsWidth > m.width/2 {
		statsWidth = m.width / 2
	}
	listWidth = m.width - statsWidth
	if listWidth < 40 {
		return m.width, 0, false, true
	}
	return listWidth, statsWidth, false, false
}
