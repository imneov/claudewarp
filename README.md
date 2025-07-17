# ClaudeWarp

[![Go Version](https://img.shields.io/badge/Go-1.23+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Documentation](https://img.shields.io/badge/docs-claudewarp.dev-blue)](https://claudewarp.dev)

> 🚀 Full-featured Claude bridge
> 
> ClaudeWarp is an intelligent remote control cockpit that connects Claude with chat platforms, providing real-time collaboration monitoring and multi-platform integration capabilities.

## ✨ Core Features

🎛️ **All-in-One Remote Control** - Built-in web interface for seamless Claude interaction  
🤝 **Real-time Collaboration** - WebSocket-powered multi-user monitoring with session history  
🌐 **Multi-platform Bridge** - Native integrations for Telegram, Slack, Discord and more  
📊 **Session History** - Complete interaction recording and structured logging  
🖥️ **PTY Technology** - Full terminal emulation with ANSI escape sequence support  

## 📖 Documentation

Visit our [documentation website](https://claudewarp.dev) for comprehensive guides, API references, and examples.

## 🏗️ Architecture

- **Non-intrusive**: Completely transparent to Claude processes while maintaining original experience
- **Real-time Monitoring**: Web interface provides complete terminal simulation with real-time updates  
- **High Performance**: Efficient I/O hijacking based on PTY technology
- **Cross-platform**: Supports macOS, Linux and other Unix-like systems

## 快速开始

### 安装依赖

```bash
go mod tidy
```

### 启动 ClaudeWarp

```bash
# 默认启动（端口 8080）
go run main.go

# 自定义端口
go run main.go -port 9000

# 构建二进制文件
go build -o claudewarp main.go
./claudewarp -port 8080
```

### 访问 Web 界面

启动后访问 `http://localhost:8080` 查看实时终端监控界面。

## 使用方式

1. **启动 ClaudeWarp**: 运行 `go run main.go`
2. **自动启动 Claude**: 程序会自动启动 Claude 子进程
3. **终端交互**: 在控制台正常使用 Claude，体验完全一致
4. **Web 监控**: 同时在浏览器中实时查看所有交互内容
5. **完整劫持**: 所有输入输出都被无缝劫持和记录

## 技术实现

### 核心组件

- **PTY 管理**: 使用 `github.com/creack/pty` 创建伪终端
- **终端控制**: 使用 `golang.org/x/term` 处理原始模式和终端状态
- **WebSocket 通信**: 使用 `github.com/gorilla/websocket` 实现实时通信
- **I/O 劫持**: 多路复用器同时输出到终端和 Web 界面

### 数据流

```
用户输入 → 原始终端模式 → PTY → Claude 进程
                ↓
Claude 输出 → PTY → 多路输出 → [终端显示 + Web界面]
```

## API 接口

### WebSocket 端点

- `GET /ws` - WebSocket 连接，用于实时数据传输

### 消息格式

```json
{
  "type": "terminal_data",
  "content": "实际终端输出内容（包含ANSI转义序列）"
}
```

## 项目结构

```
claudewarp/
├── main.go           # 主程序入口
├── go.mod           # Go 模块定义
├── go.sum           # 依赖校验
├── CLAUDE.md        # 项目指导文档
└── README.md        # 项目说明
```

## 核心代码结构

### ClaudeWarp 结构体

```go
type ClaudeWarp struct {
    claudeCmd      *exec.Cmd               // Claude子进程
    ptmx           *os.File                // PTY主端
    messages       []Message               // 消息历史
    clients        map[*websocket.Conn]bool // WebSocket客户端
    // ... 其他字段
}
```

### 关键功能

- **PTY 劫持**: `hijackIO()` - 实现完全透明的输入输出劫持
- **Web 服务**: `startWebServer()` - 提供 HTTP 和 WebSocket 服务
- **终端仿真**: Web 端完整的 ANSI 转义序列处理和终端模拟

## 系统要求

- **Go 1.23+**: 需要 Go 1.23 或更高版本
- **Unix-like 系统**: 支持 PTY 的操作系统（macOS、Linux 等）
- **现代浏览器**: 支持 WebSocket 的浏览器

## 依赖项

```go
require (
    github.com/creack/pty v1.1.24          // PTY 管理
    github.com/gorilla/websocket v1.5.3    // WebSocket 通信
    golang.org/x/term v0.33.0              // 终端控制
)
```

## 特性详解

### 终端劫持模式

- **完全透明**: 用户在控制台的体验与直接使用 Claude 完全一致
- **实时同步**: Web 界面实时显示所有终端输出，包括 Unicode 字符和 ANSI 颜色
- **无延迟**: 基于 PTY 的高效实现，几乎无性能损耗

### Web 监控界面

- **终端模拟**: 完整的 ANSI 转义序列支持，准确显示颜色和格式
- **实时更新**: WebSocket 实现毫秒级的实时数据传输
- **Unicode 支持**: 正确处理中文、emoji 和特殊字符显示
- **响应式设计**: 适配不同屏幕尺寸的设备

## 开发与调试

### 代理环境支持

程序会自动检测并显示以下代理环境变量：
- `HTTP_PROXY` / `http_proxy`
- `HTTPS_PROXY` / `https_proxy`
- `all_proxy`
- `no_proxy`

### 信号处理

- **Ctrl+C**: 安全退出，自动清理所有资源
- **窗口大小变化**: 自动同步终端窗口大小到 Claude 进程

## 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

## 贡献

欢迎提交 Issue 和 Pull Request！

## 作者

- **项目维护者**: [imneov](https://github.com/imneov)

---

*ClaudeWarp - 让 Claude 的每一次交互都可见、可控、可回放* 🚀