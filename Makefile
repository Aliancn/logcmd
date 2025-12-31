.PHONY: build install clean test

# 变量定义
BINARY_NAME=logcmd
BUILD_DIR=./bin
CACHE_DIR=$(abspath ./.cache)
GO_BUILD_CACHE=$(CACHE_DIR)/go-build
GO_BIN:=$(shell go env GOBIN)
ifeq ($(GO_BIN),)
GO_BIN:=$(shell go env GOPATH)/bin
endif
INSTALL_PATH?=$(GO_BIN)

# 默认目标
all: build

# 编译
build:
	@echo "正在编译 $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@mkdir -p $(GO_BUILD_CACHE)
	GOCACHE=$(GO_BUILD_CACHE) go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) cmd/logcmd/main.go
	@echo "编译完成: $(BUILD_DIR)/$(BINARY_NAME)"

# 安装到系统
install: build
	@echo "正在安装到 $(INSTALL_PATH)..."
	@mkdir -p $(INSTALL_PATH)
	@install -m 0755 $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "安装完成！"
	@echo "请确认已将 $(INSTALL_PATH) 加入 PATH 后运行: logcmd <command>"

# 运行测试
test:
	@echo "运行测试..."
	@echo "======================================"
	go test -v ./test/...
	@echo "======================================"
	@echo "所有测试完成！"

# 运行测试并显示覆盖率
test-coverage:
	@echo "运行测试并生成覆盖率报告..."
	@mkdir -p coverage
	go test -v -coverprofile=coverage/coverage.out ./test/...
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "覆盖率报告已生成: coverage/coverage.html"

# 运行特定模块的测试
test-config:
	@echo "测试 config 模块..."
	go test -v ./test/config/...

test-executor:
	@echo "测试 executor 模块..."
	go test -v ./test/executor/...

test-model:
	@echo "测试 model 模块..."
	go test -v ./test/model/...

test-registry:
	@echo "测试 registry 模块..."
	go test -v ./test/registry/...

test-history:
	@echo "测试 history 模块..."
	go test -v ./test/history/...

test-template:
	@echo "测试 template 模块..."
	go test -v ./test/template/...

test-stats:
	@echo "测试 stats 模块..."
	go test -v ./test/stats/...

test-search:
	@echo "测试 search 模块..."
	go test -v ./test/search/...

# 快速测试（不显示详细输出）
test-quick:
	@echo "快速测试..."
	go test ./test/...

# 场景功能测试（端到端测试）
test-scenarios: build
	@echo "运行场景功能测试..."
	@./test/scenarios/run_all.sh

# 运行特定场景测试
test-scenarios-basic: build
	@echo "运行基础场景测试..."
	@./test/scenarios/run_all.sh basic

test-scenarios-project: build
	@echo "运行项目管理场景测试..."
	@./test/scenarios/run_all.sh project

test-scenarios-stats: build
	@echo "运行统计和搜索场景测试..."
	@./test/scenarios/run_all.sh stats

test-scenarios-template: build
	@echo "运行模板配置场景测试..."
	@./test/scenarios/run_all.sh template

test-scenarios-long-log: build
	@echo "运行大文件输出性能测试..."
	@./test/scenarios/run_all.sh long_log

# 运行所有测试（单元测试 + 场景测试）
test-all: test test-scenarios
	@echo ""
	@echo "======================================"
	@echo "所有测试（单元测试 + 场景测试）完成！"
	@echo "======================================"

# 清理构建产物
clean:
	@echo "清理构建产物..."
	rm -rf $(BUILD_DIR)
	rm -rf $(CACHE_DIR)
	rm -rf logs
	rm -rf coverage
	@echo "清理完成"

# 格式化代码
fmt:
	@echo "格式化代码..."
	go fmt ./...

# 代码检查
lint:
	@echo "运行代码检查..."
	golangci-lint run

# 交叉编译
build-all:
	@echo "交叉编译所有平台..."
	@mkdir -p $(BUILD_DIR)
	@mkdir -p $(GO_BUILD_CACHE)
	GOCACHE=$(GO_BUILD_CACHE) GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 cmd/logcmd/main.go
	GOCACHE=$(GO_BUILD_CACHE) GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 cmd/logcmd/main.go
	GOCACHE=$(GO_BUILD_CACHE) GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 cmd/logcmd/main.go
	GOCACHE=$(GO_BUILD_CACHE) GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe cmd/logcmd/main.go
	@echo "交叉编译完成"

# 运行示例
example:
	@echo "运行示例..."
	go run cmd/logcmd/main.go echo "Hello, LogCmd!"
	@echo ""
	@echo "查看日志文件:"
	@ls -lh logs/

# 显示帮助
help:
	@echo "可用的 make 目标:"
	@echo ""
	@echo "构建相关:"
	@echo "  make build          - 编译程序"
	@echo "  make install        - 安装到系统"
	@echo "  make build-all      - 交叉编译所有平台"
	@echo ""
	@echo "单元测试:"
	@echo "  make test           - 运行所有单元测试"
	@echo "  make test-coverage  - 运行测试并生成覆盖率报告"
	@echo "  make test-quick     - 快速测试（无详细输出）"
	@echo "  make test-config    - 测试 config 模块"
	@echo "  make test-executor  - 测试 executor 模块"
	@echo "  make test-model     - 测试 model 模块"
	@echo "  make test-registry  - 测试 registry 模块"
	@echo "  make test-history   - 测试 history 模块"
	@echo "  make test-template  - 测试 template 模块"
	@echo "  make test-stats     - 测试 stats 模块"
	@echo "  make test-search    - 测试 search 模块"
	@echo ""
	@echo "场景测试（端到端）:"
	@echo "  make test-scenarios            - 运行所有场景测试"
	@echo "  make test-scenarios-basic      - 运行基础场景测试"
	@echo "  make test-scenarios-project    - 运行项目管理场景测试"
	@echo "  make test-scenarios-stats      - 运行统计和搜索场景测试"
	@echo "  make test-scenarios-template   - 运行模板配置场景测试"
	@echo "  make test-scenarios-long-log   - 运行大文件输出性能测试"
	@echo "  make test-all                  - 运行所有测试（单元+场景）"
	@echo ""
	@echo "代码质量:"
	@echo "  make fmt            - 格式化代码"
	@echo "  make lint           - 代码检查"
	@echo ""
	@echo "其他:"
	@echo "  make clean          - 清理构建产物"
	@echo "  make example        - 运行示例"
	@echo "  make help           - 显示此帮助信息"
