# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ClaudeWarp is a Go-based bridge that connects Claude Code environment to chat platforms (Telegram, Slack, Discord). It enables remote control of Claude through chat interfaces, streaming outputs back to chat apps in real-time.

## Architecture

The project is currently in early development with a minimal structure:

- **Core Language**: Go
- **Main Entry Point**: `example_bash.go` - contains a basic PTY (pseudo-terminal) example using bash execution
- **Communication Model**: PTY-based command execution with stdin/stdout redirection
- **Target Platforms**: Chat adapters for Telegram, Slack, Discord

## Key Dependencies

Based on `example_bash.go`:
- `github.com/creack/pty` - PTY creation and management
- `golang.org/x/term` - Terminal control and raw mode handling

## Development Commands

Build and run the main application:
```bash
go mod tidy
go run main.go
```

Run with custom options:
```bash
go run main.go -port 8080 -claude "claude --interactive"
```

Access the web interface:
```bash
# Open browser to http://localhost:8080
```

## Architecture Notes

The current example demonstrates the core PTY functionality that will likely be used for:
1. Spawning Claude processes in a controlled terminal environment
2. Capturing input/output streams for chat bridge functionality
3. Handling terminal resize events and signal management

The project aims to create a bridge between chat platforms and Claude, suggesting future architecture will include:
- Chat adapter interfaces (Telegram, Slack, Discord)
- Claude process management and communication
- Bidirectional message routing between chat and Claude
- Session management for multi-turn conversations

## Core Components

- **main.go**: Complete CLI application with web interface
- **PTY Management**: Captures all Claude subprocess I/O
- **Web Server**: Provides real-time session monitoring at http://localhost:8080
- **WebSocket API**: Real-time bidirectional communication
- **Message System**: Structured logging of all Claude interactions

## Features Implemented

- ✅ CLI program that launches Claude subprocess
- ✅ PTY-based I/O capture of all Claude interactions
- ✅ Web interface for real-time session monitoring
- ✅ Input capability from web page to respond to Claude
- ✅ WebSocket real-time communication
- ✅ Message history and structured logging

## API Endpoints

- `GET /` - Web interface
- `GET /ws` - WebSocket connection for real-time updates
- `GET /api/messages` - Get message history
- `POST /api/input` - Send input to Claude

## Usage Example

```bash
go run main.go -port 8080 -claude "claude --interactive"
# Open http://localhost:8080 in browser
```