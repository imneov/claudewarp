---
sidebar_position: 1
---

# ClaudeWarp 介绍

**ClaudeWarp** 是一个智能远程驾驶舱，连接 Claude 与聊天平台的桥梁。它让你可以通过 Web 界面远程控制 Claude，并支持多人协作监控和多平台集成。

## 🌟 核心特性

### 智能远程控制
- 通过现代化 Web 界面远程操作 Claude
- 支持实时交互和命令执行
- 基于 PTY (伪终端) 技术，完整捕获 I/O 流

### 实时协作监控
- WebSocket 实时通信，多人同时监控会话
- 完整的会话历史记录和结构化日志
- 支持从 Web 页面直接输入和响应

### 多平台集成
- 为 Telegram、Slack、Discord 等聊天平台提供桥接
- 一键部署，快速集成到现有工作流
- RESTful API 和 WebSocket 双重支持

## 🚀 快速开始

### 环境要求

- [Go](https://golang.org/) 1.19 或更高版本
- [Claude CLI](https://claude.ai/cli) 已安装并配置

### 安装运行

```bash
# 克隆项目
git clone https://github.com/imneov/claudewarp.git
cd claudewarp

# 构建并运行
go mod tidy
go run main.go -port 8080 -claude "claude --interactive"
```

### 访问界面

打开浏览器访问 [http://localhost:8080](http://localhost:8080)，即可开始使用 ClaudeWarp 的 Web 界面。

## 🎯 使用场景

- **远程演示**：在会议中远程展示 Claude 的使用过程
- **团队协作**：多人同时监控和参与 Claude 会话
- **聊天集成**：将 Claude 无缝集成到团队聊天工具中
- **会话记录**：完整记录和回溯 Claude 交互历史

## 📖 接下来

- 查看 [用户指南](/docs/user-guide) 了解详细使用方法
- 阅读 [架构文档](/docs/architecture) 理解技术实现
- 参考 [开发文档](/docs/development) 贡献代码
