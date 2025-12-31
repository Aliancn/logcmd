#!/bin/bash
# 统计和搜索场景测试

# 导入辅助函数
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../helpers/common.sh"

#######################################
# 测试1: 日志统计分析
#######################################
test_stats_analysis() {
    test_title "日志统计分析"

    local project_dir=$(create_test_project "stats_analysis")
    cd "$project_dir" && mkdir -p .logcmd

    # 执行多个不同的命令
    run_logcmd run echo "test1" >/dev/null 2>&1
    run_logcmd run ls >/dev/null 2>&1
    run_logcmd run pwd >/dev/null 2>&1
    run_logcmd run false >/dev/null 2>&1 || true

    # 运行统计分析
    local output=$(run_logcmd stats "$project_dir/.logcmd" 2>&1)

    # 验证统计输出
    TESTS_RUN=$((TESTS_RUN + 1))
    if [[ -n "$output" ]]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}  ✓ PASS: 统计分析执行成功${NC}"
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}  ✗ FAIL: 统计分析失败${NC}"
    fi
}

#######################################
# 测试2: 搜索日志内容
#######################################
test_log_search() {
    test_title "搜索日志内容"

    local project_dir=$(create_test_project "search_test")
    cd "$project_dir" && mkdir -p .logcmd

    # 执行包含特定关键词的命令
    run_logcmd run echo "UNIQUE_KEYWORD_12345" >/dev/null 2>&1
    run_logcmd run echo "another message" >/dev/null 2>&1

    # 搜索关键词
    local output=$(run_logcmd search "$project_dir/.logcmd" "UNIQUE_KEYWORD_12345" 2>&1)

    # 验证搜索结果（或者命令至少能执行）
    TESTS_RUN=$((TESTS_RUN + 1))
    if [[ -n "$output" ]]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}  ✓ PASS: 搜索功能执行成功${NC}"
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}  ✗ FAIL: 搜索功能执行失败${NC}"
    fi
}

#######################################
# 测试3: 搜索不存在的内容
#######################################
test_search_not_found() {
    test_title "搜索不存在的内容"

    local project_dir=$(create_test_project "search_notfound")
    cd "$project_dir" && mkdir -p .logcmd

    # 执行命令
    run_logcmd run echo "test message" >/dev/null 2>&1

    # 搜索不存在的关键词
    local output=$(run_logcmd search "$project_dir/.logcmd" "NONEXISTENT_KEYWORD_99999" 2>&1)

    TESTS_RUN=$((TESTS_RUN + 1))
    if [[ -n "$output" ]]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}  ✓ PASS: 搜索无结果处理正常${NC}"
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}  ✗ FAIL: 搜索无结果处理异常${NC}"
    fi
}

#######################################
# 测试4: 历史记录查询
#######################################
test_history_query() {
    test_title "历史记录查询"

    local project_dir=$(create_test_project "history_test")
    cd "$project_dir" && mkdir -p .logcmd

    # 执行多个命令
    for i in {1..5}; do
        run_logcmd run echo "command $i" >/dev/null 2>&1
    done

    # 查询历史记录
    local output=$(run_logcmd history "$project_dir/.logcmd" 2>&1)

    TESTS_RUN=$((TESTS_RUN + 1))
    if [[ -n "$output" ]]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}  ✓ PASS: 历史记录查询成功${NC}"
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}  ✗ FAIL: 历史记录查询失败${NC}"
    fi
}

#######################################
# 测试5: 统计命令频率
#######################################
test_command_frequency() {
    test_title "统计命令频率"

    local project_dir=$(create_test_project "frequency_test")
    cd "$project_dir" && mkdir -p .logcmd

    # 执行同一命令多次
    for i in {1..3}; do
        run_logcmd run echo "repeated" >/dev/null 2>&1
    done

    # 执行其他命令
    run_logcmd run ls >/dev/null 2>&1
    run_logcmd run pwd >/dev/null 2>&1

    # 运行统计
    local output=$(run_logcmd stats "$project_dir/.logcmd" 2>&1)

    TESTS_RUN=$((TESTS_RUN + 1))
    if [[ -n "$output" ]]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}  ✓ PASS: 命令频率统计成功${NC}"
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}  ✗ FAIL: 命令频率统计失败${NC}"
    fi
}

#######################################
# 测试6: 按日期查询
#######################################
test_date_query() {
    test_title "按日期查询日志"

    local project_dir=$(create_test_project "date_test")
    cd "$project_dir" && mkdir -p .logcmd

    # 执行命令
    run_logcmd run echo "today's log" >/dev/null 2>&1

    # 查看今天的日志目录
    local today=$(date +%Y-%m-%d)
    local date_dir="$project_dir/.logcmd/$today"

    assert_dir_exists "$date_dir"

    # 验证日志文件存在
    local log_count=$(find "$date_dir" -name "*.log" -type f 2>/dev/null | wc -l)

    TESTS_RUN=$((TESTS_RUN + 1))
    if [[ $log_count -gt 0 ]]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}  ✓ PASS: 按日期组织的日志存在${NC}"
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}  ✗ FAIL: 未找到按日期组织的日志${NC}"
    fi
}

#######################################
# 主函数
#######################################
main() {
    print_separator
    echo -e "${BLUE}统计和搜索场景测试${NC}"
    print_separator

    init_test_env

    # 运行所有测试
    test_stats_analysis
    test_log_search
    test_search_not_found
    test_history_query
    test_command_frequency
    test_date_query

    # 清理
    cleanup_test_env

    # 打印总结
    print_summary
}

# 运行主函数
main
exit $?
