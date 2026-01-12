# LogCmd 功能开发规划路线图 (Roadmap)

本文档旨在规划 `logcmd` 的未来发展方向，目标是将其打造为一个功能完备、使用便捷的命令行日志管理专家。

## P0: 核心功能增强 (Core Improvements)
*基础架构的完善，确保工具的长期可用性和可维护性。*

- [x] **日志清理 (Log Rotation)**
    - **已实现**:
        - `logcmd clean` 命令，支持按天数 (`--days`) 和数量 (`--keep`) 手动清理。
        - 自动清理配置: `max_retention_days` 和 `max_retention_count`，每次执行命令时在后台自动维护。

## P1: 交互体验升级 (Interactive Experience) ✅
*从"命令行工具"进化为"终端应用"，大幅提升易用性。*

- [x] **交互式终端界面 (TUI)**
    - **需求**: `search` 和 `list` 命令的输出是静态的，查看详情需要复制路径再打开，体验割裂。
    - **功能**: 引入 TUI 库 (bubbletea + bubbles)。
        - **极简模式系统**: 类似 vim 的设计，单一焦点模式
        - **SearchMode**: 实时模糊搜索，支持全项目/单项目范围切换
        - **LogViewMode**: 完整日志查看，关键词高亮，自动定位
        - **ProjectMode**: 项目管理，快速切换到项目日志
        - **TaskMode**: 后台任务管理，实时日志预览
        - **StatsMode**: 统计分析，支持全局和项目级统计
        - **CommandMode**: vim 风格命令输入（`:search`, `:quit` 等）
        - **快捷键**: `/` 搜索, `p` 项目, `t` 任务, `s` 统计, `:` 命令, `q` 退出
    - **命令**: `logcmd ui`
    - **状态**: ✅ 已完成 (Phase 1-4)
    - **架构文档**: [UI_REDESIGN_MINIMAL.md](./docs/UI_REDESIGN_MINIMAL.md)

## P2: 高级分析与处理 (Advanced Analysis)
*挖掘日志数据的价值。*

- [ ] **高级搜索语法**
    - **需求**: 现有的 regex 可能对普通用户有门槛。
    - **功能**: 支持逻辑运算符，如 `error AND timeout`，`database NOT connection`。
- [x] **结构化数据导出**
    - **已实现**: `logcmd search ... --format=json`，输出 JSON 格式的搜索结果数组。
- [ ] **错误特征聚类**
    - **需求**: 快速识别最常见的错误模式。
    - **功能**: 自动分析日志中的 Error 模式，归纳为"Top 5 常见错误类型"。

## P3: 生态与集成 (Ecosystem)
*拓展使用场景。*

- [ ] **Web 可视化面板 (Web Dashboard)**
    - **需求**: 在浏览器中查看日志，适合团队共享或大屏展示。
    - **功能**: `logcmd server` 启动一个本地 Web 服务，提供图表和日志浏览界面。
- [x] **Shell钩子/别名集成**
    - **已实现**: `logcmd alias` 生成 shell 别名脚本 (如 `lrun`, `lsearch`)。

## 待讨论特性 (Ideas Pool)

- **远程日志同步**: 支持将本地日志同步到 S3 或远程服务器备份。
- **插件系统**: 允许编写 Lua/Python 脚本对日志流进行实时处理（如触发系统通知）。
- **Diff 模式**: 比较两次运行（如两次构建）的日志差异。
