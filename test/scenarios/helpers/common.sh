#!/bin/bash
# 场景测试通用辅助函数库

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 测试统计
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# 获取项目根目录
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
LOGCMD_BIN="${PROJECT_ROOT}/bin/logcmd"

# 临时测试目录
TEST_TMP_DIR=""

#######################################
# 初始化测试环境
#######################################
init_test_env() {
    # 创建临时测试目录
    TEST_TMP_DIR=$(mktemp -d -t logcmd-test-XXXXXX)
    export HOME="$TEST_TMP_DIR"

    # 确保二进制文件存在
    if [[ ! -f "$LOGCMD_BIN" ]]; then
        echo -e "${RED}错误: logcmd 二进制文件不存在，请先运行 make build${NC}"
        exit 1
    fi

    echo -e "${BLUE}=== 测试环境初始化 ===${NC}"
    echo "项目根目录: $PROJECT_ROOT"
    echo "测试临时目录: $TEST_TMP_DIR"
    echo "logcmd 路径: $LOGCMD_BIN"
    echo ""
}

#######################################
# 清理测试环境
#######################################
cleanup_test_env() {
    if [[ -n "$TEST_TMP_DIR" && -d "$TEST_TMP_DIR" ]]; then
        rm -rf "$TEST_TMP_DIR"
    fi
}

#######################################
# 打印测试标题
# Arguments:
#   $1 - 测试名称
#######################################
test_title() {
    local title="$1"
    echo -e "\n${BLUE}▶ 测试: ${title}${NC}"
}

#######################################
# 断言命令成功
# Arguments:
#   $@ - 要执行的命令
#######################################
assert_success() {
    TESTS_RUN=$((TESTS_RUN + 1))

    if "$@" >/dev/null 2>&1; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}  ✓ PASS${NC}"
        return 0
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}  ✗ FAIL: 命令执行失败${NC}"
        echo -e "${YELLOW}    命令: $*${NC}"
        return 1
    fi
}

#######################################
# 断言命令失败
# Arguments:
#   $@ - 要执行的命令
#######################################
assert_failure() {
    TESTS_RUN=$((TESTS_RUN + 1))

    if ! "$@" >/dev/null 2>&1; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}  ✓ PASS (预期失败)${NC}"
        return 0
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}  ✗ FAIL: 命令应该失败但成功了${NC}"
        echo -e "${YELLOW}    命令: $*${NC}"
        return 1
    fi
}

#######################################
# 断言文件存在
# Arguments:
#   $1 - 文件路径
#######################################
assert_file_exists() {
    local file="$1"
    TESTS_RUN=$((TESTS_RUN + 1))

    if [[ -f "$file" ]]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}  ✓ PASS: 文件存在${NC}"
        return 0
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}  ✗ FAIL: 文件不存在: $file${NC}"
        return 1
    fi
}

#######################################
# 断言目录存在
# Arguments:
#   $1 - 目录路径
#######################################
assert_dir_exists() {
    local dir="$1"
    TESTS_RUN=$((TESTS_RUN + 1))

    if [[ -d "$dir" ]]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}  ✓ PASS: 目录存在${NC}"
        return 0
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}  ✗ FAIL: 目录不存在: $dir${NC}"
        return 1
    fi
}

#######################################
# 断言字符串包含
# Arguments:
#   $1 - 主字符串
#   $2 - 要查找的子字符串
#######################################
assert_contains() {
    local haystack="$1"
    local needle="$2"
    TESTS_RUN=$((TESTS_RUN + 1))

    if [[ "$haystack" == *"$needle"* ]]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}  ✓ PASS: 包含预期内容${NC}"
        return 0
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}  ✗ FAIL: 未找到预期内容${NC}"
        echo -e "${YELLOW}    查找: $needle${NC}"
        return 1
    fi
}

#######################################
# 断言字符串相等
# Arguments:
#   $1 - 实际值
#   $2 - 期望值
#######################################
assert_equals() {
    local actual="$1"
    local expected="$2"
    TESTS_RUN=$((TESTS_RUN + 1))

    if [[ "$actual" == "$expected" ]]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}  ✓ PASS: 值相等${NC}"
        return 0
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}  ✗ FAIL: 值不相等${NC}"
        echo -e "${YELLOW}    期望: $expected${NC}"
        echo -e "${YELLOW}    实际: $actual${NC}"
        return 1
    fi
}

#######################################
# 执行logcmd命令
# Arguments:
#   $@ - logcmd 参数
#######################################
run_logcmd() {
    "$LOGCMD_BIN" "$@"
}

#######################################
# 创建测试项目目录
# Arguments:
#   $1 - 项目名称
# Returns:
#   项目路径
#######################################
create_test_project() {
    local project_name="$1"
    local project_path="$TEST_TMP_DIR/$project_name"
    mkdir -p "$project_path"
    echo "$project_path"
}

#######################################
# 等待文件创建
# Arguments:
#   $1 - 文件路径
#   $2 - 超时时间（秒，默认5）
#######################################
wait_for_file() {
    local file="$1"
    local timeout="${2:-5}"
    local elapsed=0

    while [[ ! -f "$file" && $elapsed -lt $timeout ]]; do
        sleep 0.1
        elapsed=$((elapsed + 1))
    done

    [[ -f "$file" ]]
}

#######################################
# 获取最新的日志文件
# Arguments:
#   $1 - 日志目录
#######################################
get_latest_log() {
    local log_dir="$1"
    find "$log_dir" -name "*.log" -type f -print0 | xargs -0 ls -t 2>/dev/null | head -1
}

#######################################
# 打印测试总结
#######################################
print_summary() {
    echo ""
    echo -e "${BLUE}=====================================${NC}"
    echo -e "${BLUE}测试总结${NC}"
    echo -e "${BLUE}=====================================${NC}"
    echo -e "总测试数: $TESTS_RUN"
    echo -e "${GREEN}通过: $TESTS_PASSED${NC}"

    if [[ $TESTS_FAILED -gt 0 ]]; then
        echo -e "${RED}失败: $TESTS_FAILED${NC}"
        echo ""
        echo -e "${RED}测试失败！${NC}"
        return 1
    else
        echo -e "失败: 0"
        echo ""
        echo -e "${GREEN}所有测试通过！${NC}"
        return 0
    fi
}

#######################################
# 打印分隔线
#######################################
print_separator() {
    echo -e "${BLUE}=====================================${NC}"
}
