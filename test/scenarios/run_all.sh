#!/bin/bash
# LogCmd 场景测试主脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# 测试结果统计
TOTAL_SUITES=0
PASSED_SUITES=0
FAILED_SUITES=0

#######################################
# 获取测试套件描述
# Arguments:
#   $1 - 测试套件名称
#######################################
get_suite_description() {
    case "$1" in
        basic) echo "基础功能测试" ;;
        project) echo "项目管理测试" ;;
        stats) echo "统计和搜索测试" ;;
        template) echo "模板配置测试" ;;
        tail) echo "tail 功能测试" ;;
        long_log) echo "大文件输出性能测试" ;;
        *) echo "未知测试" ;;
    esac
}

#######################################
# 打印横幅
#######################################
print_banner() {
    echo -e "${CYAN}"
    echo "╔═══════════════════════════════════════════════════════════╗"
    echo "║                                                           ║"
    echo "║           LogCmd 场景功能测试套件                        ║"
    echo "║           Scenario Testing Suite                         ║"
    echo "║                                                           ║"
    echo "╚═══════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

#######################################
# 检查环境
#######################################
check_environment() {
    echo -e "${BLUE}检查测试环境...${NC}"

    # 检查是否在项目根目录
    if [[ ! -f "$PROJECT_ROOT/go.mod" ]]; then
        echo -e "${RED}错误: 请在项目根目录运行此脚本${NC}"
        exit 1
    fi

    # 检查二进制文件是否存在
    if [[ ! -f "$PROJECT_ROOT/bin/logcmd" ]]; then
        echo -e "${YELLOW}警告: logcmd 二进制文件不存在，正在编译...${NC}"
        cd "$PROJECT_ROOT"
        make build || {
            echo -e "${RED}错误: 编译失败${NC}"
            exit 1
        }
    fi

    echo -e "${GREEN}✓ 环境检查通过${NC}\n"
}

#######################################
# 运行单个测试套件
# Arguments:
#   $1 - 测试套件名称
#   $2 - 测试套件描述
#######################################
run_test_suite() {
    local suite_name="$1"
    local suite_desc="$2"
    local test_script="$SCRIPT_DIR/$suite_name/test_${suite_name}.sh"

    echo -e "\n${CYAN}╔═══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║ ${suite_desc}${NC}"
    echo -e "${CYAN}╚═══════════════════════════════════════════════════════════╝${NC}"

    TOTAL_SUITES=$((TOTAL_SUITES + 1))

    if [[ ! -f "$test_script" ]]; then
        echo -e "${RED}✗ 测试脚本不存在: $test_script${NC}"
        FAILED_SUITES=$((FAILED_SUITES + 1))
        return 1
    fi

    # 确保脚本可执行
    chmod +x "$test_script"

    # 运行测试脚本
    if bash "$test_script"; then
        PASSED_SUITES=$((PASSED_SUITES + 1))
        echo -e "\n${GREEN}✓ $suite_desc 通过${NC}"
        return 0
    else
        FAILED_SUITES=$((FAILED_SUITES + 1))
        echo -e "\n${RED}✗ $suite_desc 失败${NC}"
        return 1
    fi
}

#######################################
# 打印最终总结
#######################################
print_final_summary() {
    echo -e "\n${CYAN}╔═══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║                   最终测试总结                            ║${NC}"
    echo -e "${CYAN}╚═══════════════════════════════════════════════════════════╝${NC}"

    echo ""
    echo "测试套件总数: $TOTAL_SUITES"
    echo -e "${GREEN}通过: $PASSED_SUITES${NC}"

    if [[ $FAILED_SUITES -gt 0 ]]; then
        echo -e "${RED}失败: $FAILED_SUITES${NC}"
        echo ""
        echo -e "${RED}╔═══════════════════════════════════════════════════════════╗${NC}"
        echo -e "${RED}║              部分测试失败，请检查输出                     ║${NC}"
        echo -e "${RED}╚═══════════════════════════════════════════════════════════╝${NC}"
        return 1
    else
        echo "失败: 0"
        echo ""
        echo -e "${GREEN}╔═══════════════════════════════════════════════════════════╗${NC}"
        echo -e "${GREEN}║              🎉 所有场景测试通过！ 🎉                    ║${NC}"
        echo -e "${GREEN}╚═══════════════════════════════════════════════════════════╝${NC}"
        return 0
    fi
}

#######################################
# 主函数
#######################################
main() {
    # 解析命令行参数
    local specific_suite=""
    if [[ $# -gt 0 ]]; then
        specific_suite="$1"
    fi

    print_banner
    check_environment

    # 如果指定了特定套件，只运行该套件
    if [[ -n "$specific_suite" ]]; then
        local suite_desc=$(get_suite_description "$specific_suite")
        if [[ "$suite_desc" != "未知测试" ]]; then
            run_test_suite "$specific_suite" "$suite_desc"
        else
            echo -e "${RED}错误: 未知的测试套件: $specific_suite${NC}"
            echo -e "${YELLOW}可用的测试套件:${NC}"
            echo "  - basic: 基础功能测试"
            echo "  - project: 项目管理测试"
            echo "  - stats: 统计和搜索测试"
            echo "  - template: 模板配置测试"
            echo "  - tail: tail 功能测试"
            echo "  - long_log: 大文件输出性能测试"
            exit 1
        fi
    else
        # 运行所有测试套件
        for suite in basic project stats template tail long_log; do
            run_test_suite "$suite" "$(get_suite_description $suite)" || true
        done
    fi

    # 打印最终总结
    print_final_summary
}

# 运行主函数
main "$@"
exit $?
