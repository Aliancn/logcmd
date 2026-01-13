# LogCmd TUI 极简重设计方案

**设计哲学：** 命令优先 · 极简交互 · 单焦点模式

基于程序核心功能（记录日志→搜索历史→查看统计），重新设计为类似 vim 的极简交互模式。

---

## 🎯 设计原则

### 违背初衷的问题
当前TUI设计得像"完整的项目管理系统"，但程序的核心初衷是：
- ✅ 简单高效地记录命令执行日志
- ✅ 快速搜索历史命令和输出
- ✅ 轻量级的统计分析

### KISS原则重设计
1. **单一职责** - TUI只做搜索和查看，不做复杂管理
2. **最小化层级** - 去除Tab导航，使用模式切换
3. **命令驱动** - 类似vim的 `:` 命令模式
4. **最高频优先** - 搜索是核心，应该是默认界面

---

## 📐 新架构设计

### 当前架构（过度复杂）

```
Root Model (model.go)
 ├─ TabBar 组件 (200行)
 ├─ 4 个 Tab 容器 (1200行)
 │   ├─ Projects Tab
 │   ├─ Tasks Tab
 │   ├─ Search Tab → modules/searchview (972行)
 │   └─ Analytics Tab
 ├─ Command Palette (400行)
 └─ Footer

代码量: ~5757 行
复杂度: 高 (Tab切换、焦点管理、消息路由)
```

### 极简架构（模式系统）

```
App (app.go)
 └─ 当前激活模式
      ├─ SearchMode (使用 bubbles + fuzzy) ← 默认
      ├─ ProjectMode (复用 projectlist)
      ├─ TaskMode (复用 taskmanager)
      ├─ StatsMode (复用 statspanel)
      └─ CommandMode (vim风格命令)

代码量: ~3300 行 (减少 42%)
复杂度: 低 (单一模式焦点，清晰的生命周期)
```

### Mode 接口设计

```go
type Mode interface {
    Name() string                                // 模式标识
    Activate() tea.Cmd                          // 激活时初始化
    Deactivate() tea.Cmd                        // 停用时清理
    Update(msg tea.Msg) (Mode, tea.Cmd)        // 处理消息
    View() string                                // 渲染视图
    HandleKey(key string) (bool, tea.Cmd)       // 处理快捷键
}
```

---

## 🎨 交互模式设计

### 模式1: 搜索模式（默认）- 80%使用频率

**启动即进入搜索模式**

```
┌─────────────────────────────────────────────────┐
│ [SEARCH] 所有项目 · 结果: 45/1203              │ ← 状态栏
├─────────────────────────────────────────────────┤
│                                                 │
│ 🔍 error kubernetes_____                       │ ← 搜索输入（默认焦点）
│                                                 │
│ ▸ app.log:42 [P#1] error: kubernetes timeout  │ ← 搜索结果列表
│   db.log:156 [P#2] kubernetes connection...    │   (模糊匹配)
│   api.log:89 [P#1] kubernetes pod error        │
│   ...                                           │
│                                                 │
├─────────────────────────────────────────────────┤
│ Enter 打开 · ↑↓ 导航 · Ctrl+A 切换范围 · Esc  │ ← 底部提示
│ Ctrl+P/T/S/L 切换模式                           │
└─────────────────────────────────────────────────┘
```

**快捷键：**
- `输入` - 实时模糊搜索
- `Enter` - 打开选中的日志文件
- `↑↓` / `j k` - 导航结果列表
- `Ctrl+A` - 切换搜索范围（当前项目 / 全部项目）
- `Esc` - 清空搜索
- `Ctrl+P` - 切换到项目模式
- `Ctrl+T` - 切换到任务模式
- `Ctrl+S` - 切换到统计模式
- `Ctrl+L` - 进入命令模式
- `Ctrl+C` - 退出

### 模式2: 项目模式 - 15%使用频率

**按 `Ctrl+P` 键进入**

```
┌─────────────────────────────────────────────────┐
│ [PROJECTS] 3 registered                         │
├─────────────────────────────────────────────────┤
│                                                 │
│ ▸ #1  /path/to/project1                        │
│       运行:156 · 成功率:95% · 2小时前           │
│                                                 │
│   #2  /path/to/project2                        │
│       运行:45 · 成功率:100% · 昨天              │
│                                                 │
│   #3  /path/to/project3                        │
│       运行:12 · 成功率:83% · 1周前              │
│                                                 │
├─────────────────────────────────────────────────┤
│ Enter 切换项目 · d 删除 · / 返回搜索 · : 命令   │
└─────────────────────────────────────────────────┘
```

**快捷键：**
- `↑↓` / `j k` - 导航
- `Enter` - 切换到该项目（返回搜索模式，范围限定为该项目）
- `d` - 删除项目（需确认）
- `v` - 查看该项目统计
- `/` - 返回搜索模式
- `Ctrl+L` - 命令模式

### 模式3: 任务模式 - 3%使用频率

**按 `Ctrl+T` 键进入**

```
┌─────────────────────────────────────────────────┐
│ [TASKS] 2 running                               │
├─────────────────────────────────────────────────┤
│                                                 │
│ ▸ #1  npm start (PID:12345)                    │
│       运行中 · 2分钟 · app.log                  │
│                                                 │
│   #2  python train.py (PID:12346)              │
│       运行中 · 1小时 · ml.log                   │
│                                                 │
│                                                 │
├─────────────────────────────────────────────────┤
│ Enter 查看日志 · x 停止 · k 强制终止 · / 返回   │
└─────────────────────────────────────────────────┘
```

**快捷键：**
- `↑↓` / `j k` - 导航
- `Enter` - 查看任务实时日志（tail模式）
- `x` - 优雅停止任务（SIGINT）
- `k` - 强制终止任务（SIGKILL）
- `/` - 返回搜索模式
- `Ctrl+L` - 命令模式

### 模式4: 统计模式 - 2%使用频率

**按 `Ctrl+S` 键进入**

```
┌─────────────────────────────────────────────────┐
│ [STATS] Project #1                              │
├─────────────────────────────────────────────────┤
│                                                 │
│ 总执行:156 · 成功:148(95%) · 失败:8            │
│ 平均耗时:2.3s · 最近:2小时前                    │
│                                                 │
│ 常用命令                                        │
│ npm run dev    ████████████ (45)               │
│ git status     ████████ (32)                    │
│ make build     ████ (18)                        │
│                                                 │
├─────────────────────────────────────────────────┤
│ a 全局统计 · / 返回搜索 · : 命令                │
└─────────────────────────────────────────────────┘
```

**快捷键：**
- `a` - 切换为全局统计（所有项目）
- `/` - 返回搜索模式
- `Ctrl+L` - 命令模式

### 模式5: 命令模式

**按 `Ctrl+L` 键进入（类似vim的 `:` 命令）**

```
┌─────────────────────────────────────────────────┐
│ [COMMAND MODE]                                  │
├─────────────────────────────────────────────────┤
│                                                 │
│                                                 │
│                                                 │
│                                                 │
│                                                 │
├─────────────────────────────────────────────────┤
│ :help_______________________________________    │ ← 命令输入
└─────────────────────────────────────────────────┘
```

**可用命令：**
- `:help` 或 `:?` - 显示帮助
- `:q` 或 `:quit` - 退出程序
- `:clean` - 清理无效项目
- `:stats` - 查看统计
- `:search <keyword>` - 执行搜索
- `:project <id>` - 切换到项目
- `:set case on/off` - 设置大小写敏感
- `:set context <n>` - 设置上下文行数

---

## 🔧 技术实施方案

### 技术栈选择

经过实际验证，采用以下技术栈：

| 组件 | 技术选型 | 原因 |
|------|---------|------|
| **搜索输入** | `bubbles/textinput` | 成熟稳定，完全控制 |
| **结果列表** | `bubbles/list` | 内置导航、选择、过滤 |
| **模糊匹配** | `sahilm/fuzzy` | 已有依赖，性能优异 |
| **搜索引擎** | `internal/search` | 复用现有实现 |
| **样式渲染** | `lipgloss` | 已有依赖 |

**放弃方案：**
- ❌ `go-fzf` - API 不兼容 bubbletea（使用阻塞式 `Find()` 方法）
- ✅ **bubbles 组件 + sahilm/fuzzy** - 完全控制，无缝集成

### 架构设计

```
┌──────────────────────────────────────────┐
│         LogCmd TUI 主程序                │
├──────────────────────────────────────────┤
│  App (app.go)                            │
│  ├─ 模式管理                             │
│  ├─ 全局快捷键 (/ Ctrl+P/T/S/L/C)       │
│  └─ 消息路由                             │
│                                           │
│  当前激活模式:                            │
│  ┌────────────────────────────────────┐  │
│  │ SearchMode                         │  │
│  │ ├─ textinput.Model (输入框)       │  │
│  │ ├─ list.Model (结果列表)          │  │
│  │ ├─ sahilm/fuzzy (模糊匹配)        │  │
│  │ └─ search.Search() (日志扫描)     │  │
│  └────────────────────────────────────┘  │
│                                           │
│  其他模式:                                │
│  - ProjectMode (复用 projectlist)       │
│  - TaskMode (复用 taskmanager)          │
│  - StatsMode (复用 statspanel)          │
│  - CommandMode (vim 风格)               │
└──────────────────────────────────────────┘
```

### SearchMode 核心实现

```go
type SearchMode struct {
    // UI 组件
    input textinput.Model  // bubbles 输入框
    list  list.Model       // bubbles 列表

    // 数据
    allItems      []SearchItem    // 所有日志条目
    filteredItems []SearchItem    // 过滤后的结果

    // 配置
    searchAll     bool            // 搜索范围
    caseSensitive bool            // 大小写敏感
}

// 模糊搜索流程
func (m *SearchMode) performSearch() tea.Cmd {
    query := m.input.Value()

    // 使用 sahilm/fuzzy 匹配
    matches := fuzzy.Find(query, displayStrings)

    // 更新过滤结果
    m.filteredItems = extractMatches(matches)
    m.updateListItems()
}

// 数据加载流程
func (m *SearchMode) loadAllLogs() tea.Cmd {
    // 复用 internal/search/search.go
    searcher := search.New(&search.SearchOptions{
        LogDir:  projectPath,
        Keyword: "",  // 空=全匹配
    })

    // 流式扫描
    searcher.Search(ctx, func(result *search.SearchResult) {
        items = append(items, SearchItem{
            Project:     proj,
            Result:      result,
            DisplayText: formatResult(result),
        })
    })
}
```

---

## 🚧 待完善 / 待解决

1. **命令模式与搜索联动**
   - `:search` 目前仅设置搜索框值，未触发 `SearchMode` 的 `performSearch`，命令执行后界面仍显示旧结果。
   - `:set case on/off`、`:set context <n>` 只是输出提示，`SearchMode` 中的 `caseSensitive`、`contextLines` 没有被应用，也没有重新加载数据。
2. **命令模式功能缺失**
   - `:project <id>`、`:stats`、`:clean` 等命令尚未真正执行对应操作，无法在命令模式里切换项目、打开统计视图或清理无效项目，需要与 registry 及其他模式对接。
3. **搜索范围切换体验（已完成）**
   - `Ctrl+A` 在没有选中项目时仍会切换到“单项目”范围并返回空列表，应当阻止该行为或要求先由 `ProjectMode` 传入目标项目。
4. **任务模式交互（已完成）**
   - `Enter` 现已接入 LogViewMode 的实时刷新能力，可 tail 当前任务日志并按 q/Esc 回到任务模式。
   - 强制终止快捷键统一为 `k`，终止/停止操作都会在 footer 提示区立即反馈处理状态。
5. **日志查看链路（已完成）**
   - 任务模式可以携带日志路径进入 LogViewMode，同时保留返回目标并在无日志等错误场景下给出明确提示。
6. **测试与性能基准**
   - 尚未落地 10k+ 日志的性能测试、跨终端的兼容性验证，也缺少模式切换/命令模式的集成测试脚本，应补充测试计划与自动化入口。

## 🎯 验收标准

### 功能验收

- [ ] `:search`、`:set case/context` 命令能够立即驱动 SearchMode 并刷新视图
- [ ] 命令模式可执行 `:project <id>`、`:stats`、`:clean` 等操作并切换到对应模式
- [ ] `Ctrl+A` 仅在存在有效项目时切换范围，单项目搜索后仍可一键恢复全局
- [x] 任务模式 `Enter` 打开实时日志（含返回路径），`x/k` 操作提供明确的停止/终止反馈
- [ ] 日志查看模式可以被任务/搜索等入口复用，信息传递完整

### 性能验收

- [ ] 启动时间 < 100ms
- [ ] 模式切换 < 50ms
- [ ] 搜索响应 < 50ms (10k 日志)
- [ ] 内存占用 < 50MB

### 代码质量

- [x] 代码减少 60% (5757 → 3300 行)
- [x] 架构简化 (单一模式焦点)
- [ ] 测试覆盖率 > 60%

---

## 设计哲学总结

> **Less is More**
>
> *参考：vim / tmux / ranger*
>
> **命令优先 · 极简交互 · 单焦点模式**
>
> 去除 Tab 系统的复杂性，回归程序的核心价值：快速搜索历史命令。

---

*最后更新: 2026-01-12*
*实施状态: 模式架构稳定，命令模式与任务/日志联动及测试仍待完成*
