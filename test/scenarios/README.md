# LogCmd 场景测试文档

## 概述

本目录包含 LogCmd 的端到端场景功能测试，用于测试实际可执行程序的功能。

## 测试架构

```
test/scenarios/
├── helpers/           # 测试辅助函数库
│   └── common.sh     # 通用测试工具和断言函数
├── basic/            # 基础功能测试
│   └── test_basic.sh
├── project/          # 项目管理测试
│   └── test_project.sh
├── stats/            # 统计和搜索测试
│   └── test_stats.sh
├── template/         # 模板配置测试
│   └── test_template.sh
├── tail/             # tail 实时查看测试
│   └── test_tail.sh
└── run_all.sh        # 主测试运行脚本
```

## 快速开始

### 运行所有场景测试

```bash
make test-scenarios
```

或直接运行：

```bash
./test/scenarios/run_all.sh
```

### 运行特定场景测试

```bash
# 基础功能测试
make test-scenarios-basic

# 项目管理测试
make test-scenarios-project

# 统计和搜索测试
make test-scenarios-stats

# 模板配置测试
make test-scenarios-template

# tail 场景测试
make test-scenarios-tail
```

或直接指定：

```bash
./test/scenarios/run_all.sh basic
./test/scenarios/run_all.sh project
./test/scenarios/run_all.sh stats
./test/scenarios/run_all.sh template
./test/scenarios/run_all.sh tail
```

## 测试场景详解

### 1. 基础功能测试 (basic)

测试 LogCmd 的核心命令执行和日志记录功能。

**测试用例：**
- ✅ 执行简单命令并生成日志
- ✅ 执行失败的命令
- ✅ 执行带参数的命令
- ✅ 子目录中执行命令
- ✅ 执行长时间运行的命令
- ✅ 多次执行命令
- ✅ 输出捕获（stdout/stderr）

### 2. 项目管理测试 (project)

测试项目注册、管理和统计功能。

**测试用例：**
- ✅ 列出所有项目
- ✅ 查看项目信息
- ✅ 项目统计自动更新
- ✅ 项目清理功能
- ✅ 多项目管理

### 3. 统计和搜索测试 (stats)

测试日志统计分析和搜索功能。

**测试用例：**
- ✅ 日志统计分析
- ✅ 搜索日志内容
- ✅ 搜索不存在的内容
- ✅ 历史记录查询
- ✅ 统计命令频率
- ✅ 按日期查询日志

### 4. 模板配置测试 (template)

测试日志文件命名模板功能。

**测试用例：**
- ✅ 默认模板功能
- ✅ 查看模板配置
- ✅ 交互式配置模板
- ✅ 配置文件持久化
- ✅ 文件名安全字符处理
- ✅ 项目名称提取

### 5. tail 功能测试 (tail)

验证后台任务日志的查询能力，包括静态查看与实时跟踪。

**测试用例：**
- ✅ tail -n 输出行数校验
- ✅ tail -f 实时跟踪输出

## 测试辅助函数

`helpers/common.sh` 提供了一套完整的测试工具：

### 断言函数

```bash
assert_success <command>          # 断言命令成功
assert_failure <command>          # 断言命令失败
assert_file_exists <file>         # 断言文件存在
assert_dir_exists <dir>           # 断言目录存在
assert_contains <text> <substr>   # 断言包含子字符串
assert_equals <actual> <expected> # 断言相等
```

### 辅助函数

```bash
init_test_env()                   # 初始化测试环境
cleanup_test_env()                # 清理测试环境
create_test_project <name>        # 创建测试项目
run_logcmd <args>                 # 运行 logcmd 命令（执行外部命令请写成 run_logcmd run <command>）
get_latest_log <dir>              # 获取最新日志文件
wait_for_file <file> [timeout]    # 等待文件创建
```

### 输出函数

```bash
test_title <title>                # 打印测试标题
print_summary()                   # 打印测试总结
print_separator()                 # 打印分隔线
```

## 编写新的场景测试

### 1. 创建测试文件

在相应的场景目录下创建测试脚本：

```bash
#!/bin/bash
# 导入辅助函数
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../helpers/common.sh"

#######################################
# 测试1: 测试描述
#######################################
test_my_feature() {
    test_title "测试功能描述"

    # 创建测试项目
    local project_dir=$(create_test_project "my_test")
    cd "$project_dir" && mkdir -p .logcmd

    # 执行测试
    assert_success run_logcmd run echo "test"

    # 验证结果
    local log_file=$(get_latest_log ".logcmd")
    assert_file_exists "$log_file"
}

#######################################
# 主函数
#######################################
main() {
    print_separator
    echo -e "${BLUE}我的测试场景${NC}"
    print_separator

    init_test_env

    # 运行测试
    test_my_feature

    # 清理
    cleanup_test_env

    # 打印总结
    print_summary
}

main
exit $?
```

### 2. 设置可执行权限

```bash
chmod +x test/scenarios/my_scenario/test_my_scenario.sh
```

### 3. 更新主测试脚本

在 `run_all.sh` 中添加新的测试场景。

## 测试环境

- **临时目录**：每次测试运行在独立的临时目录中
- **HOME 环境**：测试期间 HOME 指向临时目录，避免污染用户环境
- **自动清理**：测试结束后自动清理所有临时文件

## 测试结果

测试结果会以彩色输出显示：

- 🟢 **绿色 ✓**：测试通过
- 🔴 **红色 ✗**：测试失败
- 🔵 **蓝色**：信息提示
- 🟡 **黄色**：警告信息

## 持续集成

这些场景测试可以集成到 CI/CD 流程中：

```yaml
# .github/workflows/test.yml
- name: Run scenario tests
  run: |
    make build
    make test-scenarios
```

## 故障排查

### 测试失败

1. 查看详细输出，找到失败的具体测试
2. 单独运行失败的测试场景获取更多信息
3. 检查临时目录中的日志文件

### 调试技巧

```bash
# 运行单个测试并保留输出
./test/scenarios/basic/test_basic.sh

# 查看测试详细日志
make test-scenarios 2>&1 | tee test.log

# 只运行特定测试
./test/scenarios/run_all.sh basic
```

## 最佳实践

1. **独立性**：每个测试应该独立运行，不依赖其他测试
2. **清理**：测试后清理所有临时资源
3. **明确**：使用清晰的测试名称和描述
4. **快速**：避免不必要的延迟，保持测试快速运行
5. **可靠**：测试应该稳定可重复

## 总结

场景测试提供了：

- ✅ 端到端功能验证
- ✅ 真实使用场景覆盖
- ✅ 自动化回归测试
- ✅ 持续集成支持
- ✅ 清晰的测试报告

配合单元测试，共同构建完整的测试体系。
