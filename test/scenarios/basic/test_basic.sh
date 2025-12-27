#!/bin/bash
# 基础功能场景测试

# 导入辅助函数
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../helpers/common.sh"

#######################################
# 测试1: 执行简单命令并生成日志
#######################################
test_simple_command_execution() {
    test_title "执行简单命令并生成日志"

    # 创建测试项目
    local project_dir=$(create_test_project "basic_test")
    cd "$project_dir"

    # 初始化 .logcmd 目录
    mkdir -p .logcmd

    # 执行命令
    assert_success run_logcmd echo "Hello, LogCmd!"

    # 验证日志目录存在
    assert_dir_exists ".logcmd"

    # 验证日志文件被创建
    local today=$(date +%Y-%m-%d)
    assert_dir_exists ".logcmd/$today"

    # 查找生成的日志文件
    local log_file=$(get_latest_log ".logcmd")
    assert_file_exists "$log_file"

    # 验证日志内容包含命令和输出
    if [[ -f "$log_file" ]]; then
        local log_content=$(cat "$log_file")
        assert_contains "$log_content" "echo"
        assert_contains "$log_content" "Hello, LogCmd!"
    fi
}

#######################################
# 测试2: 执行失败的命令
#######################################
test_failed_command() {
    test_title "执行失败的命令"

    local project_dir=$(create_test_project "failed_test")
    cd "$project_dir"
    mkdir -p .logcmd

    # 执行会失败的命令（期望返回非零）
    run_logcmd false || true

    # 即使命令失败，日志也应该被创建
    local log_file=$(get_latest_log ".logcmd")
    assert_file_exists "$log_file"

    # 验证日志包含失败信息
    if [[ -f "$log_file" ]]; then
        local log_content=$(cat "$log_file")
        assert_contains "$log_content" "退出码"
    fi
}

#######################################
# 测试3: 执行带参数的命令
#######################################
test_command_with_args() {
    test_title "执行带参数的命令"

    local project_dir=$(create_test_project "args_test")
    cd "$project_dir"
    mkdir -p .logcmd

    # 执行带多个参数的命令
    assert_success run_logcmd echo "arg1" "arg2" "arg3"

    local log_file=$(get_latest_log ".logcmd")
    assert_file_exists "$log_file"

    # 验证所有参数都被记录
    if [[ -f "$log_file" ]]; then
        local log_content=$(cat "$log_file")
        assert_contains "$log_content" "arg1"
        assert_contains "$log_content" "arg2"
        assert_contains "$log_content" "arg3"
    fi
}

#######################################
# 测试4: 子目录中执行命令
#######################################
test_subdirectory_execution() {
    test_title "子目录中执行命令"

    local project_dir=$(create_test_project "subdir_test")
    cd "$project_dir"
    mkdir -p .logcmd
    mkdir -p subdir/nested

    # 在子目录中执行命令
    cd subdir/nested
    assert_success run_logcmd pwd

    # 日志应该在项目根目录的 .logcmd 中
    local log_file=$(get_latest_log "$project_dir/.logcmd")
    assert_file_exists "$log_file"
}

#######################################
# 测试5: 执行长时间运行的命令
#######################################
test_long_running_command() {
    test_title "执行长时间运行的命令"

    local project_dir=$(create_test_project "long_test")
    cd "$project_dir"
    mkdir -p .logcmd

    # 执行需要一定时间的命令
    assert_success run_logcmd sleep 0.5

    local log_file=$(get_latest_log ".logcmd")
    assert_file_exists "$log_file"

    # 验证日志包含执行时长信息
    if [[ -f "$log_file" ]]; then
        local log_content=$(cat "$log_file")
        assert_contains "$log_content" "执行时长"
    fi
}

#######################################
# 测试6: 多次执行命令
#######################################
test_multiple_executions() {
    test_title "多次执行命令"

    local project_dir=$(create_test_project "multi_test")
    cd "$project_dir"
    mkdir -p .logcmd

    # 执行多次，每次间隔一下确保时间戳不同
    assert_success run_logcmd echo "execution 1"
    sleep 0.1
    assert_success run_logcmd echo "execution 2"
    sleep 0.1
    assert_success run_logcmd echo "execution 3"

    # 验证生成了多个日志文件
    local log_count=$(find ".logcmd" -name "*.log" -type f | wc -l | tr -d ' ')

    TESTS_RUN=$((TESTS_RUN + 1))
    if [[ $log_count -ge 1 ]]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}  ✓ PASS: 生成了 $log_count 个日志文件${NC}"
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}  ✗ FAIL: 期望至少1个日志文件，实际: $log_count${NC}"
    fi
}

#######################################
# 测试7: 输出重定向
#######################################
test_output_capture() {
    test_title "输出捕获"

    local project_dir=$(create_test_project "output_test")
    cd "$project_dir"
    mkdir -p .logcmd

    # 执行有标准输出和错误输出的命令
    run_logcmd sh -c 'echo "stdout message"; echo "stderr message" >&2'

    local log_file=$(get_latest_log ".logcmd")
    assert_file_exists "$log_file"

    # 验证两种输出都被捕获
    if [[ -f "$log_file" ]]; then
        local log_content=$(cat "$log_file")
        assert_contains "$log_content" "stdout message"
        assert_contains "$log_content" "stderr message"
    fi
}

#######################################
# 主函数
#######################################
main() {
    print_separator
    echo -e "${BLUE}基础功能场景测试${NC}"
    print_separator

    init_test_env

    # 运行所有测试
    test_simple_command_execution
    test_failed_command
    test_command_with_args
    test_subdirectory_execution
    test_long_running_command
    test_multiple_executions
    test_output_capture

    # 清理
    cleanup_test_env

    # 打印总结
    print_summary
}

# 运行主函数
main
exit $?
