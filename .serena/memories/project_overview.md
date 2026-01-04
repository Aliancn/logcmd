# LogCmd 项目概览
- **定位**: 纯 Go 实现的 CLI，用于执行命令并自动记录日志、支持搜索与统计，核心针对在本地项目中统一整理命令输出。
- **技术栈**: Go modules、Cobra CLI、SQLite(通过 internal/registry & migration)、自研 logger/persistence 层；单元测试放在 `test/go_module_test`，端到端 Shell 场景测试位于 `test/scenarios`。
- **结构**:
  - `cmd/logcmd`: CLI 入口 (`main.go`) 以及所有子命令实现（run/task/search/stats/tail/project/config 等）。
  - `internal/config|logger|registry|tasks|persistence|services|stats|template`: 领域与基础设施模块，覆盖配置加载、日志执行、SQLite 管理、后台任务、统计分析和模板渲染。
  - `bin/` 为构建产物，`.logcmd` 目录由运行时生成，`docs/`、`ROADMAP.md`、`TODO.md` 记录设计/规划。
  - `test/go_module_test`：Go 单测包按功能分类；`test/scenarios`：基于 shell 的端到端测试脚本集。
- **运行方式**: 通过 `logcmd` 可执行文件（`make install` 或 `go run cmd/logcmd/main.go ...`）提供命令执行、日志搜索、项目管理等能力；默认会在当前/父目录寻找 `.logcmd` 目录并注册项目。
- **依赖/外部接口**: 主要依靠本地文件系统、SQLite 数据库，无远程服务依赖。