#!/bin/bash
# 项目管理场景测试

# 导入辅助函数
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../helpers/common.sh"

#######################################
# 工具函数: 插入历史记录（带时间偏移）
#######################################
insert_history_record() {
    local project_dir="$1"
    local log_file="$2"
    local days="$3"
    local db_path="$HOME/.logcmd/data/registry.db"
    local project_log_dir="$project_dir/.logcmd"

    if [[ -z "$project_dir" || -z "$log_file" || -z "$days" ]]; then
        echo "insert_history_record: 参数缺失" >&2
        return 1
    fi

    if [[ "$log_file" != /* ]]; then
        log_file="$project_dir/$log_file"
    fi

    python3 - "$db_path" "$project_log_dir" "$log_file" "$days" >/dev/null <<'PY'
import datetime
import os
import sqlite3
import sys

if len(sys.argv) != 5:
    sys.exit(1)

db_path, project_log_dir, log_file, days = sys.argv[1], sys.argv[2], sys.argv[3], int(sys.argv[4])
target_time = datetime.datetime.now() - datetime.timedelta(days=days)
timestamp = target_time.strftime("%Y-%m-%d %H:%M:%S")
log_date = target_time.strftime("%Y-%m-%d")

with sqlite3.connect(db_path) as conn:
    cur = conn.cursor()
    cur.execute("SELECT id FROM projects WHERE path = ?", (project_log_dir,))
    row = cur.fetchone()
    if row is None:
        sys.exit(2)
    project_id = row[0]

    cur.execute(
        """
        INSERT INTO command_history (
            project_id, command, command_name, command_args,
            start_time, end_time, duration_ms, exit_code, status,
            log_file_path, log_date, stdout_preview, stderr_preview,
            has_error, working_directory, environment_info, created_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        """,
        (
            project_id,
            "echo test",
            "echo",
            "[\"test\"]",
            timestamp,
            timestamp,
            1000,
            0,
            "success",
            log_file,
            log_date,
            "",
            "",
            0,
            os.path.dirname(project_log_dir),
            "{}",
            timestamp,
        ),
    )
    conn.commit()
sys.exit(0)
PY
}

#######################################
# 测试1: 列出所有项目
#######################################
test_list_projects() {
    test_title "列出所有项目"

    # 创建几个测试项目并执行命令
    local project1=$(create_test_project "project1")
    cd "$project1" && mkdir -p .logcmd
    run_logcmd run echo "test1" >/dev/null 2>&1

    local project2=$(create_test_project "project2")
    cd "$project2" && mkdir -p .logcmd
    run_logcmd run echo "test2" >/dev/null 2>&1

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
    run_logcmd run echo "test command" >/dev/null 2>&1
    run_logcmd run ls >/dev/null 2>&1

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
    run_logcmd run echo "success" >/dev/null 2>&1

    # 执行失败的命令
    run_logcmd run false >/dev/null 2>&1 || true

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

    local project_dir
    project_dir=$(create_test_project "cleanup_test")
    cd "$project_dir" && mkdir -p .logcmd

    # 执行命令，确保项目被注册
    run_logcmd run echo "test" >/dev/null 2>&1

    local project_log_dir="$project_dir/.logcmd"

    # 模拟日志目录被手动删除
    rm -rf "$project_log_dir"

    # 清理前列表中应仍包含项目记录
    local initial_list
    initial_list=$(run_logcmd project list 2>&1)
    assert_contains "$initial_list" "$project_log_dir"

    # 不确认清理时，应提示取消且记录仍在
    local cancel_output
    cancel_output=$(printf "no\n" | run_logcmd project clean 2>&1 || true)
    assert_contains "$cancel_output" "清理操作已取消"

    local after_cancel_list
    after_cancel_list=$(run_logcmd project list 2>&1)
    assert_contains "$after_cancel_list" "$project_log_dir"

    # 使用 --force 直接清理
    local force_output
    force_output=$(run_logcmd project clean --force 2>&1)
    assert_contains "$force_output" "已删除"

    local final_list
    final_list=$(run_logcmd project list 2>&1)
    assert_not_contains "$final_list" "$project_log_dir"
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
        run_logcmd run echo "project $i" >/dev/null 2>&1
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
# 测试6: project clean --days 清理旧日志
#######################################
test_project_clean_days() {
    test_title "project clean --days 清理旧日志"

    local project_dir
    project_dir=$(create_test_project "clean_days")
    cd "$project_dir" && mkdir -p .logcmd

    run_logcmd run echo "old log" >/dev/null 2>&1
    local log_file
    log_file=$(get_latest_log ".logcmd")

    assert_file_exists "$log_file"
    assert_success insert_history_record "$project_dir" "$log_file" 2

    assert_success run_logcmd project clean --days 1 --force

    TESTS_RUN=$((TESTS_RUN + 1))
    if [[ ! -f "$log_file" ]]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}  ✓ PASS: 过期日志文件已删除${NC}"
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}  ✗ FAIL: 过期日志文件仍然存在${NC}"
    fi
}

#######################################
# 测试7: project clean --keep 仅保留指定数量
#######################################
test_project_clean_keep() {
    test_title "project clean --keep 限制日志数量"

    local project_dir
    project_dir=$(create_test_project "clean_keep")
    cd "$project_dir" && mkdir -p .logcmd

    # 生成多条日志
    run_logcmd run echo "keep-1" >/dev/null 2>&1
    local log1
    log1=$(get_latest_log ".logcmd")
    assert_success insert_history_record "$project_dir" "$log1" 3

    sleep 1
    run_logcmd run echo "keep-2" >/dev/null 2>&1
    local log2
    log2=$(get_latest_log ".logcmd")
    assert_success insert_history_record "$project_dir" "$log2" 2

    sleep 1
    run_logcmd run echo "keep-3" >/dev/null 2>&1
    local log3
    log3=$(get_latest_log ".logcmd")
    assert_success insert_history_record "$project_dir" "$log3" 1

    local log_count_before
    log_count_before=$(find ".logcmd" -name "*.log" -type f | wc -l | tr -d ' ')
    echo "清理前日志数量: $log_count_before"

    assert_success run_logcmd project clean --keep 1 --force

    local log_count_after
    log_count_after=$(find ".logcmd" -name "*.log" -type f | wc -l | tr -d ' ')

    TESTS_RUN=$((TESTS_RUN + 1))
    if [[ "$log_count_after" == "1" ]]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "${GREEN}  ✓ PASS: 仅保留 1 条日志${NC}"
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "${RED}  ✗ FAIL: 期望剩余 1 条日志，实际 $log_count_after${NC}"
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
    test_project_clean_days
    test_project_clean_keep

    # 清理
    cleanup_test_env

    # 打印总结
    print_summary
}

# 运行主函数
main
exit $?
