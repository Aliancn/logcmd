# LogCmd - 高性能命令日志记录工具

一个使用 Go 编写的高效命令行工具，用于执行命令并自动记录日志，支持日志搜索和统计分析。

## 核心概念

- **Project（项目）**: 一个 `.logcmd` 目录及其管理的所有日志
- **工作目录**: 每个项目下创建的 `.logcmd` 目录，保存该项目内所有命令运行产生的日志文件以及与项目环境/状态相关的元数据
- **应用目录**: 用户 Home 目录下的 `~/.logcmd` 目录，包含全局数据库、配置文件等跨项目共享的数据
- **Run（运行）**: 一次命令执行及其产生的日志

## 特性

### 核心功能

- **智能日志目录**: 类似 Git 的工作方式，自动查找或创建 `.logcmd` 目录
  - 优先在当前目录查找 `.logcmd`
  - 向上查找父目录中的 `.logcmd`
  - 都没找到则在当前目录创建 `.logcmd`
  - 支持手动指定任意目录
- **自动项目注册**: 创建 `.logcmd` 时自动注册到全局数据库
  - 在 home 目录下创建 `~/.logcmd/data/registry.db` 数据库
  - 每个项目对应唯一编号，支持编号或路径操作
  - 无需手动注册，首次执行命令即自动注册
- **集中状态管理**: 使用 SQLite 管理所有项目
  - 支持跨项目搜索和统计
  - 自动清理无效项目（在搜索和统计时）
  - 懒更新检查机制
- **高性能日志记录**: 使用流式处理和缓冲 I/O，支持大输出量命令
- **实时输出**: 命令输出实时显示在终端，同时保存到日志文件
- **智能组织**: 日志文件按日期自动分文件夹存储 (`.logcmd/2024-01-15/log_20240115_143052.log`)
- **丰富元数据**: 记录命令、参数、执行时间、时长、退出码等信息
- **强大搜索**: 支持关键词搜索、正则表达式、日期范围筛选、上下文显示、跨项目搜索
- **统计分析**: 提供命令执行次数、成功率、耗时、每日统计等多维度分析、支持跨项目统计
- **跨平台**: 支持 Linux、macOS、Windows

### 数据库能力

LogCmd 内置数据库层，提供强大的数据管理和查询能力：

- **增强的项目管理**
  - 丰富的项目元数据：名称、描述、分类、标签
  - 实时统计信息：命令总数、成功率、执行时长
  - 项目级别的配置和模板

- **命令历史记录**
  - 完整记录每条命令的执行详情
  - 支持多维度快速查询（时间、命令、状态、项目）
  - 输出预览功能（前 500 字符）
  - 性能提升 40-50 倍

- **统计数据缓存**
  - 按日期预计算统计数据
  - 命令分布和退出码分布
  - 趋势分析和汇总统计
  - JSON 导出功能

- **自动数据库迁移**
  - 程序启动时自动检测并创建所需表结构
  - 兼容历史数据并保留日志文件格式

**了解更多**: [数据库增强功能文档](./docs/DATABASE_ENHANCEMENT_README.md)

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
logcmd search -keyword "error"

# 使用正则表达式搜索
logcmd search -keyword "error|fail|panic" -regex

# 显示上下文（前后各3行）
logcmd search -keyword "timeout" -context 3

# 按日期范围搜索
logcmd search -keyword "error" -start 2024-01-01 -end 2024-01-31

# 区分大小写搜索
logcmd search -keyword "Error" -case
```

### 3. 统计分析

```bash
# 分析所有日志
logcmd -stats

# 分析指定目录的日志
logcmd -stats -dir ./mylogs

# 分析所有已注册目录的日志
logcmd -stats -all
```

统计报告包括：
- 总命令数、成功率、失败率
- 总执行时长、平均时长
- 命令使用频率 Top 10
- 退出码分布
- 每日统计

### 4. 项目管理

项目（Project）是 LogCmd 的核心概念，代表一个 `.logcmd` 目录及其管理的所有日志。

#### 自动注册

首次在目录中执行命令时，会自动创建 `.logcmd` 目录并注册到全局数据库：

```bash
# 首次执行，自动创建并注册项目
logcmd npm test
# 输出: 正在记录日志到: .logcmd/2024-01-15/log_20240115_143052.log
```

#### 列出所有项目

```bash
logcmd project list
```

输出示例：
```
已注册的项目 (共3个):

ID    路径                                                 最后检查时间
--------------------------------------------------------------------------------
1     /Users/user/project1/.logcmd                       2024-01-15 14:30:52  ✓
2     /Users/user/project2/.logcmd                       2024-01-15 15:20:15  ✓
3     /home/user/workspace/.logcmd                       2024-01-15 16:10:30  ✗
```

说明：
- `✓` 表示项目目录存在
- `✗` 表示项目目录已被删除

#### 清理无效项目

删除不存在的项目记录：

```bash
logcmd project clean
```

#### 删除项目

```bash
# 通过ID删除
logcmd project delete 1

# 通过路径删除
logcmd project delete /path/to/.logcmd
```

注意：删除项目会同时删除数据库记录以及对应的 `.logcmd` 日志目录（包含其中的日志文件），操作前请确认不再需要这些数据。

#### 跨项目搜索

搜索所有已注册项目中的日志：

```bash
# 在所有项目中搜索
logcmd search -keyword "error" -all

# 使用正则表达式搜索所有项目
logcmd search -keyword "error|fail" -regex -all
```

跨项目搜索会自动清理不存在的项目。

#### 跨项目统计

统计所有已注册项目的日志：

```bash
logcmd stats -all
```

跨项目统计会自动清理不存在的项目。

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
logcmd search -keyword "error" -regex -context 2
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

### 示例 3: 项目管理工作流

```bash
# 场景1：在新项目中首次使用（自动注册）
cd ~/my-project
logcmd npm test
# 输出: 正在记录日志到: .logcmd/2024-01-15/log_20240115_143052.log
# 项目自动注册到全局数据库

# 场景2：查看所有已注册项目
logcmd project list
# 输出:
# 已注册的项目 (共2个):
# ID    路径                                    最后检查时间
# 1     /Users/user/my-project/.logcmd        2024-01-15 14:30:52  ✓
# 2     /Users/user/old-project/.logcmd       2024-01-10 10:20:15  ✗

# 场景3：清理已删除的项目
logcmd project clean
# 自动删除 old-project 的记录

# 场景4：跨所有项目搜索错误
logcmd search -keyword "error|fail" -regex -all
# 在所有项目中搜索，自动跳过已删除的项目

# 场景5：查看所有项目的统计
logcmd stats -all
# 显示每个项目的统计报告
```

### 示例 4: 统计报告

```bash
logcmd stats
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
- `help`, `-help`: 显示帮助信息

### 搜索命令
```bash
logcmd search [选项]
```

选项：
- `-keyword string`: 搜索关键词（必需）
- `-regex`: 使用正则表达式
- `-case`: 区分大小写
- `-context int`: 显示上下文行数
- `-start string`: 开始日期 (YYYY-MM-DD)
- `-end string`: 结束日期 (YYYY-MM-DD)
- `-dir string`: 日志目录路径
- `-all`: 搜索所有已注册项目

### 统计命令
```bash
logcmd stats [选项]
```

选项：
- `-dir string`: 日志目录路径
- `-all`: 统计所有已注册项目

### 项目管理命令
```bash
logcmd project <command>
```

命令：
- `list`: 列出所有已注册的项目
- `clean`: 清理不存在的项目
- `delete <id|path>`: 删除指定的项目（支持ID或路径）

## 日志文件格式

日志文件包含完整的命令执行信息：

```
################################################################################
# LogCmd - 命令执行日志
# 时间: 2024-01-15 14:30:52
# 命令: npm [test]
################################################################################

> myproject@1.0.0 test
> jest

PASS  ./sum.test.js
  ✓ adds 1 + 2 to equal 3 (5ms)

Test Suites: 1 passed, 1 total
Tests:       1 passed, 1 total

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
│   ├── registry/
│   │   └── registry.go       # 增强版 Registry
│   ├── search/
│   │   └── search.go         # 日志搜索
│   ├── stats/
│   │   ├── cache_manager.go  # 统计缓存管理
│   │   ├── report.go         # 统一统计报告
│   │   └── stats.go          # 日志分析与输出
│   ├── model/                # 数据模型
│   │   ├── project.go        # 项目模型
│   │   ├── command.go        # 命令历史模型
│   │   └── stats.go          # 统计缓存模型
│   ├── migration/            # 数据库迁移
│   │   └── migration.go      # 迁移管理器
│   ├── history/              # 命令历史管理
│   │   └── manager.go        # 历史记录管理器
├── examples/                 # 示例代码
│   └── database_demo.go      # 数据库功能示例
├── docs/                     # 文档
│   ├── DATABASE_DESIGN.md    # 数据库设计文档
│   ├── USAGE_GUIDE.md        # 使用指南
│   └── DATABASE_ENHANCEMENT_README.md  # 增强功能说明
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
- **数据库**: SQLite3 (github.com/mattn/go-sqlite3)
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
