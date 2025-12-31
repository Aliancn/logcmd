# LogCmd 功能开发规划路线图 (Roadmap)

本文档旨在规划 `logcmd` 的未来发展方向，目标是将其打造为一个功能完备、使用便捷的命令行日志管理专家。

## P0: 核心功能增强 (Core Improvements)
*基础架构的完善，确保工具的长期可用性和可维护性。*

- [ ] **日志自动清理 (Log Rotation & Retention)**
    - **需求**: 防止日志无限增长占用磁盘空间。
    - **功能**:
        - 支持按时间保留（如：保留最近 30 天）。
        - 支持按数量保留（如：保留最近 1000 条）。
        - 支持按大小保留（如：项目日志总大小限制为 1GB）。
        - 提供 `logcmd clean --auto` 命令及配置项。
- [ ] **配置文件支持 (Configuration Management)**
    - **需求**: 目前主要依赖命令行参数，缺乏持久化的个性化配置。
    - **功能**:
        - 支持全局配置 `~/.logcmd/config.yaml`。
        - 支持项目级配置 `.logcmd/config.yaml`。
        - 配置项包括：默认保留策略、默认排除模式、颜色主题等。
- [ ] **实时日志流 (Live Tailing)**
    - **需求**: 类似于 `tail -f`，但在 logcmd 的管理上下文中查看正在运行的命令。
    - **功能**: `logcmd tail <id>` 或 `logcmd attach <id>`。

## P1: 交互体验升级 (Interactive Experience)
*从“命令行工具”进化为“终端应用”，大幅提升易用性。*

- [ ] **交互式终端界面 (TUI)**
    - **需求**: `search` 和 `list` 命令的输出是静态的，查看详情需要复制路径再打开，体验割裂。
    - **功能**: 引入 TUI 库 (如 bubbletea)。
        - **Dashboard**: 键盘上下选择项目/日志。
        - **Preview**: 选中日志即时预览内容。
        - **Filter**: 界面内直接输入关键词过滤。
    - **命令**: `logcmd ui` 或 `logcmd browse`。
- [ ] **智能补全 (Shell Completion)**
    - **需求**: 提高输入效率。
    - **功能**: 生成 Bash/Zsh/Fish 补全脚本，支持自动补全子命令、项目路径甚至最近的日志 ID。

## P2: 高级分析与处理 (Advanced Analysis)**
*挖掘日志数据的价值。*

- [ ] **高级搜索语法**
    - **需求**: 现有的 regex 可能对普通用户有门槛。
    - **功能**: 支持逻辑运算符，如 `error AND timeout`，`database NOT connection`。
- [ ] **结构化数据导出**
    - **需求**: 便于与其他工具集成。
    - **功能**: `logcmd search ... --format=json|csv`，方便导入 Excel 或 ELK。
- [ ] **错误特征聚类**
    - **需求**: 快速识别最常见的错误模式。
    - **功能**: 自动分析日志中的 Error 模式，归纳为“Top 5 常见错误类型”。

## P3: 生态与集成 (Ecosystem)
*拓展使用场景。*

- [ ] **Web 可视化面板 (Web Dashboard)**
    - **需求**: 在浏览器中查看日志，适合团队共享或大屏展示。
    - **功能**: `logcmd server` 启动一个本地 Web 服务，提供图表和日志浏览界面。
- [ ] **Shell钩子/别名集成**
    - **需求**: 某些关键命令（如 `make`, `mvn`, `npm install`）用户可能忘记加 `logcmd` 前缀。
    - **功能**: 提供 helper 脚本，允许用户设置 alias，自动拦截特定命令并记录。
        - **示例**: `alias make="logcmd run make"`。

## 待讨论特性 (Ideas Pool)

- **远程日志同步**: 支持将本地日志同步到 S3 或远程服务器备份。
- **插件系统**: 允许编写 Lua/Python 脚本对日志流进行实时处理（如触发系统通知）。
- **Diff 模式**: 比较两次运行（如两次构建）的日志差异。
