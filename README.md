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
- **交互式 TUI**: 极简模式系统，类似 vim 的设计理念
  - **SearchMode（默认）**: 实时模糊搜索，支持全项目/单项目范围切换
  - **ProjectMode**: 项目管理，选择项目即可快速切换到项目日志
  - **TaskMode**: 后台任务管理，实时日志预览
  - **StatsMode**: 统计分析，支持全局和项目级统计
  - **CommandMode**: vim 风格命令输入（`:search`, `:quit` 等）
  - **LogViewMode**: 完整日志查看，关键词高亮，自动定位匹配行
  - **快捷键**: `/` 搜索, `Ctrl+P` 项目, `Ctrl+T` 任务, `Ctrl+S` 统计, `Ctrl+L` 命令, `Ctrl+C` 退出
- **智能组织**: 日志文件按日期自动分文件夹存储 (`.logcmd/2024-01-15/log_20240115_143052.log`)
- **丰富元数据**: 记录命令、参数、执行时间、时长、退出码等信息
- **强大搜索**: 支持关键词搜索、正则表达式、日期范围筛选、上下文显示、跨项目搜索
- **统计分析**: 提供命令执行次数、成功率、耗时、每日统计等多维度分析、支持跨项目统计
- **安全增强**: 
  - **执行白名单**: 可配置允许执行的命令列表，防止恶意操作
  - **路径清洗**: 智能处理文件名，防止路径穿越和非法字符
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
- **持久化协调层**
  - `internal/persistence` 复用 Registry 的数据库连接
  - `RunRepository` 负责项目注册、命令历史写入与统计缓存刷新
  - `StatsUpdater` 将 Logger 的统计增量统一路由到 Registry

- **自动数据库迁移**
  - 程序启动时自动检测并创建所需表结构
  - 兼容历史数据并保留日志文件格式

**了解更多**: [数据库架构设计](./docs/DATABASE_ARCHITECTURE.md)

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
cd my-project/src && logcmd run go build    # → my-project/.logcmd/
cd my-project/tests && logcmd run go test   # → my-project/.logcmd/

# 场景 2: 独立目录
cd /tmp
logcmd run echo "test"                   # → /tmp/.logcmd/

# 场景 3: 手动指定
logcmd -dir ./custom-logs run npm test   # → ./custom-logs/
```

**优势**：
- 项目日志集中管理（在项目根目录）
- 子目录命令自动归档到项目日志
- 避免日志文件散落各处

### 1. 执行命令并记录日志

```bash
# 基本用法（自动查找或创建 .logcmd）
logcmd run ls -la

# 在项目根目录初始化（可选）
mkdir .logcmd  # 手动创建，子目录会自动使用

# 指定日志目录
logcmd -dir ./mylogs run npm test

# 执行复杂命令
logcmd run python train.py --epochs 100

# 后台执行命令（任务模式）
logcmd run -d npm start
logcmd task list
logcmd task stop 1

# 启动交互式界面 (TUI) - 模式系统
logcmd ui
# 默认进入 SearchMode
# 快捷键:
#   /      - 搜索模式（默认）
#   Ctrl+P - 项目模式
#   Ctrl+T - 任务模式
#   Ctrl+S - 统计模式
#   Ctrl+L - 命令模式（如 :search error, :quit）
#   Ctrl+C - 退出
#   Enter  - 打开选中的日志文件（在搜索结果中）
#   Ctrl+A - 切换搜索范围（全部项目/单项目）
#   Esc    - 清空搜索或返回
```

日志文件格式：`.logcmd/YYYY-MM-DD/log_YYYYMMDD_HHMMSS.log`

### 2. 搜索日志（SearchMode）

`logcmd ui` 默认进入搜索模式。所有项目的日志会被增量加载到内存，输入即搜，适合快速查找最近的命令输出。

- 直接输入关键词即可实时过滤，支持中文/英文混合模糊匹配。
- `Enter` 打开选中的日志文件，`Ctrl+A` 切换搜索范围（当前项目/全局）。
- `Esc` 清空搜索，`Ctrl+P`/`Ctrl+T`/`Ctrl+S` 随时切换其他模式。
- 需要命令式操作时按 `Ctrl+L` 进入 CommandMode，可执行 `:search <keyword>`、`:set case on/off`、`:set context <n>` 等指令，保持与旧版 CLI 搜索一致的体验。

### 3. 统计分析（StatsMode）

统计面板也通过 `logcmd ui` 提供，按 `Ctrl+S` 即可切换到 StatsMode：

- 默认展示全局统计（命令总数、成功率、耗时趋势、Top 命令等），按 `a` 键在全局与当前项目之间切换。
- `r` 手动刷新，或在项目模式中选择项目后再次进入 StatsMode 以查看该项目详情。
- 所有统计数据仍由 `internal/stats` 服务驱动，只是入口统一到了 UI，不再需要单独的 CLI 子命令。

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
logcmd run npm test
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

执行时会显示需要删除的项目清单并要求确认，可通过 `--force` 跳过确认直接删除。

#### 清理旧日志和历史记录

为避免全局日志无限增长，可以在 `project clean` 中指定条件：

```bash
# 删除 30 天前的记录
logcmd project clean --days 30

# 仅保留最近 500 条记录，并跳过确认
logcmd project clean --keep 500 --force
```

说明：
- 未提供 `--days`/`--keep` 时，`project clean` 会像以往一样仅清理不存在的项目。
- `--days` 与 `--keep` 可组合使用，执行前会提示确认，可通过 `--force` 跳过（同样适用于无效项目清理）。

#### 删除项目

```bash
# 通过ID删除
logcmd project delete 1

# 通过路径删除
logcmd project delete /path/to/.logcmd
```

注意：删除项目会同时删除数据库记录以及对应的 `.logcmd` 日志目录（包含其中的日志文件），操作前请确认不再需要这些数据。

### 5. 后台任务管理

适合需要长时间运行且无需实时查看输出的命令。

```bash
# 以后台任务方式执行命令
logcmd run -d npm start

# 查看所有运行中的任务
logcmd task list

# 优雅停止任务
logcmd task stop 3

# 强制终止任务
logcmd task kill 3
```

说明：
- `logcmd run -d` 会立即返回，并在后台继续写入日志
- 任务信息、PID 和日志路径会记录在 `logcmd task list` 中
- `stop` 尝试发送 `INT` 信号，`kill` 直接结束进程
- 任务结束后可以在历史记录和统计中查看执行结果

#### 跨项目搜索

搜索所有已注册项目中的日志：

在 SearchMode 中按 `Ctrl+A` 可切换“当前项目 / 全部项目”两种范围。切换为全局时，LogCmd 会自动移除已失效的项目路径，并在所有注册项目中搜索匹配内容。

#### 跨项目统计

进入 StatsMode (`Ctrl+S`) 后，按 `a` 键即可在“全局统计”和“当前项目统计”之间切换。切换为全局后会自动跳过已删除的项目目录，保持数据库干净。

## 配置指南

LogCmd 支持通过 `~/.logcmd/config.json` 进行全局配置。

### 示例配置

```json
{
  "buffer_size": 8192,
  "auto_compress": false,
  "time_format": "20060102_150405",
  "flush_interval_ms": 200,
  "whitelist": [
    "npm",
    "make",
    "go",
    "python",
    "docker"
  ],
  "max_retention_days": 30,
  "max_retention_count": 1000
}
```

### 配置项说明

- **buffer_size**: 日志写入缓冲区大小（字节），默认 8192。
- **flush_interval_ms**: 日志刷新间隔（毫秒），默认 200ms。
- **whitelist**: 命令执行白名单。
- **max_retention_days**: 自动清理超过指定天数的日志（0 表示不限制）。
- **max_retention_count**: 自动保留最近 N 条记录（0 表示不限制）。
- **auto_compress**: 是否自动压缩旧日志（暂未实现）。

## 使用示例

### 示例 1: 记录构建过程

```bash
logcmd run make build
```

输出：
```
正在记录日志到: logs/2024-01-15/log_20240115_143052.log
gcc -o myapp main.c
Build successful!
```

### 示例 2: 搜索错误日志

1. 执行 `logcmd ui`，保持在默认的 SearchMode。
2. 输入 `error`（或任意关键字），搜索结果列表即时刷新。
3. 选中目标记录并按 `Enter` 打开完整日志。

示意输出：
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
logcmd run npm test
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
logcmd ui
# 默认进入 SearchMode，按 Ctrl+A 切换为“全部项目”，输入 error|fail 即可实时查看命中结果

# 场景5：查看所有项目的统计
# 在 UI 中按 Ctrl+S 切换到 StatsMode，再按 a 查看全局统计面板
```

### 示例 4: 统计报告

在 StatsMode 中可以看到类似的统计输出：
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

### 执行命令
```bash
logcmd run [选项] <command> [args...]
```

选项：
- `-d`: 以后台任务方式运行命令 (`task` 子命令可管理)

### 项目管理命令
```bash
logcmd project <command>
```

命令：
- `list`: 列出所有已注册的项目
- `clean`: 默认清理不存在的项目，附加 `--days` / `--keep` 时用于清理旧日志与历史记录
- `delete <id|path>`: 删除指定的项目（支持ID或路径）

### 任务管理命令
```bash
logcmd task <command>
```

命令：
- `list`: 查看正在运行的后台任务
- `stop <id>`: 发送中断信号，尝试优雅停止任务
- `kill <id>`: 强行终止任务
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

## 技术栈

- **语言**: Go 1.21+
- **数据库**: SQLite3 (github.com/mattn/go-sqlite3)
- **时区**: 默认跟随系统本地时区（`time.Local`）
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
