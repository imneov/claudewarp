#!/bin/bash

# ClaudeWarp 使用示例

echo "=== ClaudeWarp 使用示例 ==="
echo

echo "1. 基本用法 - 启动默认Claude命令："
echo "   go run main.go"
echo

echo "2. 指定端口启动："
echo "   go run main.go -port 9090"
echo

echo "3. 指定自定义Claude命令："
echo "   go run main.go -claude 'claude --model claude-3-sonnet-20240229'"
echo

echo "4. 完整示例："
echo "   go run main.go -port 8080 -claude 'claude --interactive'"
echo

echo "5. 访问Web界面："
echo "   打开浏览器访问: http://localhost:8080"
echo

echo "=== 功能特性 ==="
echo "• 实时捕获Claude所有输入输出"
echo "• Web界面观察session状态"
echo "• 通过页面向Claude发送回复"
echo "• WebSocket实时通信"
echo "• 消息历史记录"
echo

echo "=== 使用流程 ==="
echo "1. 启动程序后，会自动启动Claude子进程"
echo "2. 在浏览器中打开Web界面"
echo "3. 观察Claude的输出和交互"
echo "4. 在需要时通过输入框回复Claude的询问"
echo

echo "注意：确保系统中已安装claude命令行工具"