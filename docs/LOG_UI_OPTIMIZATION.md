# 日志 UI 优化方案 (Log UI Optimization Plan)

## 1. 现状分析 (Current Status Analysis)

通过对 `internal/ui/modules/logviewer` (日志详情) 和 `internal/ui/modules/taskmanager` (任务管理预览) 的代码审查，识别出以下关键痛点：

### 1.1 性能瓶颈
- **全量加载风险**: `LoadContentCmd` 使用 `os.ReadFile` 一次性读取整个日志文件到内存。对于数百 MB 或 GB 级别的生产环境日志，这将导致严重的内存消耗（OOM）甚至程序崩溃。
- **UI 阻塞**: 大量文本的正则处理和渲染在主线程进行，可能导致界面卡顿。

### 1.2 视觉体验局限
- **硬编码高亮**: 当前的高亮逻辑 (`styleLine`) 基于简单的字符串包含检查和手动 Regex，难以维护且不支持复杂的语法结构。
- **JSON 支持薄弱**: 现有的 JSON 格式化 (`Ctrl+J`) 实现脆弱（基于简单的花括号查找），缺乏专业的色彩高亮，且容易破坏非标准 JSON 行的显示。
- **预览不一致**: 任务管理器中的日志预览 (`taskmanager`) 仅截取末尾 2KB 文本，且无任何语法高亮，体验与详情页割裂。

## 2. 优化目标 (Optimization Goals)

1.  **高性能 (Performance)**: 支持 GB 级大文件秒开，内存占用恒定。
2.  **专业视觉 (Visuals)**: 实现类似 IDE 的语法高亮，支持 Logfmt, JSON, HTTP Trace 等常见格式。
3.  **交互增强 (UX)**: 提供行号、智能搜索高亮、自动换行及实时日志跟随 (`tail -f`)。

## 3. 技术选型 (Technology Stack)

为了实现上述目标，建议引入以下成熟的 Go 社区库：

| 功能 | 推荐库 | 优势 |
| :--- | :--- | :--- |
| **通用语法高亮** | [**alecthomas/chroma**](https://github.com/alecthomas/chroma) | 纯 Go 实现，兼容 Pygments 样式，支持数百种语言（含 Log, JSON），输出 ANSI 颜色序列，性能优异。 |
| **JSON 美化** | [**tidwall/pretty**](https://github.com/tidwall/pretty) | 专为高性能设计，支持终端彩色输出，比标准库快且更适合 CLI 展示。 |
| **大文件读取** | 原生 `os.File` + `Seek` | 通过分块读取（Chunked Reading）或内存映射（Mmap）实现按需加载。 |
| **实时日志** | [**hpcloud/tail**](https://github.com/hpcloud/tail) | (可选) 如果需要实现类似 `tail -f` 的实时滚动功能。 |

## 4. 执行路线图 (Execution Roadmap)

### 阶段一：视觉升级 (Phase 1: Visual Upgrade)
**目标**: 提升现有小文件的阅读体验，替换手动高亮逻辑。
1.  **集成 Chroma**: 替换 `styleLine` 方法，使用 Chroma 的 `lexers.Fallback` 或自定义 Log Lexer 对日志行进行着色。
2.  **集成 Pretty JSON**: 重构 `Ctrl+J` 逻辑，使用 `tidwall/pretty` 对检测到的 JSON 块进行格式化和高亮。
3.  **统一预览组件**: 提取一个公共的 `LogView` 组件，供详情页和任务管理器预览共用，确保体验一致。

### 阶段二：大文件支持 (Phase 2: Large File Support)
**目标**: 解决内存溢出问题，支持大文件浏览。
1.  **实现分页读取 (Pager)**: 修改 `model.go`，不再存储 `content string` 全文。
    - 仅维护当前 Viewport 所需的 `[]byte` 缓冲区（例如：前后各预加载 10KB）。
    - 监听滚动事件，动态触发 `Seek` 和 `Read` 操作。
2.  **文件末尾优先**: 默认打开文件时自动 Seek 到末尾（Tail 模式），符合查看日志的习惯。
3.  **加载指示器**: 在数据加载时显示 Loading 状态。

### 阶段三：交互细节 (Phase 3: UX Refinements)
**目标**: 增加高级辅助功能。
1.  **行号显示 (Gutter)**: 在 Viewport 左侧增加行号栏。
2.  **搜索增强**: 
    - 搜索时不仅跳转，还要高亮视口内所有匹配的关键词（背景色高亮）。
    - 支持 `n`/`N` 在匹配项间跳转。
3.  **自动换行 (Soft Wrap)**: 增加 `Ctrl+W` 开关，支持超长日志行的软换行显示。

## 5. 架构调整建议

建议将 UI 渲染与数据获取解耦：

```go
// 建议的新结构
type LogModel struct {
    // 核心组件
    viewport   viewport.Model
    highlighter *Highlighter // 封装 Chroma
    fileHandler *FileHandler // 封装 Seek/Read 逻辑

    // 状态
    offset     int64 // 当前文件读取位置
    buffer     []byte // 当前内存中的数据片段
    ...
}
```
