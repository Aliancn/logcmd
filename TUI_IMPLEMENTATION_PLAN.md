# LogCmd TUI 实现计划

本文档详细描述了 LogCmd 交互式终端界面 (TUI) 的设计与实施计划。

## 1. 愿景与目标

构建一个现代、美观且高效的终端交互界面，使用户能够：
- 浏览和管理所有注册项目
- 查看项目的命令执行历史
- 沉浸式阅读命令日志文件
- 实时监控和管理后台任务
- 无需离开终端即可进行复杂的搜索和统计查看

## 2. 技术栈选型

我们将采用 Go 语言社区最流行的 **Charm** 生态系统：

*   **核心框架**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) (基于 Elm 架构的 M-V-U 模式)
*   **样式引擎**: [Lipgloss](https://github.com/charmbracelet/lipgloss) (声明式布局与样式)
*   **组件库**: [Bubbles](https://github.com/charmbracelet/bubbles) (提供 List, Table, Viewport, TextInput 等标准组件)

## 3. 架构设计

### 3.1 目录结构

```text
internal/ui/
├── app.go              # TUI 程序入口 (Start 函数)
├── model.go            # 根 Model 定义 (包含全局状态)
├── update.go           # 根 Update 逻辑 (路由分发)
├── view.go             # 根 View 逻辑 (布局渲染)
├── common/             # 通用定义
│   ├── keys.go         # 全局快捷键定义
│   └── styles.go       # 全局 Lipgloss 样式
└── modules/            # 功能模块 (每个模块都是一个 bubble)
    ├── projectlist/    # 项目列表视图
    ├── historylist/    # 历史记录视图
    ├── logviewer/      # 日志查看器
    └── taskmanager/    # 任务管理器
```

### 3.2 状态管理 (State Management)

应用采用单一状态树，根 `Model` 持有当前激活的视图模式和各子模块的 Model。

```go
type SessionState int

const (
    ProjectListView SessionState = iota // 浏览项目列表
    HistoryListView                     // 浏览某项目的历史
    LogViewerView                       // 查看具体日志内容
    TaskListView                        // 查看后台任务
)

type Model struct {
    state        SessionState
    width        int
    height       int
    
    // 子模块 Models
    projectList  projectlist.Model
    historyList  historylist.Model
    logViewer    logviewer.Model
    taskManager  taskmanager.Model
    
    // 全局依赖
    registry     *registry.Registry
    historyMgr   *history.Manager
    tasksMgr     *tasks.Manager
}
```

## 4. 实施阶段 (Phases)

### 第一阶段：浏览器核心 (MVP) - ✅ 已完成
**目标**: 实现完整的“浏览”路径：项目列表 -> 历史列表 -> 日志详情。

1.  **基础架构**:
    - [x] 添加 `logcmd ui` 命令。
    - [x] 初始化 `internal/ui` 包结构。
    - [x] 定义基本的 Lipgloss 样式 (Theme)。

2.  **项目列表 (Project List)**:
    - [x] 集成 `internal/registry` 获取数据。
    - [x] 使用 `bubbles/list` 展示项目 (ID, 路径, 更新时间)。
    - [x] 实现 `Enter` 键选中项目并切换状态。

3.  **历史列表 (History List)**:
    - [x] 接收选中的 Project ID。
    - [x] 集成 `internal/history` 查询该项目的运行记录。
    - [x] 展示命令、时间、状态 (✅/❌)、耗时。
    - [x] 实现 `Esc` 返回项目列表，`Enter` 进入日志详情。

4.  **日志查看器 (Log Viewer)**:
    - [x] 读取选定历史记录的日志文件路径。
    - [x] 使用 `bubbles/viewport` 加载文件内容。
    - [x] 支持 `j/k` 滚动，`gg/G` 跳转，`/` 搜索内容。

### 第二阶段：交互与管理（大部分完成）
**目标**: 增加对项目和任务的操作能力。

1.  **任务监控 (Task Manager)**:
    - [x] 新增快捷键 `tab` 切换到任务视图，`esc`/`tab` 返回上一视图。
    - [x] 根 Model 注入 `tasks.Manager`，`taskmanager` 模块使用 `ListActive()` 拉取任务列表并在状态栏反馈刷新结果。
    - [x] 通过 `tea.Tick` 每 5 秒刷新，展示 ID / PID / Command / Status / Log。
    - [x] 支持 `s`（Stop）与 `k`（Kill）快捷键，调用 `internal/tasks/operations.StopTask`，操作后自动刷新列表。

2.  **数据操作**:
    - [x] 项目列表支持 `d` 一键清理无效项目（`registry.CheckAndCleanup`），并显示状态提示。
    - [x] 支持逐个删除项目（`x` 快捷键，确认后调用 `registry.Delete` 并自动刷新列表）。
    - [x] 历史列表增加筛选、状态过滤及更丰富的列展示（`f` 循环 All/Success/Failed，标题同步展示筛选状态）。

### 第三阶段：可视化与增强
**目标**: 提供更直观的数据展示。

1.  **UX 改进 (Layout & Responsiveness)**:
    - [x] 重构布局系统，支持响应式设计（根据窗口宽度自动切换 紧凑/垂直/水平 布局）。
    - [x] 优化状态栏和主内容区域的分割，防止溢出。

2.  **统计面板**:
    - 在项目列表侧边或底部显示选中项目的简单统计 (成功率 bar chart)。
    - 历史命令分布图。

3.  **高级搜索**:
    - 集成 `internal/search`，提供全局搜索界面。

## 5. 关键集成点

*   **数据源**: 必须重用现有的 `internal/` 包，不可复制逻辑。
    *   `registry.NewRegistry()`: 获取项目列表。
    *   `history.NewManager()`: 获取历史记录。
    *   `tasks.NewManager()`: 获取后台任务。
*   **配置**: TUI 应尊重现有的全局配置 (如数据库位置)。

## 6. 下一步行动 (Action Plan)

根据当前代码审查，第一阶段和第二阶段的大部分核心功能已就绪。接下来的工作重点是完善用户体验和数据管理能力。

### 优先任务
1.  **统计信息预览 (Preparation for Phase 3)**:
    *   设计 Statistics Model，计划在 Project List 界面右侧或底部展示选中项目的 `Success Rate` 和 `Total Runs` 可视化组件。

### 待定任务
*   全局搜索界面 (集成 `internal/search`)。
