# 日志 UI 优化方案 (Log UI Optimization Plan)

## 1. 现状回顾与进展 (Status Overview)

项目已完成核心的 UI 现代化改造与性能优化。通过引入 `viewport` 虚拟化与 `ChunkedReader` 分块读取，成功解决了大文件加载的内存与卡顿问题；通过集成 `Chroma` 与自定义 JSON 格式化器，极大提升了日志的可读性。

### 已解决的痛点 (Resolved Issues)
- [x] **性能瓶颈**: 实现了 >10MB 文件的自动分块读取 (`ChunkedReader`) 与虚拟滚动，不再一次性加载全量文件到内存。
- [x] **视觉体验**: 替换了硬编码的高亮逻辑，全面集成 `alecthomas/chroma` 实现语法高亮。
- [x] **JSON 支持**: 实现了 `Ctrl+J` 智能 JSON 格式化与高亮，支持从混合文本中提取 JSON。
- [x] **交互增强**: 增加了行号显示 (`Ctrl+L`)，统一了详情页与任务预览页的组件体验。

## 2. 剩余优化目标 (Remaining Optimization Goals)

虽然核心架构已就绪，但在交互细节与高级功能上仍有提升空间。

### 2.1 交互体验 (UX)
1.  **搜索高亮 (Visual Search Highlight)**:
    - 当前搜索仅实现了跳转 (`Jump`)。
    - **目标**: 在 Viewport 内对匹配的关键词应用背景色高亮（如黄色背景），便于用户快速定位上下文中的关键词。
2.  **软换行 (Soft Wrap)**:
    - 代码中已预留 `Ctrl+W` 快捷键，但逻辑尚未实现。
    - **目标**: 支持长日志行的自动折行显示，避免用户频繁左右滚动。
3.  **指定行跳转 (Go to Line)**:
    - 底层已支持 `JumpToLine`。
    - **目标**: 增加 UI 交互（如输入 `:100`），允许用户跳转到指定行号。

### 2.2 性能进阶 (Performance Advanced)
1.  **大文件搜索优化**:
    - 当前大文件搜索采用线性全量扫描 (`ReadLines` loop)，对于 GB 级文件耗时较长且阻塞 UI。
    - **目标**: 实现后台异步搜索，或利用 `ripgrep` 等外部工具/索引加速搜索。

## 3. 执行路线图 (Execution Roadmap)

### 阶段一：视觉升级 (Phase 1: Visual Upgrade) - [COMPLETED]
- [x] **集成 Chroma**: 使用 `highlighter.ChromaHighlighter` 替代旧逻辑。
- [x] **集成 Pretty JSON**: 实现 `formatter.JSONFormatter` 及 `Ctrl+J` 交互。
- [x] **统一预览组件**: 任务管理器与日志详情页共享高亮与渲染逻辑。

### 阶段二：大文件支持 (Phase 2: Large File Support) - [COMPLETED]
- [x] **实现分页读取 (Pager)**: `logviewer` 模块已实现 `ChunkedReader` 与 `viewport` 虚拟化。
- [x] **加载指示器**: 实现了索引构建时的状态提示。

### 阶段三：交互细节 (Phase 3: UX Refinements) - [IN PROGRESS]
- [x] **行号显示 (Gutter)**: `Ctrl+L` 已实现。
- [ ] **搜索增强**: 
    - [x] 基础跳转 (n/N)
    - [ ] 关键词视觉高亮 (Highlight matches in viewport)
    - [ ] 异步非阻塞搜索 (Async Search)
- [ ] **自动换行 (Soft Wrap)**: 实现 `Ctrl+W` 的折行渲染逻辑。
- [ ] **行号跳转**: 实现类似 Vim 的 `:nnnn` 跳转指令。

## 4. 架构快照 (Architecture Snapshot)

当前的 `LogModel` 结构已更新为：

```go
type Model struct {
    // 核心组件
    viewport    viewport.Model
    
    // 服务层
    highlighter *highlighter.ChromaHighlighter
    formatter   *formatter.JSONFormatter

    // 虚拟滚动（大文件）
    chunkedReader  *reader.ChunkedReader
    usesChunked    bool     // 是否启用分块
    cachedLines    []string // 当前 Viewport 缓存行
    bufferSize     int      // 预加载缓冲窗口
    
    // 状态
    searchMatches []int // 搜索结果索引
    // ...
}
```