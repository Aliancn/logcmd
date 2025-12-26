#!/bin/bash
# 测试日志命名模板功能

cd /Users/aliancn/code/logcmd

echo "=========================================="
echo "测试1: 查看当前模板配置"
echo "=========================================="
cat ~/.logcmd_template.json
echo ""

echo "=========================================="
echo "测试2: 使用当前模板执行命令"
echo "=========================================="
./bin/logcmd echo "hello world"
echo ""

echo "=========================================="
echo "测试3: 配置新模板 (project + time)"
echo "=========================================="
cat > ~/.logcmd_template.json <<EOF
{
  "separator": "-",
  "elements": [
    {
      "type": "project",
      "config": {}
    },
    {
      "type": "time",
      "config": {
        "format": "20060102_150405"
      }
    }
  ]
}
EOF

cat ~/.logcmd_template.json
echo ""

echo "=========================================="
echo "测试4: 使用新模板执行命令"
echo "=========================================="
./bin/logcmd npm test 2>&1 || true
echo ""

echo "=========================================="
echo "测试5: 配置完整模板 (project + command + time + custom)"
echo "=========================================="
cat > ~/.logcmd_template.json <<EOF
{
  "separator": "_",
  "elements": [
    {
      "type": "project",
      "config": {}
    },
    {
      "type": "command",
      "config": {}
    },
    {
      "type": "time",
      "config": {
        "format": "0102_1504"
      }
    },
    {
      "type": "custom",
      "config": {
        "text": "日志"
      }
    }
  ]
}
EOF

cat ~/.logcmd_template.json
echo ""

echo "=========================================="
echo "测试6: 使用完整模板执行命令"
echo "=========================================="
./bin/logcmd git status
echo ""

echo "=========================================="
echo "测试7: 查看今天生成的所有日志文件"
echo "=========================================="
ls -lh .logcmd/$(date +%Y-%m-%d)/ | tail -10
echo ""

echo "=========================================="
echo "测试完成！"
echo "=========================================="
