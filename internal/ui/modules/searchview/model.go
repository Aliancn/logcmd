package searchview

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aliancn/logcmd/internal/model"
	"github.com/aliancn/logcmd/internal/registry"
	"github.com/aliancn/logcmd/internal/search"
	"github.com/aliancn/logcmd/internal/ui/common"
	"github.com/aliancn/logcmd/internal/ui/components/panel"
)

const (
	maxSearchResults = 200
)

// Messages
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
	Stats   searchSummary
}

type searchFailedMsg struct {
	QueryID int
	Err     error
}

type debounceMsg struct {
	ID    int
	Value string
}

// Data Structures
type searchResultItem struct {
	project *model.Project
	result  *search.SearchResult
}

type searchSummary struct {
	Query        string
	MatchCount   int
	ProjectCount int
	Duration     time.Duration
	Limited      bool
	ExecutedAt   time.Time
}

// searchResultItem implements list.Item
func (i searchResultItem) Title() string {
	if i.result == nil {
		return ""
	}
	// Use project style if available
	projectLabel := displayProject(i.project)
	return fmt.Sprintf("[%s] %s:%d",
		projectLabel,
		relativePath(i.result.FilePath, i.project),
		i.result.LineNum,
	)
}

func (i searchResultItem) Description() string {
	if i.result == nil {
		return ""
	}
	if len(i.result.Context) == 0 {
		return "> " + strings.TrimSpace(i.result.Line)
	}
	// find the match line index
	matchIdx := i.result.MatchLineIndex
	if matchIdx < 0 || matchIdx >= len(i.result.Context) {
		// Fallback if index is weird, though it shouldn't be
		matchIdx = len(i.result.Context) - 1
	}

	var sb strings.Builder
	for idx, line := range i.result.Context {
		prefix := "  "
		if idx == matchIdx {
			prefix = "> "
		}
		if idx > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(prefix + strings.TrimSpace(line))
	}
	return sb.String()
}

func (i searchResultItem) FilterValue() string {
	if i.result == nil {
		return ""
	}
	return fmt.Sprintf("%s %s", i.result.FilePath, i.result.Line)
}

// KeyMap
type keyMap struct {
	Run         key.Binding
	Scope       key.Binding
	Config      key.Binding
	FocusInput  key.Binding
	ToggleFocus key.Binding
	// Config specific
	ConfigUp    key.Binding
	ConfigDown  key.Binding
	ConfigLeft  key.Binding
	ConfigRight key.Binding
	ConfigEnter key.Binding
	ConfigEsc   key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Run: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "搜索"),
		),
		Scope: key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("ctrl+a", "范围"),
		),
		Config: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "设置"),
		),
		FocusInput: key.NewBinding(
			key.WithKeys("ctrl+f"),
			key.WithHelp("ctrl+f", "聚焦输入"),
		),
		ToggleFocus: key.NewBinding(
			key.WithKeys("ctrl+/", "ctrl+_", "tab", "shift+tab"),
			key.WithHelp("tab", "切换焦点"),
		),
		ConfigUp: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "上移"),
		),
		ConfigDown: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "下移"),
		),
		ConfigLeft: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "减少"),
		),
		ConfigRight: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "增加"),
		),
		ConfigEnter: key.NewBinding(
			key.WithKeys("enter", "space"),
			key.WithHelp("enter", "确认/切换"),
		),
		ConfigEsc: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "关闭"),
		),
	}
}

// Model
type Model struct {
	registry *registry.Registry
	panel    *panel.Panel
	theme    common.Theme
	styles   common.Styles
	width    int
	height   int
	active   bool

	// Inputs & List
	keyword       textinput.Model
	results       list.Model
	statusSpinner spinner.Model

	// State
	keys          keyMap
	caseSensitive bool
	contextLines  int
	searchAll     bool
	manualScope   bool // user manually toggled scope
	inputFocused  bool

	// Debounce
	debounceID int

	// Config Modal State
	showConfig  bool
	configFocus int // 0: Case Sensitive, 1: Context Lines

	currentProject *model.Project
	status         string
	loading        bool
	lastExecuted   time.Time
	lastStats      *searchSummary
	err            error
	queryID        int
}

// New creates the Search View Model
func New(reg *registry.Registry, theme common.Theme, styles common.Styles) Model {
	input := textinput.New()
	input.Placeholder = "输入关键词..."
	input.Prompt = "🔍 "
	input.PromptStyle = lipgloss.NewStyle().Foreground(theme.Primary)
	input.TextStyle = styles.Normal
	input.PlaceholderStyle = styles.Muted
	input.Cursor.Style = lipgloss.NewStyle().Foreground(theme.Primary)

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.Styles.SelectedTitle = styles.ListItemSelected
	delegate.Styles.SelectedDesc = styles.ListItemSelected.Copy().Foreground(theme.TextMuted)

	results := list.New(nil, delegate, 0, 0)
	results.DisableQuitKeybindings()
	results.Title = "搜索结果"
	results.SetShowTitle(false)
	results.SetShowStatusBar(false)
	results.SetFilteringEnabled(false)
	results.SetShowPagination(true)
	results.Styles.PaginationStyle = styles.Muted

	p := panel.NewDefault("", theme, styles)

	spin := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(theme.Primary)),
	)

	return Model{
		registry:      reg,
		panel:         p,
		theme:         theme,
		styles:        styles,
		keyword:       input,
		results:       results,
		statusSpinner: spin,
		keys:          newKeyMap(),
		caseSensitive: false,
		contextLines:  0,
		status:        "准备就绪",
		inputFocused:  true,
		configFocus:   0,
	}
}

// Init
func (m Model) Init() tea.Cmd {
	return nil
}

// SetSize
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height

	// 1. First set the total size of the panel to calculate the available content width
	m.panel.SetSize(width, height)

	// 2. Render header and footer using the calculated content width
	m.panel.SetHeader(m.renderHeader())
	m.panel.SetFooter(m.renderFooter())

	// 3. Get the final content size (after header/footer height deduction)
	contentW, contentH := m.panel.GetContentSize()

	if contentW < 10 {
		contentW = 10
	}
	if contentH < 5 {
		contentH = 5
	}
	m.results.SetSize(contentW, contentH)
}

// Activate
func (m *Model) Activate() tea.Cmd {
	m.active = true
	m.inputFocused = true
	m.keyword.Focus()
	m.keyword.CursorEnd()

	cmds := []tea.Cmd{textinput.Blink}
	if m.registry != nil {
		cmds = append(cmds, m.loadProjectsCmd())
	}
	return tea.Batch(cmds...)
}

// Deactivate
func (m *Model) Deactivate() tea.Cmd {
	m.active = false
	m.keyword.Blur()
	m.inputFocused = false
	return nil
}

// FocusInput
func (m *Model) FocusInput() tea.Cmd {
	m.inputFocused = true
	return m.keyword.Focus()
}

// SetCurrentProject
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

// IsInputFocused returns true if the input is currently focused
func (m Model) IsInputFocused() bool {
	return m.inputFocused
}

// Update
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch typed := msg.(type) {
	case debounceMsg:
		if typed.ID == m.debounceID {
			return m, m.startSearch()
		}
		return m, nil

	case tea.KeyMsg:
		if !m.active {
			break
		}

		// Config Overlay Handling
		if m.showConfig {
			switch {
			case key.Matches(typed, m.keys.ConfigEsc):
				m.showConfig = false
				return m, nil

			case key.Matches(typed, m.keys.ConfigUp):
				m.configFocus--
				if m.configFocus < 0 {
					m.configFocus = 1 // wrap
				}
				return m, nil

			case key.Matches(typed, m.keys.ConfigDown):
				m.configFocus++
				if m.configFocus > 1 {
					m.configFocus = 0 // wrap
				}
				return m, nil

			case key.Matches(typed, m.keys.ConfigEnter):
				if m.configFocus == 0 {
					m.caseSensitive = !m.caseSensitive
				}
				// For context lines, Enter doesn't do much if we use Left/Right, maybe close?
				return m, nil

			case key.Matches(typed, m.keys.ConfigLeft):
				if m.configFocus == 1 && m.contextLines > 0 {
					m.contextLines--
				}
				return m, nil

			case key.Matches(typed, m.keys.ConfigRight):
				if m.configFocus == 1 && m.contextLines < 20 {
					m.contextLines++
				}
				return m, nil
			}
			// In config mode, eat all other keys
			return m, nil
		}

		// Global Search View shortcuts
		switch {
		case key.Matches(typed, m.keys.Config):
			m.showConfig = true
			return m, nil

		case key.Matches(typed, m.keys.ToggleFocus):
			m.inputFocused = !m.inputFocused
			if m.inputFocused {
				return m, m.keyword.Focus()
			} else {
				m.keyword.Blur()
				return m, nil
			}

		case key.Matches(typed, m.keys.FocusInput):
			if !m.inputFocused {
				m.inputFocused = true
				return m, m.keyword.Focus()
			}
		
		case key.Matches(typed, m.keys.Scope):
			m.toggleScope()
			return m, nil
		}

		if m.inputFocused {
			// Input Mode handling
			switch {
			case key.Matches(typed, m.keys.Run): // Enter to search
				cmds = append(cmds, m.startSearch())
			// NOTE: Removed Down arrow auto-switching
			default:
				oldVal := m.keyword.Value()
				var cmd tea.Cmd
				m.keyword, cmd = m.keyword.Update(typed)
				cmds = append(cmds, cmd)

				// Trigger debounce search if value changed
				newVal := m.keyword.Value()
				if newVal != oldVal {
					m.debounceID++
					id := m.debounceID
					cmds = append(cmds, tea.Tick(time.Millisecond*400, func(t time.Time) tea.Msg {
						return debounceMsg{ID: id, Value: newVal}
					}))
				}
			}

		} else {
			// List Mode handling
			switch {
			// NOTE: Removed Up arrow auto-switching
			// NOTE: Removed "/" key switching (now handled by ToggleFocus or FocusInput explicitly via Ctrl)
			
			case typed.String() == "enter":
				// Enter to open result
				if item, ok := m.results.SelectedItem().(searchResultItem); ok && item.result != nil {
					return m, func() tea.Msg {
						return common.OpenProjectLogMsg{
							Project:  item.project,
							FilePath: item.result.FilePath,
							LineNum:  item.result.LineNum,
						}
					}
				}
			}

			var cmd tea.Cmd
			m.results, cmd = m.results.Update(typed)
			cmds = append(cmds, cmd)
		}

		// Always update header/footer on key event
		m.panel.SetHeader(m.renderHeader())
		m.panel.SetFooter(m.renderFooter())

		return m, tea.Batch(cmds...)

	case projectsLoadedMsg:
		if len(typed.Projects) == 0 && m.currentProject == nil {
			m.searchAll = true
		}

	case projectsLoadFailedMsg:
		m.status = fmt.Sprintf("加载项目失败: %v", typed.Err)
		m.err = typed.Err

	case searchFinishedMsg:
		if typed.QueryID == m.queryID {
			m.loading = false
			m.err = nil
			m.status = typed.Summary
			stats := typed.Stats
			m.lastStats = &stats
			m.lastExecuted = stats.ExecutedAt

			items := make([]list.Item, len(typed.Items))
			for i, item := range typed.Items {
				items[i] = item
			}
			m.results.SetItems(items)
			m.results.ResetSelected()

			// Auto focus list if we have results
			if len(items) > 0 {
				m.inputFocused = false
				m.keyword.Blur()
			}
		}

	case searchFailedMsg:
		if typed.QueryID == m.queryID {
			m.loading = false
			m.err = typed.Err
			m.status = fmt.Sprintf("搜索失败: %v", typed.Err)
			m.lastExecuted = time.Now()
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.statusSpinner, cmd = m.statusSpinner.Update(typed)
		if m.loading && cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) toggleScope() {
	if m.currentProject != nil {
		m.manualScope = true
		m.searchAll = !m.searchAll
	} else {
		m.searchAll = true
	}
}

// Actions
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
		CaseSensitive: m.caseSensitive,
		ContextLines:  m.contextLines,
		SearchAll:     m.searchAll || m.currentProject == nil,
		Project:       m.currentProject,
	}
	return tea.Batch(
		m.searchCmd(params),
		m.statusSpinner.Tick,
	)
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
	CaseSensitive bool
	ContextLines  int
	SearchAll     bool
	Project       *model.Project
}

func (m Model) searchCmd(params searchParams) tea.Cmd {
	reg := m.registry
	return func() tea.Msg {
		started := time.Now()
		var targets []*model.Project

		// Determine targets
		if params.SearchAll {
			if reg == nil {
				return searchFailedMsg{QueryID: params.QueryID, Err: fmt.Errorf("registry not initialized")}
			}
			list, err := reg.List()
			if err != nil {
				return searchFailedMsg{QueryID: params.QueryID, Err: err}
			}
			targets = list
		} else if params.Project != nil {
			targets = []*model.Project{params.Project}
		} else {
			return searchFailedMsg{QueryID: params.QueryID, Err: fmt.Errorf("no project selected")}
		}

		matches := make([]searchResultItem, 0, 32)
		ctx := context.Background()

		// Execute Search
		for _, project := range targets {
			// 检查日志目录是否存在，不存在则跳过该项目
			if _, err := os.Stat(project.Path); os.IsNotExist(err) {
				continue
			}

			opts := &search.SearchOptions{
				LogDir:        project.Path,
				Keyword:       params.Keyword,
				UseRegex:      false, // Disabled
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

		// Summary
		summary := fmt.Sprintf("匹配 %d 条 · 项目 %d 个", len(matches), len(targets))
		limited := len(matches) >= maxSearchResults
		if limited {
			summary = fmt.Sprintf("%s (已截断)", summary)
		}
		if len(matches) == 0 {
			summary = "未找到匹配项"
		}

		stats := searchSummary{
			Query:        params.Keyword,
			MatchCount:   len(matches),
			ProjectCount: len(targets),
			Duration:     time.Since(started),
			Limited:      limited,
			ExecutedAt:   time.Now(),
		}

		return searchFinishedMsg{
			QueryID: params.QueryID,
			Items:   matches,
			Summary: summary,
			Stats:   stats,
		}
	}
}

// Rendering

func (m Model) renderHeader() string {
	width, _ := m.panel.GetContentSize()
	if width <= 0 {
		width = m.width - 4
	}
	if width <= 0 {
		return ""
	}

	// 1. Input Line
	inputStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Border(lipgloss.RoundedBorder())

	if m.inputFocused {
		inputStyle = inputStyle.BorderForeground(m.theme.BorderActive)
	} else {
		inputStyle = inputStyle.BorderForeground(m.theme.Border)
	}

	inputView := inputStyle.Width(width - 2).Render(m.keyword.View())

	// 2. Info Line (Options + Status)

	// Options
	var options []string
	// Only show Scope and active settings.
	// Since regex/case/context are now in config, we might want to just show the "Active" bits as badges?
	// Or just show Scope.
	options = append(options, m.renderOption(m.scopeLabel(), true))
	
	// Show indicators if non-default settings are active
	if m.caseSensitive {
		options = append(options, m.renderOption("Aa", true))
	}
	if m.contextLines > 0 {
		options = append(options, m.renderOption(fmt.Sprintf("Ctx:%d", m.contextLines), true))
	}

	leftSide := lipgloss.JoinHorizontal(lipgloss.Center, options...)

	// Status
	var statusStr string
	if m.loading {
		statusStr = fmt.Sprintf("%s %s", m.statusSpinner.View(), m.status)
	} else {
		statusStr = m.status
		if m.lastStats != nil {
			statusStr += fmt.Sprintf(" [%s]", formatDuration(m.lastStats.Duration))
		}
	}
	rightSide := m.styles.Muted.Render(statusStr)

	availInnerWidth := width - 2
	leftW := lipgloss.Width(leftSide)
	rightW := lipgloss.Width(rightSide)

	gap := availInnerWidth - leftW - rightW
	if gap < 2 {
		gap = 2
	}

	var infoLineContent string
	if leftW+rightW+gap > availInnerWidth {
		infoLineContent = lipgloss.JoinHorizontal(lipgloss.Top, leftSide, "  ", rightSide)
	} else {
		infoLineContent = lipgloss.JoinHorizontal(lipgloss.Center,
			leftSide,
			strings.Repeat(" ", gap),
			rightSide,
		)
	}

	infoLine := lipgloss.NewStyle().
		Padding(0, 1).
		Width(width).
		Render(infoLineContent)

	return lipgloss.JoinVertical(lipgloss.Left, inputView, infoLine)
}

func (m Model) renderOption(label string, active bool) string {
	style := lipgloss.NewStyle().
		MarginRight(2).
		Foreground(m.theme.TextMuted)

	if active {
		style = style.Foreground(m.theme.Primary).Bold(true)
	}
	return style.Render(label)
}

func (m Model) renderFooter() string {
	if m.showConfig {
		return common.JoinKeyHelps(
			common.FormatKeyHelp(m.keys.ConfigUp),
			common.FormatKeyHelp(m.keys.ConfigEnter),
			common.FormatKeyHelp(m.keys.ConfigLeft),
			common.FormatKeyHelp(m.keys.ConfigEsc),
		)
	}

	return common.JoinKeyHelps(
		common.FormatKeyHelp(m.keys.Run),
		common.FormatKeyHelp(m.keys.ToggleFocus),
		common.FormatKeyHelp(m.keys.FocusInput),
		common.FormatKeyHelp(m.keys.Config),
		common.FormatKeyHelp(m.keys.Scope),
	)
}

func (m Model) View() string {
	if !m.active {
		return ""
	}
	
	var content string
	
	if m.showConfig {
		// Render Config Popup
		content = m.renderConfigPopup()
	} else {
		content = m.results.View()
	}

	return m.panel.Render(content)
}

func (m Model) renderConfigPopup() string {
	// Calculate center relative to panel
	w, h := m.panel.GetContentSize()
	
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Primary).
		Padding(1, 2).
		Width(40)

	title := lipgloss.NewStyle().
		Foreground(m.theme.Primary).
		Bold(true).
		MarginBottom(1).
		Render("搜索设置")

	// Option 1: Case Sensitive
	opt1Title := "大小写敏感"
	opt1Value := "OFF"
	if m.caseSensitive {
		opt1Value = "ON"
	}
	opt1 := m.renderConfigItem(opt1Title, opt1Value, m.configFocus == 0)

	// Option 2: Context Lines
	opt2Title := "上下文行数"
	opt2Value := fmt.Sprintf("%d", m.contextLines)
	opt2 := m.renderConfigItem(opt2Title, opt2Value, m.configFocus == 1)

	// Helper hint
	hint := lipgloss.NewStyle().
		MarginTop(1).
		Foreground(m.theme.TextMuted).
		Render("Esc 关闭 · Space/←/→ 调整")

	form := lipgloss.JoinVertical(lipgloss.Center,
		title,
		opt1,
		opt2,
		hint,
	)

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, boxStyle.Render(form))
}

func (m Model) renderConfigItem(label, value string, active bool) string {
	container := lipgloss.NewStyle().
		Width(34).
		PaddingLeft(1).
		PaddingRight(1)
	
	labelStyle := lipgloss.NewStyle().Foreground(m.theme.Foreground)
	valueStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted)
	
	if active {
		container = container.Background(m.theme.TabActive) // Highlights row
		labelStyle = labelStyle.Bold(true).Foreground(m.theme.Primary)
		valueStyle = valueStyle.Bold(true).Foreground(m.theme.Primary)
	}

	// Justify
	l := labelStyle.Render(label)
	v := valueStyle.Render(value)
	
	// spacer
	dots := 34 - lipgloss.Width(l) - lipgloss.Width(v) - 2 // -2 for padding
	if dots < 1 { dots = 1 }
	spacer := strings.Repeat(" ", dots)

	return container.Render(lipgloss.JoinHorizontal(lipgloss.Center, l, spacer, v))
}

// Helpers
func boolLabel(v bool) string {
	if v {
		return "ON"
	}
	return "OFF"
}

func (m Model) scopeLabel() string {
	if !m.searchAll && m.currentProject != nil {
		return fmt.Sprintf("项目(#%d)", m.currentProject.ID)
	}
	return "全部项目"
}

func cloneResult(res *search.SearchResult) *search.SearchResult {
	if res == nil {
		return nil
	}
	ctx := append([]string(nil), res.Context...)
	return &search.SearchResult{
		FilePath:       res.FilePath,
		LineNum:        res.LineNum,
		Line:           res.Line,
		Context:        ctx,
		MatchLineIndex: res.MatchLineIndex,
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

func formatDuration(d time.Duration) string {
	if d >= time.Second {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
	return fmt.Sprintf("%dms", d.Milliseconds())
}