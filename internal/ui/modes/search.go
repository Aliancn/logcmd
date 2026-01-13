package modes

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"

	"github.com/aliancn/logcmd/internal/history"
	"github.com/aliancn/logcmd/internal/model"
	"github.com/aliancn/logcmd/internal/registry"
	"github.com/aliancn/logcmd/internal/search"
	"github.com/aliancn/logcmd/internal/ui/common"
)

// SearchMode 搜索模式 - 使用 bubbles 组件实现
//
// 核心功能：
//   - 实时模糊搜索日志内容
//   - 支持全部项目或单项目搜索
//   - 预览匹配行的上下文
//   - 快捷键导航和选择
type SearchMode struct {
	// 依赖
	registry   *registry.Registry
	historyMgr *history.Manager

	// UI 组件
	input textinput.Model // 搜索输入框
	list  list.Model      // 结果列表

	// 搜索状态
	allItems       []SearchItem   // 所有已加载的日志条目
	filteredItems  []SearchItem   // 过滤后的结果
	currentProject *model.Project // 当前限定的项目（nil=全部）
	searchAll      bool           // 是否搜索所有项目
	loaded         bool           // 数据是否已加载
	loading        bool           // 是否正在加载
	loadProgress   string         // 加载进度提示
	errorMsg       string         // 错误消息

	// 配置
	caseSensitive bool // 大小写敏感
	contextLines  int  // 上下文行数

	// 布局
	width  int
	height int

	// 样式
	theme  common.Theme
	styles common.Styles
}

// SearchItem 表示一条搜索结果
type SearchItem struct {
	Project     *model.Project
	Result      *search.SearchResult
	DisplayText string // 预渲染的显示文本
}

// 实现 list.Item 接口
func (i SearchItem) Title() string       { return i.DisplayText }
func (i SearchItem) Description() string { return "" }
func (i SearchItem) FilterValue() string { return i.DisplayText }

// NewSearchMode 创建搜索模式
func NewSearchMode(reg *registry.Registry, historyMgr *history.Manager, theme common.Theme, styles common.Styles) *SearchMode {
	// 创建搜索输入框
	ti := textinput.New()
	ti.Placeholder = "输入关键词搜索..."
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 50

	// 创建结果列表
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "搜索结果"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(false) // 我们自己处理过滤
	l.SetShowHelp(false)
	l.DisableQuitKeybindings() // 禁用默认的退出快捷键（q等）

	return &SearchMode{
		registry:      reg,
		historyMgr:    historyMgr,
		input:         ti,
		list:          l,
		searchAll:     true,
		contextLines:  2,
		caseSensitive: false,
		theme:         theme,
		styles:        styles,
	}
}

// Name 实现 Mode 接口
func (m *SearchMode) Name() string {
	return "search"
}

// Activate 实现 Mode 接口
func (m *SearchMode) Activate() tea.Cmd {
	m.input.Focus()

	if !m.loaded && !m.loading {
		// 首次激活，加载所有日志
		return m.loadAllLogsCmd()
	}

	return nil
}

// Deactivate 实现 Mode 接口
func (m *SearchMode) Deactivate() tea.Cmd {
	m.input.Blur()
	return nil
}

// Update 实现 Mode 接口
func (m *SearchMode) Update(msg tea.Msg) (Mode, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// 更新组件尺寸
		m.input.Width = msg.Width - 4

		// 列表高度 = 总高度 - 状态栏(3行) - 输入框区域(4行) - 底部提示(2行)
		listHeight := msg.Height - 9
		if listHeight < 5 {
			listHeight = 5
		}
		m.list.SetSize(msg.Width-2, listHeight)

	case logsLoadedMsg:
		// 日志加载完成
		m.allItems = msg.items
		m.loaded = true
		m.loading = false
		m.loadProgress = ""
		m.errorMsg = ""

		// 初始显示所有项
		m.filteredItems = m.allItems
		m.updateListItems()

		return m, nil

	case logsBatchMsg:
		// 增量加载批次
		m.allItems = append(m.allItems, msg.items...)
		m.loadProgress = msg.progress

		// 如果当前没有搜索，更新显示
		if m.input.Value() == "" {
			m.filteredItems = m.allItems
			m.updateListItems()
		}

		return m, nil

	case logsLoadFailedMsg:
		// 加载失败
		m.loading = false
		m.loaded = true
		m.loadProgress = ""
		m.errorMsg = fmt.Sprintf("加载失败: %v", msg.err)
		return m, nil

	case searchCompleteMsg:
		// 搜索完成，更新过滤结果
		m.filteredItems = msg.results
		m.updateListItems()
		return m, nil

	case tea.KeyMsg:
		// 不在这里处理，交给 HandleKey
	}

	// 更新输入框
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	// 输入变化时触发搜索
	if msg, ok := msg.(tea.KeyMsg); ok {
		if msg.Type == tea.KeyRunes || msg.Type == tea.KeyBackspace || msg.Type == tea.KeyDelete {
			cmds = append(cmds, m.performSearchCmd())
		}
	}

	// 更新列表
	m.list, cmd = m.list.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View 实现 Mode 接口
func (m *SearchMode) View() string {
	if !m.loaded {
		if m.loading {
			return m.renderLoading()
		}
		return "准备加载日志数据..."
	}

	// 状态栏
	statusBar := m.renderStatusBar()

	// 错误消息（如果有）
	var errorView string
	if m.errorMsg != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(m.theme.Error).
			Padding(0, 2)
		errorView = errorStyle.Render("⚠️  " + m.errorMsg)
	}

	// 搜索输入区域
	inputArea := m.renderInputArea()

	// 结果列表
	listView := m.list.View()

	// 底部提示
	footer := m.renderFooter()

	if errorView != "" {
		return lipgloss.JoinVertical(lipgloss.Left,
			statusBar,
			errorView,
			inputArea,
			listView,
			footer,
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		statusBar,
		inputArea,
		listView,
		footer,
	)
}

// HandleKey 实现 Mode 接口
func (m *SearchMode) HandleKey(key string) (bool, tea.Cmd) {
	switch key {
	case "enter":
		// 打开选中的日志文件
		if item := m.selectedItem(); item != nil {
			return true, func() tea.Msg {
				return OpenLogFileMsg{
					FilePath:    item.Result.FilePath,
					LineNum:     item.Result.LineNum,
					SearchQuery: m.input.Value(),
					ReturnMode:  "search",
					Follow:      false,
				}
			}
		}
	case "ctrl+a":
		// 切换搜索范围
		if m.toggleSearchScope() {
			return true, m.loadAllLogsCmd()
		}
		return true, nil
	case "esc":
		// 清空搜索
		if m.input.Value() != "" {
			m.input.SetValue("")
			return true, m.performSearchCmd()
		}
	}

	return false, nil
}

// selectedItem 获取当前选中的搜索项
func (m *SearchMode) selectedItem() *SearchItem {
	if item, ok := m.list.SelectedItem().(SearchItem); ok {
		return &item
	}
	return nil
}

// SetProject 设置当前项目（用于限定搜索范围）
func (m *SearchMode) SetProject(proj *model.Project) {
	if proj != nil {
		m.currentProject = proj
		m.searchAll = false
	} else {
		m.currentProject = nil
		m.searchAll = true
	}
	m.loaded = false
}

// SetSearchKeyword 设置搜索关键词并触发搜索
func (m *SearchMode) SetSearchKeyword(keyword string) {
	m.input.SetValue(keyword)
	// 焦点到输入框
	m.input.Focus()
}

// loadAllLogsCmd 加载所有日志条目
func (m *SearchMode) loadAllLogsCmd() tea.Cmd {
	if m.loading {
		return nil
	}

	m.loading = true

	return func() tea.Msg {
		ctx := context.Background()
		items := make([]SearchItem, 0, 10000)

		// 确定搜索目标
		var targets []*model.Project
		if m.searchAll {
			projects, err := m.registry.List()
			if err != nil {
				return logsLoadFailedMsg{err: err}
			}
			targets = projects
		} else if m.currentProject != nil {
			targets = []*model.Project{m.currentProject}
		} else {
			// 没有项目可搜索
			return logsLoadedMsg{items: items}
		}

		// 遍历所有项目，扫描日志文件
		for _, proj := range targets {
			logDir := m.resolveLogDir(proj)
			if logDir == "" {
				continue
			}

			// 使用现有的搜索引擎，空关键词=匹配所有行
			opts := &search.SearchOptions{
				LogDir:        logDir,
				Keyword:       "", // 空关键词=全匹配
				CaseSensitive: false,
				ShowContext:   m.contextLines,
			}

			searcher, err := search.New(opts)
			if err != nil {
				continue // 跳过失败的项目
			}

			// 流式扫描
			searcher.Search(ctx, func(result *search.SearchResult) error {
				// 预渲染显示文本
				displayText := formatSearchResult(proj, result)

				items = append(items, SearchItem{
					Project:     proj,
					Result:      result,
					DisplayText: displayText,
				})

				return nil
			})
		}

		return logsLoadedMsg{items: items}
	}
}

// performSearchCmd 执行模糊搜索
func (m *SearchMode) performSearchCmd() tea.Cmd {
	return func() tea.Msg {
		query := strings.TrimSpace(m.input.Value())

		if query == "" {
			// 空查询，显示所有项
			return searchCompleteMsg{results: m.allItems}
		}

		// 使用 sahilm/fuzzy 进行模糊匹配
		strs := make([]string, len(m.allItems))
		for i, item := range m.allItems {
			strs[i] = item.DisplayText
		}

		matches := fuzzy.Find(query, strs)

		// 构建匹配结果
		results := make([]SearchItem, len(matches))
		for i, match := range matches {
			results[i] = m.allItems[match.Index]
		}

		return searchCompleteMsg{results: results}
	}
}

// updateListItems 更新列表项
func (m *SearchMode) updateListItems() {
	items := make([]list.Item, len(m.filteredItems))
	for i, item := range m.filteredItems {
		items[i] = item
	}
	m.list.SetItems(items)
}

// toggleSearchScope 切换搜索范围
func (m *SearchMode) toggleSearchScope() bool {
	if m.searchAll {
		if m.currentProject == nil {
			m.errorMsg = "请先在项目模式选择项目后再切换到单项目搜索"
			return false
		}
		m.searchAll = false
	} else {
		m.searchAll = true
	}
	m.loaded = false
	return true
}

// resolveLogDir 返回项目日志目录，兼容已经指向 .logcmd 的路径
func (m *SearchMode) resolveLogDir(proj *model.Project) string {
	if proj == nil {
		return ""
	}

	path := filepath.Clean(proj.Path)
	if filepath.Base(path) == ".logcmd" {
		return path
	}
	return filepath.Join(path, ".logcmd")
}

// formatSearchResult 格式化搜索结果显示
func formatSearchResult(proj *model.Project, result *search.SearchResult) string {
	// 相对路径
	relPath := strings.TrimPrefix(result.FilePath, proj.Path)
	relPath = strings.TrimPrefix(relPath, "/")

	// 格式: "file.log:42 [P#1] content preview"
	content := strings.TrimSpace(result.Line)
	if len(content) > 80 {
		content = content[:77] + "..."
	}

	return fmt.Sprintf("%s:%d [P#%d] %s", relPath, result.LineNum, proj.ID, content)
}

// renderStatusBar 渲染状态栏
func (m *SearchMode) renderStatusBar() string {
	scope := "所有项目"
	if !m.searchAll && m.currentProject != nil {
		scope = fmt.Sprintf("项目#%d", m.currentProject.ID)
	}

	resultCount := fmt.Sprintf("%d / %d", len(m.filteredItems), len(m.allItems))

	status := fmt.Sprintf("[搜索] %s · 结果: %s", scope, resultCount)

	statusStyle := lipgloss.NewStyle().
		Foreground(m.theme.Foreground).
		Background(m.theme.StatusBar).
		Padding(0, 1).
		Width(m.width)

	return statusStyle.Render(status)
}

// renderInputArea 渲染输入区域
func (m *SearchMode) renderInputArea() string {
	prompt := lipgloss.NewStyle().
		Foreground(m.theme.Primary).
		Bold(true).
		Render("🔍 ")

	return "\n" + prompt + m.input.View() + "\n"
}

// renderFooter 渲染底部提示
func (m *SearchMode) renderFooter() string {
	hints := "Enter 打开 · ↑↓ 导航 · Ctrl+A 切换范围 · Esc 清空 · Ctrl+P/T/S/L 切换模式"

	footerStyle := lipgloss.NewStyle().
		Foreground(m.theme.TextMuted).
		Padding(0, 1)

	return footerStyle.Render(hints)
}

// renderLoading 渲染加载状态
func (m *SearchMode) renderLoading() string {
	if m.height == 0 {
		return "加载中..."
	}

	padding := (m.height - 3) / 2
	if padding < 0 {
		padding = 0
	}

	var view string
	for i := 0; i < padding; i++ {
		view += "\n"
	}

	loadingStyle := lipgloss.NewStyle().
		Foreground(m.theme.Primary).
		Bold(true)

	msg := "⏳ 正在加载日志数据..."
	if m.loadProgress != "" {
		msg = "⏳ " + m.loadProgress
	}

	view += loadingStyle.Render(msg)
	return view
}

// 消息类型

type logsLoadedMsg struct {
	items []SearchItem
}

type logsLoadFailedMsg struct {
	err error
}

type logsBatchMsg struct {
	items    []SearchItem
	progress string // 进度提示，如 "已加载 500/1000"
}

type searchCompleteMsg struct {
	results []SearchItem
}
