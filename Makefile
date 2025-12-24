.PHONY: build install clean test

# 变量定义
BINARY_NAME=logcmd
INSTALL_PATH=$(GOPATH)/bin
BUILD_DIR=./bin

# 默认目标
all: build

# 编译
build:
	@echo "正在编译 $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) cmd/logcmd/main.go
	@echo "编译完成: $(BUILD_DIR)/$(BINARY_NAME)"

# 安装到系统
install: build
	@echo "正在安装到 $(INSTALL_PATH)..."
	go install cmd/logcmd/main.go
	@echo "安装完成！"
	@echo "现在可以直接使用: logcmd <command>"

# 运行测试
test:
	@echo "运行测试..."
	go test -v ./...

# 清理构建产物
clean:
	@echo "清理构建产物..."
	rm -rf $(BUILD_DIR)
	rm -rf logs
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
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 cmd/logcmd/main.go
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 cmd/logcmd/main.go
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 cmd/logcmd/main.go
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe cmd/logcmd/main.go
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
	@echo "  make build      - 编译程序"
	@echo "  make install    - 安装到系统"
	@echo "  make test       - 运行测试"
	@echo "  make clean      - 清理构建产物"
	@echo "  make fmt        - 格式化代码"
	@echo "  make lint       - 代码检查"
	@echo "  make build-all  - 交叉编译所有平台"
	@echo "  make example    - 运行示例"
	@echo "  make help       - 显示此帮助信息"
