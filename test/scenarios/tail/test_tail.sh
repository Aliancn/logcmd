#!/bin/bash
# tail 功能场景测试

set -euo pipefail

# 导入辅助函数
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../helpers/common.sh"

#######################################
# 启动后台任务并写入日志
#######################################
start_detached_tail_task() {
    local project_dir=$(create_test_project "tail_project")
    cd "$project_dir"
    mkdir -p .logcmd

    local output
    output=$(run_logcmd run -d -- python3 -c 'import sys, time; [print(f"Log line {i}") or sys.stdout.flush() or time.sleep(0.05) for i in range(1, 61)]')
    echo "$output" >&2

    local task_id
    task_id=$(echo "$output" | grep -Eo "#([0-9]+)" | tr -d '#')

    TESTS_RUN=$((TESTS_RUN + 1))
    if [[ -n "$task_id" ]]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}  ✓ PASS: 任务 ID $task_id${NC}" >&2
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}  ✗ FAIL: 未获取任务 ID${NC}" >&2
        exit 1
    fi

    echo "$task_id"
}

#######################################
# 测试 tail -n 行为
#######################################
test_tail_static() {
    test_title "tail -n 输出"

    local task_id="$1"
    local output
    output=$(run_logcmd tail -n 10 "$task_id")
    local line_count=$(echo "$output" | wc -l | tr -d ' ')

    TESTS_RUN=$((TESTS_RUN + 1))
    if [[ "$line_count" == "10" ]]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}  ✓ PASS: tail -n 10 输出行数正确${NC}"
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}  ✗ FAIL: 期望 10 行, 实际: $line_count${NC}"
        echo "$output"
        exit 1
    fi

    assert_contains "$output" "Log line"
}

#######################################
# 测试 tail -f 行为
#######################################
test_tail_follow() {
    test_title "tail -f 跟踪"

    local task_id="$1"
    local temp_file="tail_follow.out"

    run_logcmd tail -f "$task_id" >"$temp_file" &
    local tail_pid=$!

    sleep 2
    kill "$tail_pid" 2>/dev/null || true
    wait "$tail_pid" 2>/dev/null || true

    local line_count=$(wc -l <"$temp_file" | tr -d ' ')
    TESTS_RUN=$((TESTS_RUN + 1))
    if [[ "$line_count" -gt 0 ]]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}  ✓ PASS: tail -f 捕获 $line_count 行${NC}"
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}  ✗ FAIL: tail -f 无输出${NC}"
        exit 1
    fi

    assert_contains "$(cat "$temp_file")" "Log line"
    rm -f "$temp_file"
}

#######################################
# 主流程
#######################################
main() {
    print_separator
    echo -e "${BLUE}tail 功能场景测试${NC}"
    print_separator

    init_test_env

    local task_id
    test_title "启动后台任务产生日志"
    task_id=$(start_detached_tail_task)
    sleep 1

    test_tail_static "$task_id"
    test_tail_follow "$task_id"

    cleanup_test_env
    print_summary
}

main "$@"
exit $?
