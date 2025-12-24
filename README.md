# LogCmd - 高性能命令日志记录工具

一个使用 Go 编写的高效命令行工具，用于执行命令并自动记录日志，支持日志搜索和统计分析。

## 特性

- **智能日志目录**: 类似 Git 的工作方式，自动查找或创建 `.logcmd` 目录
  - 优先在当前目录查找 `.logcmd`
  - 向上查找父目录中的 `.logcmd`
  - 都没找到则在当前目录创建 `.logcmd`
  - 支持手动指定任意目录
- **高性能日志记录**: 使用流式处理和缓冲 I/O，支持大输出量命令
- **实时输出**: 命令输出实时显示在终端，同时保存到日志文件
- **智能组织**: 日志文件按日期自动分文件夹存储 (`.logcmd/2024-01-15/log_20240115_143052.log`)
- **丰富元数据**: 记录命令、参数、执行时间、时长、退出码等信息
- **强大搜索**: 支持关键词搜索、正则表达式、日期范围筛选、上下文显示
- **统计分析**: 提供命令执行次数、成功率、耗时、每日统计等多维度分析
- **跨平台**: 支持 Linux、macOS、Windows

## 安装

### 方式一：从源码编译

```bash
# 克隆仓库
git clone https://github.com/aliancn/logcmd.git
cd logcmd

# 编译安装
make install

# 或者直接编译
go build -o logcmd cmd/logcmd/main.go
```

### 方式二：使用 go install

```bash
go install github.com/aliancn/logcmd/cmd/logcmd@latest
```

## 快速开始

### 日志目录说明

LogCmd 采用类似 Git 的目录查找机制：

**查找逻辑**：
1. 从当前目录开始，检查是否存在 `.logcmd` 目录
2. 如果没有，向上查找父目录中的 `.logcmd`
3. 如果一直到根目录都没找到，在当前目录创建 `.logcmd`

**使用场景示例**：

```bash
# 场景 1: 项目根目录管理
my-project/
├── .logcmd/           # 在项目根创建
├── src/
│   └── main.go
└── tests/

# 在任何子目录执行命令，都使用项目根的 .logcmd
cd my-project/src && logcmd go build    # → my-project/.logcmd/
cd my-project/tests && logcmd go test   # → my-project/.logcmd/

# 场景 2: 独立目录
cd /tmp
logcmd echo "test"                       # → /tmp/.logcmd/

# 场景 3: 手动指定
logcmd -dir ./custom-logs npm test       # → ./custom-logs/
```

**优势**：
- 项目日志集中管理（在项目根目录）
- 子目录命令自动归档到项目日志
- 避免日志文件散落各处

### 1. 执行命令并记录日志

```bash
# 基本用法（自动查找或创建 .logcmd）
logcmd ls -la

# 在项目根目录初始化（可选）
mkdir .logcmd  # 手动创建，子目录会自动使用

# 指定日志目录
logcmd -dir ./mylogs npm test

# 执行复杂命令
logcmd python train.py --epochs 100
```

日志文件格式：`.logcmd/YYYY-MM-DD/log_YYYYMMDD_HHMMSS.log`

### 2. 搜索日志

```bash
# 搜索包含 "error" 的日志
logcmd -search -keyword "error"

# 使用正则表达式搜索
logcmd -search -keyword "error|fail|panic" -regex

# 显示上下文（前后各3行）
logcmd -search -keyword "timeout" -context 3

# 按日期范围搜索
logcmd -search -keyword "error" -start 2024-01-01 -end 2024-01-31

# 区分大小写搜索
logcmd -search -keyword "Error" -case
```

### 3. 统计分析

```bash
# 分析所有日志
logcmd -stats

# 分析指定目录的日志
logcmd -stats -dir ./mylogs
```

统计报告包括：
- 总命令数、成功率、失败率
- 总执行时长、平均时长
- 命令使用频率 Top 10
- 退出码分布
- 每日统计

## 使用示例

### 示例 1: 记录构建过程

```bash
logcmd make build
```

输出：
```
正在记录日志到: logs/2024-01-15/log_20240115_143052.log
gcc -o myapp main.c
Build successful!
```

### 示例 2: 搜索错误日志

```bash
logcmd -search -keyword "error" -regex -context 2
```

输出：
```
找到 3 条匹配记录:

文件: logs/2024-01-15/log_20240115_143052.log:45
上下文:
  Compiling module A...
  Compiling module B...
  Error: undefined reference to 'foo'
  Build failed
  Exit code: 1

...
```

### 示例 3: 查看统计报告

```bash
logcmd -stats
```

输出：
```
============================================================
日志统计分析报告
============================================================

总命令数: 156
成功: 142 (91.0%)
失败: 14 (9.0%)
总执行时长: 2h15m30s
平均执行时长: 52s

命令使用频率 (Top 10):
----------------------------------------
  1. npm: 45 次
  2. make: 32 次
  3. python: 28 次
  4. go: 21 次
  5. docker: 15 次
  ...

退出码分布:
----------------------------------------
  退出码 0: 142 次
  退出码 1: 12 次
  退出码 2: 2 次

每日统计:
----------------------------------------
  2024-01-15: 45 个命令 (成功: 42, 失败: 3, 总时长: 1h20m15s)
  2024-01-14: 38 个命令 (成功: 35, 失败: 3, 总时长: 55m20s)
  ...
```

## 命令行参数

### 全局选项
- `-dir string`: 日志目录路径（默认：自动查找或创建 `.logcmd`）
- `-version`: 显示版本信息
- `-help`: 显示帮助信息

### 执行模式
- `-run`: 明确指定执行模式（默认行为，可省略）

### 搜索模式
- `-search`: 启用搜索模式
- `-keyword string`: 搜索关键词（必需）
- `-regex`: 使用正则表达式
- `-case`: 区分大小写
- `-context int`: 显示上下文行数
- `-start string`: 开始日期 (YYYY-MM-DD)
- `-end string`: 结束日期 (YYYY-MM-DD)

### 统计模式
- `-stats`: 启用统计分析模式

## 日志文件格式

日志文件包含完整的命令执行信息：

```
################################################################################
# LogCmd - 命令执行日志
# 时间: 2024-01-15 14:30:52
# 命令: npm [test]
################################################################################

[STDOUT] > myproject@1.0.0 test
[STDOUT] > jest
[STDOUT]
[STDOUT] PASS  ./sum.test.js
[STDOUT]   ✓ adds 1 + 2 to equal 3 (5ms)
[STDOUT]
[STDOUT] Test Suites: 1 passed, 1 total
[STDOUT] Tests:       1 passed, 1 total

================================================================================
命令: npm [test]
开始时间: 2024-01-15 14:30:52
结束时间: 2024-01-15 14:30:55
执行时长: 3.2s
退出码: 0
执行状态: 成功
================================================================================
```

## 项目结构

```
logcmd/
├── cmd/
│   └── logcmd/
│       └── main.go           # 主程序入口
├── internal/
│   ├── config/
│   │   └── config.go         # 配置管理
│   ├── executor/
│   │   └── executor.go       # 命令执行器
│   ├── logger/
│   │   └── logger.go         # 日志记录核心
│   ├── search/
│   │   └── search.go         # 日志搜索
│   └── stats/
│       └── stats.go          # 统计分析
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## 性能优化

- **流式处理**: 使用 `bufio.Scanner` 逐行处理输出，内存占用低
- **缓冲 I/O**: 8KB 缓冲区减少磁盘写入次数
- **并发处理**: stdout 和 stderr 并发处理，不阻塞
- **大文件支持**: 支持 1MB 的超长行处理

## 技术栈

- **语言**: Go 1.21+
- **时区**: 默认使用东八区（Asia/Shanghai）
- **并发**: 使用 goroutine 和 WaitGroup
- **I/O**: bufio 缓冲、io.MultiWriter 多路输出

## 开发

```bash
# 运行测试
make test

# 编译
make build

# 安装到 $GOPATH/bin
make install

# 清理
make clean
```

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！

## 作者

aliancn
