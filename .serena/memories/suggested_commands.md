# 常用命令
- **构建/安装**: `make build`（生成 `bin/logcmd`）、`make install`（安装到 `$GOBIN`）、`go run cmd/logcmd/main.go echo hi`（快速运行）。
- **测试**: `make test`（`go test -v ./test/go_module_test/...`）、`go test ./...`、`make test-scenarios` / `make test-scenarios-<suite>` 运行端到端脚本、`make test-coverage` 生成覆盖率。
- **质量检查**: `make fmt`（`go fmt ./...`）、`make lint`（`golangci-lint run`）。
- **CLI 使用**: `logcmd run <cmd>`、`logcmd run -d <cmd>`（后台任务）、`logcmd task list|stop|kill`、`logcmd search --keyword <kw>`、`logcmd stats`, `logcmd project list|clean|delete`、`logcmd config set|get|list`。
- **系统/辅助**: macOS 上常用 `ls`, `rg <pattern> -n`, `git status`, `git diff`, `make clean` 清理构建缓存。