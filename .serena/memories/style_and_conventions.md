# 样式与约定
- **编码**: Go 1.x，保持 gofmt/goimports 风格，遵循 Go 命名（驼峰、首字母大写导出）。CLI 文案统一使用简体中文。
- **结构化 CLI**: 所有子命令通过 Cobra 定义在 `cmd/logcmd/cmd` 下，共享的初始化/依赖（配置、registry、任务管理器等）应抽取成辅助函数，强调 KISS/DRY。
- **错误处理**: 使用 `fmt.Errorf("...: %w", err)` 包装，CLI 层通过自定义 `ExitError` 控制退出码；需要根据 context 取消和信号处理中断。
- **配置/模型**: 倾向使用 Go 原生类型或指针表达可选字段，数据库转换放在 persistence 层完成，避免在 model 中暴露 `sql.Null*`。
- **测试**: 单测位于 `test/go_module_test/<module>`，使用 `go test` 运行；端到端脚本在 `test/scenarios`，必要时需先 `make build`。
- **格式化/静态检查**: `make fmt`（`go fmt ./...`）和 `make lint`（`golangci-lint run`）是默认校验方式。