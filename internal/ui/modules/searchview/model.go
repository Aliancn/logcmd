package searchview

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aliancn/logcmd/internal/model"
	"github.com/aliancn/logcmd/internal/registry"
	"github.com/aliancn/logcmd/internal/search"
	"github.com/aliancn/logcmd/internal/ui/common"
	"github.com/aliancn/logcmd/internal/ui/components/panel"
)

const (
	maxSearchResults = 200
)

type projectsLoadedMsg struct {
	Projects []*model.Project
}

type projectsLoadFailedMsg struct {
	Err error
}

type searchFinishedMsg struct {
	QueryID int
	Items   []searchResultItem
	Summary string
}

type searchFailedMsg struct {
	QueryID int
	Err     error
}

type searchResultItem struct {
	project *model.Project
	result  *search.SearchResult
}

func (i searchResultItem) Title() string {
	if i.result == nil {
		return ""
	}
	return fmt.Sprintf("[%s] %s:%d",
		displayProject(i.project),
		relativePath(i.result.FilePath, i.project),
		i.result.LineNum,
	)
}

func (i searchResultItem) Description() string {
	if i.result == nil {
		return ""
	}
	if len(i.result.Context) == 0 {
		return strings.TrimSpace(i.result.Line)
	}
	lines := make([]string, len(i.result.Context))
	for idx, line := range i.result.Context {
		prefix := "  "
		if idx == len(i.result.Context)-1 {
			prefix = "> "
		}
		lines[idx] = prefix + strings.TrimSpace(line)
	}
	return strings.Join(lines, "\n")
}

func (i searchResultItem) FilterValue() string {
	if i.result == nil {
		return ""
	}
	return fmt.Sprintf("%s %s", i.result.FilePath, i.result.Line)
}

// Model 管理全局搜索界面。
type Model struct {
	registry *registry.Registry
	panel    *panel.Panel
	theme    common.Theme
	styles   common.Styles
	width    int
	height   int
	active   bool

	keyword       textinput.Model
	results       list.Model
	keys          keyMap
	contextValues []int

	regex         bool
	caseSensitive bool
	searchAll     bool
	manualScope   bool
	contextIndex  int
	inputFocused  bool

	currentProject *model.Project
	status         string
	loading        bool
	err            error
	queryID        int
}

// New 创建搜索视图。
func New(reg *registry.Registry, theme common.Theme, styles common.Styles) Model {
	input := textinput.New()
	input.Placeholder = "输入关键词后按 Enter 搜索"
	input.Prompt = "> "

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	results := list.New(nil, delegate, 0, 0)
	// 搜索结果列表禁用默认退出键，避免 Esc 触发 Quit
	results.DisableQuitKeybindings()
	results.Title = "搜索结果"
	results.SetShowStatusBar(false)
	results.SetFilteringEnabled(false)
	results.SetShowPagination(true)

	// 创建Panel布局容器
	p := panel.NewDefault("", theme, styles)

	return Model{
		registry:      reg,
		panel:         p,
		theme:         theme,
		styles:        styles,
		keyword:       input,
		results:       results,
		keys:          newKeyMap(),
		contextValues: []int{0, 2, 5},
		status:        "输入关键词后按 Enter 搜索",
	}
}

// Init 实现 tea.Model。
func (m Model) Init() tea.Cmd {
	return nil
}

// SetSize 更新尺寸。
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height

	// 构建header（form）
	header := m.renderForm()
	m.panel.SetHeader(header)

	// 设置Panel尺寸，Panel会自动计算内容区域（已扣除header）
	m.panel.SetSize(width, height)

	// 获取Panel计算后的精确内容尺寸
	contentW, contentH := m.panel.GetContentSize()

	// 确保最小尺寸
	if contentW < 30 {
		contentW = 30
	}
	if contentH < 5 {
		contentH = 5
	}

	// 设置results使用精确的内容尺寸
	m.results.SetSize(contentW, contentH)

	// 仅提示该视图特有的快捷键
	m.panel.SetFooter("Ctrl+O 浏览结果 · / 编辑关键词")
}

// Activate 激活搜索视图。
func (m *Model) Activate() tea.Cmd {
	m.active = true
	m.inputFocused = true
	m.status = "输入关键词后按 Enter 搜索"
	m.keyword.SetValue("")
	m.keyword.CursorEnd()
	cmds := []tea.Cmd{m.keyword.Focus()}
	if m.registry != nil {
		cmds = append(cmds, m.loadProjectsCmd())
	}
	return tea.Batch(cmds...)
}

// Deactivate 退出搜索视图。
func (m *Model) Deactivate() tea.Cmd {
	m.active = false
	m.keyword.Blur()
	m.inputFocused = false
	m.loading = false
	return nil
}

// FocusInput 让搜索输入框重新获得焦点
func (m *Model) FocusInput() tea.Cmd {
	m.inputFocused = true
	return m.keyword.Focus()
}

// SetCurrentProject 设置默认搜索项目。
func (m *Model) SetCurrentProject(project *model.Project) {
	m.currentProject = project
	if project == nil {
		m.searchAll = true
		m.manualScope = false
		return
	}
	if !m.manualScope {
		m.searchAll = false
	}
}

// Update 处理消息。
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(typed.Width, typed.Height)
	case tea.KeyMsg:
		if !m.active {
			break
		}
		if m.inputFocused {
			var cmd tea.Cmd
			m.keyword, cmd = m.keyword.Update(typed)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		if handled, cmd := m.handleKey(typed); handled {
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			break
		}
		if !m.inputFocused {
			var cmd tea.Cmd
			m.results, cmd = m.results.Update(typed)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
	case projectsLoadedMsg:
		if len(typed.Projects) == 0 && m.currentProject == nil {
			m.searchAll = true
		}
	case projectsLoadFailedMsg:
		m.status = fmt.Sprintf("加载项目失败: %v", typed.Err)
		m.err = typed.Err
	case searchFinishedMsg:
		if typed.QueryID != m.queryID {
			break
		}
		m.loading = false
		m.err = nil
		m.status = typed.Summary
		items := make([]list.Item, len(typed.Items))
		for i, item := range typed.Items {
			items[i] = item
		}
		m.results.SetItems(items)
	case searchFailedMsg:
		if typed.QueryID != m.queryID {
			break
		}
		m.loading = false
		m.err = typed.Err
		m.status = fmt.Sprintf("搜索失败: %v", typed.Err)
	}

	if _, ok := msg.(tea.KeyMsg); !ok {
		var cmd tea.Cmd
		m.results, cmd = m.results.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View 渲染界面。
func (m Model) View() string {
	if !m.active {
		return ""
	}

	// 使用Panel渲染results内容
	// header（form）已经在SetSize中设置好了
	return m.panel.Render(m.results.View())
}

func (m *Model) handleKey(keyMsg tea.KeyMsg) (bool, tea.Cmd) {
	switch {
	case key.Matches(keyMsg, m.keys.Run):
		if m.inputFocused {
			return true, m.startSearch()
		}
	case key.Matches(keyMsg, m.keys.Regex):
		m.regex = !m.regex
		return true, nil
	case key.Matches(keyMsg, m.keys.Case):
		m.caseSensitive = !m.caseSensitive
		return true, nil
	case key.Matches(keyMsg, m.keys.Scope):
		if m.currentProject != nil {
			m.manualScope = true
			m.searchAll = !m.searchAll
		} else {
			m.searchAll = true
		}
		return true, nil
	case key.Matches(keyMsg, m.keys.Context):
		m.contextIndex = (m.contextIndex + 1) % len(m.contextValues)
		return true, nil
	case key.Matches(keyMsg, m.keys.FocusInput):
		m.inputFocused = true
		return true, m.keyword.Focus()
	case key.Matches(keyMsg, m.keys.FocusResult):
		m.inputFocused = false
		m.keyword.Blur()
		return true, nil
	}
	return false, nil
}

func (m *Model) startSearch() tea.Cmd {
	query := strings.TrimSpace(m.keyword.Value())
	if query == "" {
		m.status = "请输入关键词"
		return nil
	}
	m.loading = true
	m.err = nil
	m.status = "正在搜索..."
	m.queryID++
	params := searchParams{
		QueryID:       m.queryID,
		Keyword:       query,
		Regex:         m.regex,
		CaseSensitive: m.caseSensitive,
		ContextLines:  m.contextValues[m.contextIndex],
		SearchAll:     m.searchAll || m.currentProject == nil,
		Project:       m.currentProject,
	}
	return m.searchCmd(params)
}

func (m Model) loadProjectsCmd() tea.Cmd {
	if m.registry == nil {
		return nil
	}
	return func() tea.Msg {
		projects, err := m.registry.List()
		if err != nil {
			return projectsLoadFailedMsg{Err: err}
		}
		return projectsLoadedMsg{Projects: projects}
	}
}

type searchParams struct {
	QueryID       int
	Keyword       string
	Regex         bool
	CaseSensitive bool
	ContextLines  int
	SearchAll     bool
	Project       *model.Project
}

func (m Model) searchCmd(params searchParams) tea.Cmd {
	reg := m.registry
	return func() tea.Msg {
		var targets []*model.Project
		if params.SearchAll {
			if reg == nil {
				return searchFailedMsg{
					QueryID: params.QueryID,
					Err:     fmt.Errorf("无法加载项目列表"),
				}
			}
			list, err := reg.List()
			if err != nil {
				return searchFailedMsg{QueryID: params.QueryID, Err: err}
			}
			targets = list
		} else if params.Project != nil {
			targets = []*model.Project{params.Project}
		} else {
			return searchFailedMsg{
				QueryID: params.QueryID,
				Err:     fmt.Errorf("没有可搜索的项目"),
			}
		}

		matches := make([]searchResultItem, 0, 32)
		ctx := context.Background()

		for _, project := range targets {
			opts := &search.SearchOptions{
				LogDir:        project.Path,
				Keyword:       params.Keyword,
				UseRegex:      params.Regex,
				CaseSensitive: params.CaseSensitive,
				ShowContext:   params.ContextLines,
			}
			searcher, err := search.New(opts)
			if err != nil {
				return searchFailedMsg{QueryID: params.QueryID, Err: err}
			}
			err = searcher.Search(ctx, func(result *search.SearchResult) error {
				if len(matches) >= maxSearchResults {
					return nil
				}
				matches = append(matches, searchResultItem{
					project: project,
					result:  cloneResult(result),
				})
				return nil
			})
			if err != nil && err != context.Canceled {
				return searchFailedMsg{QueryID: params.QueryID, Err: err}
			}
			if len(matches) >= maxSearchResults {
				break
			}
		}

		summary := fmt.Sprintf("匹配 %d 条 · 项目 %d 个", len(matches), len(targets))
		if len(matches) == 0 {
			summary = "未找到匹配"
		}

		return searchFinishedMsg{
			QueryID: params.QueryID,
			Items:   matches,
			Summary: summary,
		}
	}
}

func (m Model) renderForm() string {
	scope := "全部项目"
	if !m.searchAll && m.currentProject != nil {
		scope = fmt.Sprintf("当前项目 (#%d)", m.currentProject.ID)
	}
	contextLabel := fmt.Sprintf("%d 行", m.contextValues[m.contextIndex])
	status := m.status
	if m.loading {
		status = "正在搜索..."
	}
	line := fmt.Sprintf(
		"Regex:%s  Case:%s  范围:%s  上下文:%s  状态:%s",
		boolLabel(m.regex),
		boolLabel(m.caseSensitive),
		scope,
		contextLabel,
		status,
	)
	help := "Enter 搜索 · r 正则 · c 大小写 · a 范围 · x 上下文 · / 编辑 · ctrl+o 浏览结果"
	return fmt.Sprintf("%s\n%s\n%s", m.keyword.View(), line, help)
}

func boolLabel(v bool) string {
	if v {
		return "开"
	}
	return "关"
}

func cloneResult(res *search.SearchResult) *search.SearchResult {
	if res == nil {
		return nil
	}
	ctx := append([]string(nil), res.Context...)
	return &search.SearchResult{
		FilePath: res.FilePath,
		LineNum:  res.LineNum,
		Line:     res.Line,
		Context:  ctx,
	}
}

func displayProject(project *model.Project) string {
	if project == nil {
		return "-"
	}
	if strings.TrimSpace(project.Name) != "" {
		return strings.TrimSpace(project.Name)
	}
	return fmt.Sprintf("#%d", project.ID)
}

func relativePath(path string, project *model.Project) string {
	if project == nil {
		return path
	}
	if rel, err := filepath.Rel(project.Path, path); err == nil {
		return rel
	}
	return path
}

type keyMap struct {
	Run         key.Binding
	Regex       key.Binding
	Case        key.Binding
	Scope       key.Binding
	Context     key.Binding
	FocusInput  key.Binding
	FocusResult key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Run: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "执行搜索"),
		),
		Regex: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "切换正则"),
		),
		Case: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "切换大小写"),
		),
		Scope: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "切换范围"),
		),
		Context: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "切换上下文行数"),
		),
		FocusInput: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "编辑关键词"),
		),
		FocusResult: key.NewBinding(
			key.WithKeys("ctrl+o"),
			key.WithHelp("ctrl+o", "浏览结果"),
		),
	}
}
