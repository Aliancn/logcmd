# 任务完成时需要执行的步骤
1. **格式化/静态检查**: 对修改过的 Go 文件运行 `gofmt -w`（或 `make fmt`），如涉及 lint 规则可再执行 `make lint`（需要 golangci-lint）。
2. **单元测试**: 至少 `go test ./...` 或 `make test`（等价于 `go test -v ./test/go_module_test/...`）；若改动影响 CLI 流程可再运行 `make test-scenarios` 触发 shell 场景测试。
3. **构建/验证 CLI**: 需要可执行文件时运行 `make build`（生成 `bin/logcmd`）或 `go run cmd/logcmd/main.go ...` 做本地验证。
4. **自检**: 查看 `git status`/`git diff`，确保没有无关变化，确认中文提示与 CLI UX 一致。