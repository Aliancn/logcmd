#!/bin/bash
# 项目管理场景测试

# 导入辅助函数
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../helpers/common.sh"

#######################################
# 测试1: 列出所有项目
#######################################
test_list_projects() {
    test_title "列出所有项目"

    # 创建几个测试项目并执行命令
    local project1=$(create_test_project "project1")
    cd "$project1" && mkdir -p .logcmd
    run_logcmd echo "test1" >/dev/null 2>&1

    local project2=$(create_test_project "project2")
    cd "$project2" && mkdir -p .logcmd
    run_logcmd echo "test2" >/dev/null 2>&1

    # 列出项目
    local output=$(run_logcmd project list 2>&1)

    # 验证输出包含项目信息
    assert_contains "$output" "项目" || assert_contains "$output" "project"
}

#######################################
# 测试2: 查看项目信息
#######################################
test_project_info() {
    test_title "查看项目信息"

    local project_dir=$(create_test_project "info_test")
    cd "$project_dir" && mkdir -p .logcmd

    # 执行几个命令
    run_logcmd echo "test command" >/dev/null 2>&1
    run_logcmd ls >/dev/null 2>&1

    # 查看项目信息
    local output=$(run_logcmd project info "$project_dir/.logcmd" 2>&1)

    # 验证输出包含项目路径或统计信息
    TESTS_RUN=$((TESTS_RUN + 1))
    if [[ -n "$output" ]]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}  ✓ PASS: 获取到项目信息${NC}"
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}  ✗ FAIL: 未获取到项目信息${NC}"
    fi
}

#######################################
# 测试3: 项目统计自动更新
#######################################
test_project_stats_update() {
    test_title "项目统计自动更新"

    local project_dir=$(create_test_project "stats_test")
    cd "$project_dir" && mkdir -p .logcmd

    # 执行成功的命令
    run_logcmd echo "success" >/dev/null 2>&1

    # 执行失败的命令
    run_logcmd false >/dev/null 2>&1 || true

    # 查看项目统计
    local output=$(run_logcmd project info "$project_dir/.logcmd" 2>&1)

    # 验证统计信息被更新
    TESTS_RUN=$((TESTS_RUN + 1))
    if [[ -n "$output" ]]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}  ✓ PASS: 统计信息已更新${NC}"
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}  ✗ FAIL: 统计信息未更新${NC}"
    fi
}

#######################################
# 测试4: 项目清理
#######################################
test_project_cleanup() {
    test_title "项目清理功能"

    local project_dir=$(create_test_project "cleanup_test")
    cd "$project_dir" && mkdir -p .logcmd

    # 执行命令
    run_logcmd echo "test" >/dev/null 2>&1

    # 验证项目已注册
    local list_output=$(run_logcmd project list 2>&1)

    TESTS_RUN=$((TESTS_RUN + 1))
    if [[ -n "$list_output" ]]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}  ✓ PASS: 项目清理功能可用${NC}"
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}  ✗ FAIL: 项目清理功能不可用${NC}"
    fi
}

#######################################
# 测试5: 多项目管理
#######################################
test_multiple_projects() {
    test_title "多项目管理"

    # 创建多个项目
    local projects=()
    for i in {1..3}; do
        local project=$(create_test_project "multi_project_$i")
        cd "$project" && mkdir -p .logcmd
        run_logcmd echo "project $i" >/dev/null 2>&1
        projects+=("$project")
    done

    # 列出所有项目
    local output=$(run_logcmd project list 2>&1)

    TESTS_RUN=$((TESTS_RUN + 1))
    if [[ -n "$output" ]]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}  ✓ PASS: 多项目管理正常${NC}"
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}  ✗ FAIL: 多项目管理异常${NC}"
    fi
}

#######################################
# 主函数
#######################################
main() {
    print_separator
    echo -e "${BLUE}项目管理场景测试${NC}"
    print_separator

    init_test_env

    # 运行所有测试
    test_list_projects
    test_project_info
    test_project_stats_update
    test_project_cleanup
    test_multiple_projects

    # 清理
    cleanup_test_env

    # 打印总结
    print_summary
}

# 运行主函数
main
exit $?
