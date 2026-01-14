# 场景测试快速参考

## 快速命令

```bash
# 运行所有场景测试
make test-scenarios

# 运行特定场景
make test-scenarios-basic      # 基础功能
make test-scenarios-project    # 项目管理
make test-scenarios-template   # 模板配置
make test-scenarios-tail       # tail 功能
make test-scenarios-long-log   # 大日志性能

# 运行所有测试（单元 + 场景）
make test-all
```

## 测试覆盖范围

### 基础功能 (7个测试)
- 命令执行和日志生成
- 成功/失败命令处理
- 参数传递
- 子目录执行
- 长时间命令
- 多次执行
- 输出捕获

### 项目管理 (5个测试)
- 项目列表
- 项目信息
- 统计更新
- 项目清理
- 多项目管理

### 模板配置 (6个测试)
### tail 功能 (2个测试)
- tail -n 输出
- tail -f 跟踪
### 大日志性能 (long_log)
- 超长输出写入
- 大文件读取与 tail

## 测试结果示例

```
╔═══════════════════════════════════════════════════════════╗
║              🎉 所有场景测试通过！ 🎉                    ║
╚═══════════════════════════════════════════════════════════╝

测试套件总数: 5
通过: 5
失败: 0
```

## 常用断言

```bash
assert_success run_logcmd run echo "test"    # 命令成功
assert_file_exists "/path/to/file"           # 文件存在
assert_dir_exists "/path/to/dir"             # 目录存在
assert_contains "$output" "expected text"    # 包含文本
```

## 故障排查

```bash
# 单独运行失败的测试
./test/scenarios/basic/test_basic.sh

# 保存测试日志
make test-scenarios 2>&1 | tee scenario-test.log

# 查看最新日志
ls -lt /tmp/logcmd-test-*/
```
