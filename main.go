package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
	"golang.org/x/term"
)

// Message 表示Claude交互消息
type Message struct {
	Type      string    `json:"type"`      // "output", "input", "error"
	Content   string    `json:"content"`   // 消息内容
	Timestamp time.Time `json:"timestamp"` // 时间戳
}

// ClaudeWarp 主要结构体
type ClaudeWarp struct {
	claudeCmd    *exec.Cmd                // Claude子进程
	ptmx         *os.File                 // PTY主端
	messages     []Message                // 消息历史
	clients      map[*websocket.Conn]bool // WebSocket客户端
	clientsMux   sync.RWMutex             // 客户端锁
	messagesMux  sync.RWMutex             // 消息锁
	inputChan    chan string              // Web输入通道
	outputReader *io.PipeReader           // 输出管道读端
	outputWriter *io.PipeWriter           // 输出管道写端
	inputReader  *io.PipeReader           // 输入管道读端
	inputWriter  *io.PipeWriter           // 输入管道写端
	resizeChan   chan os.Signal           // 窗口大小变化通道
	termState    *term.State              // 终端状态
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源，生产环境需要限制
	},
}

func main() {
	var port = flag.Int("port", 8080, "Web监控端口")
	flag.Parse()

	// 显示启动LOGO
	printLogo()

	warp := &ClaudeWarp{
		messages:   make([]Message, 0),
		clients:    make(map[*websocket.Conn]bool),
		inputChan:  make(chan string, 100),
		resizeChan: make(chan os.Signal, 1),
	}

	// 创建管道用于劫持输入输出
	warp.outputReader, warp.outputWriter = io.Pipe()
	warp.inputReader, warp.inputWriter = io.Pipe()

	// 启动Claude子进程
	claudeCmd := "claude"
	if err := warp.startClaude(claudeCmd); err != nil {
		log.Fatalf("启动Claude失败: %v", err)
	}

	// 启动Web服务器
	go warp.startWebServer(*port)

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 等待信号或劫持完成
	go func() {
		<-sigChan
		fmt.Println("\n👋 ClaudeWarp 正在关闭...")
		warp.cleanup()
		os.Exit(0)
	}()

	// 启动输入输出劫持（会阻塞直到PTY关闭）
	warp.hijackIO()

	// 如果hijackIO返回，说明Claude进程结束了
	fmt.Println("Claude进程已结束")
	warp.cleanup()
}

// printLogo 打印启动LOGO
func printLogo() {
	logo := `
╔═══════════════════════════════════════════════════════════════╗
║                                                               ║
║   ██████╗██╗      █████╗ ██╗   ██╗██████╗ ███████╗           ║
║  ██╔════╝██║     ██╔══██╗██║   ██║██╔══██╗██╔════╝           ║
║  ██║     ██║     ███████║██║   ██║██║  ██║█████╗             ║
║  ██║     ██║     ██╔══██║██║   ██║██║  ██║██╔══╝             ║
║  ╚██████╗███████╗██║  ██║╚██████╔╝██████╔╝███████╗           ║
║   ╚═════╝╚══════╝╚═╝  ╚═╝ ╚═════╝ ╚═════╝ ╚══════╝           ║
║                                                               ║
║                        W A R P                               ║
║                                                               ║
║              🚀 Session Hijacker & Monitor                    ║
║                                                               ║
╚═══════════════════════════════════════════════════════════════╝

🔍 正在启动Claude会话劫持器...
📡 Web监控界面将在 http://localhost:8080 启动
💡 控制台保持Claude原始体验，Web界面提供实时监控

`
	fmt.Print(logo)

	// 显示代理设置信息
	if httpProxy := os.Getenv("HTTP_PROXY"); httpProxy != "" {
		fmt.Printf("🌐 检测到HTTP代理: %s\n", httpProxy)
	}
	if httpsProxy := os.Getenv("HTTPS_PROXY"); httpsProxy != "" {
		fmt.Printf("🔐 检测到HTTPS代理: %s\n", httpsProxy)
	}
	if httpProxy := os.Getenv("http_proxy"); httpProxy != "" {
		fmt.Printf("🌐 检测到http代理: %s\n", httpProxy)
	}
	if httpsProxy := os.Getenv("https_proxy"); httpsProxy != "" {
		fmt.Printf("🔐 检测到https代理: %s\n", httpsProxy)
	}
	if allProxy := os.Getenv("all_proxy"); allProxy != "" {
		fmt.Printf("🔄 检测到all代理: %s\n", allProxy)
	}
	if noProxy := os.Getenv("no_proxy"); noProxy != "" {
		fmt.Printf("🚫 检测到no代理: %s\n", noProxy)
	}
	fmt.Println()
}

// startClaude 启动Claude子进程并设置PTY劫持
func (w *ClaudeWarp) startClaude(cmdStr string) error {
	// 创建Claude命令
	w.claudeCmd = exec.Command("sh", "-c", cmdStr)

	// 继承当前进程的所有环境变量（包括代理设置）
	w.claudeCmd.Env = os.Environ()

	// 调试：显示传递给Claude的关键环境变量
	for _, env := range w.claudeCmd.Env {
		if strings.Contains(strings.ToLower(env), "proxy") {
			w.addMessage("output", fmt.Sprintf("🔧 传递环境变量: %s", env))
		}
	}

	// 启动带PTY的命令
	var err error
	w.ptmx, err = pty.Start(w.claudeCmd)
	if err != nil {
		return fmt.Errorf("启动PTY失败: %v", err)
	}

	// 设置PTY窗口大小以匹配当前终端
	w.setupPTYSize()

	// 监听窗口大小变化
	w.handleWindowResize()

	w.addMessage("output", "🚀 Claude会话已启动")
	w.addMessage("output", "💡 劫持模式：控制台正常显示，此处监控交互")

	return nil
}

// setupPTYSize 设置PTY窗口大小
func (w *ClaudeWarp) setupPTYSize() {
	// 继承当前终端的窗口大小
	if err := pty.InheritSize(os.Stdin, w.ptmx); err != nil {
		// 如果无法继承，设置一个默认大小
		w.addMessage("error", fmt.Sprintf("无法继承终端大小: %v", err))
	}
}

// handleWindowResize 处理窗口大小变化
func (w *ClaudeWarp) handleWindowResize() {
	// 监听窗口大小变化信号
	signal.Notify(w.resizeChan, syscall.SIGWINCH)

	go func() {
		for range w.resizeChan {
			if err := pty.InheritSize(os.Stdin, w.ptmx); err != nil {
				w.addMessage("error", fmt.Sprintf("调整窗口大小失败: %v", err))
			}
		}
	}()

	// 发送初始窗口大小信号
	w.resizeChan <- syscall.SIGWINCH
}

// hijackIO 劫持Claude的输入输出
func (w *ClaudeWarp) hijackIO() {
	// 设置终端原始模式 - 这是关键！
	var err error
	w.termState, err = term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		w.addMessage("error", fmt.Sprintf("设置终端原始模式失败: %v", err))
		return
	}

	// 输入代理：stdin -> PTY (完全透明) - 必须先启动
	go func() {
		buffer := make([]byte, 1)
		for {
			n, err := os.Stdin.Read(buffer)
			if err != nil {
				break
			}

			// 检查是否是Ctrl+C (ASCII 3)
			if n == 1 && buffer[0] == 3 {
				fmt.Println("\n👋 ClaudeWarp 正在关闭...")
				w.cleanup()
				os.Exit(0)
			}

			// 正常转发给PTY
			w.ptmx.Write(buffer[:n])
		}
	}()

	// Web输入处理（独立通道）
	go func() {
		for input := range w.inputChan {
			if _, err := w.ptmx.Write([]byte(input + "\n")); err != nil {
				w.addMessage("error", fmt.Sprintf("发送Web输入失败: %v", err))
				continue
			}
			w.addMessage("input", input+" (Web界面)")
		}
	}()

	// 输出代理：PTY -> stdout + Web (阻塞主线程)
	webWriter := &webWriter{warp: w}
	multiWriter := io.MultiWriter(os.Stdout, webWriter)

	// 这个调用会阻塞，直到PTY关闭
	io.Copy(multiWriter, w.ptmx)
}

// webWriter 实现io.Writer接口，用于Web界面监控
type webWriter struct {
	warp *ClaudeWarp
}

func (w *webWriter) Write(p []byte) (n int, err error) {
	// 发送原始终端数据到Web界面（包含ANSI转义序列）
	if len(p) > 0 {
		content := string(p)
		w.warp.sendTerminalData(content)
	}
	return len(p), nil
}

// sendTerminalData 发送原始终端数据到Web界面
func (w *ClaudeWarp) sendTerminalData(content string) {
	w.clientsMux.RLock()
	defer w.clientsMux.RUnlock()

	// 发送原始终端数据（包含ANSI转义序列）
	data, _ := json.Marshal(map[string]interface{}{
		"type":    "terminal_data",
		"content": content,
	})

	for client := range w.clients {
		if err := client.WriteMessage(websocket.TextMessage, data); err != nil {
			client.Close()
			delete(w.clients, client)
		}
	}
}

// addMessage 添加消息并广播给所有客户端
func (w *ClaudeWarp) addMessage(msgType, content string) {
	msg := Message{
		Type:      msgType,
		Content:   content,
		Timestamp: time.Now(),
	}

	w.messagesMux.Lock()
	w.messages = append(w.messages, msg)
	w.messagesMux.Unlock()

	// 广播给所有WebSocket客户端
	w.broadcastMessage(msg)
}

// broadcastMessage 广播消息给所有客户端
func (w *ClaudeWarp) broadcastMessage(msg Message) {
	w.clientsMux.RLock()
	defer w.clientsMux.RUnlock()

	data, _ := json.Marshal(msg)
	for client := range w.clients {
		if err := client.WriteMessage(websocket.TextMessage, data); err != nil {
			client.Close()
			delete(w.clients, client)
		}
	}
}

// startWebServer 启动Web服务器
func (w *ClaudeWarp) startWebServer(port int) {
	http.HandleFunc("/", w.handleIndex)
	http.HandleFunc("/ws", w.handleWebSocket)
	http.HandleFunc("/api/messages", w.handleMessages)
	http.HandleFunc("/api/input", w.handleInputAPI)

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("📱 Web监控界面: http://localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

// handleIndex 处理主页
func (w *ClaudeWarp) handleIndex(wr http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>ClaudeWarp - Terminal Hijacker</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/xterm@5.3.0/css/xterm.min.css" />
    <style>
        body {
            font-family: 'Menlo', 'Courier New', monospace;
            margin: 0;
            padding: 20px;
            background-color: #1e1e1e;
            color: #d4d4d4;
        }
        .container {
            max-width: 1400px;
            margin: 0 auto;
        }
        .header {
            text-align: center;
            margin-bottom: 20px;
        }
        .info-box {
            background-color: #2d2d30;
            border: 1px solid #3e3e42;
            border-radius: 5px;
            padding: 15px;
            margin-bottom: 20px;
            border-left: 4px solid #0e639c;
        }
        #terminal-container {
            width: 100%;
            height: 65vh;
            padding: 10px;
            box-sizing: border-box;
            background-color: #0c0c0c;
            border: 1px solid #333;
            border-radius: 5px;
        }
        #terminal {
            width: 100%;
            height: 100%;
        }
        .input-section {
            display: flex;
            gap: 10px;
            margin-top: 20px;
        }
        .input-box {
            flex: 1;
            padding: 10px;
            background-color: #3c3c3c;
            border: 1px solid #555;
            border-radius: 3px;
            color: #d4d4d4;
            font-family: inherit;
        }
        .send-btn {
            padding: 10px 20px;
            background-color: #0e639c;
            color: white;
            border: none;
            border-radius: 3px;
            cursor: pointer;
        }
        .send-btn:hover {
            background-color: #1177bb;
        }
        .status {
            text-align: center;
            margin-bottom: 10px;
            font-weight: bold;
        }
        .connected {
            color: #16825d;
        }
        .disconnected {
            color: #f14949;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔍 ClaudeWarp Terminal Hijacker</h1>
            <div id="status" class="status disconnected">● 连接中...</div>
        </div>
        
        <div class="info-box">
            <strong>💡 终端劫持模式:</strong> 完全同步真实终端输出，支持所有ANSI转义序列和颜色
        </div>
        
        <div id="terminal-container">
            <div id="terminal"></div>
        </div>
        
        <div class="input-section">
            <input type="text" id="inputBox" class="input-box" placeholder="远程输入到Claude..." />
            <button id="sendBtn" class="send-btn">发送</button>
        </div>
    </div>

    <script src="https://cdn.jsdelivr.net/npm/xterm@5.3.0/lib/xterm.min.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/xterm-addon-fit@0.8.0/lib/xterm-addon-fit.min.js"></script>
    <script>
        const terminalContainer = document.getElementById('terminal-container');
        const terminalDiv = document.getElementById('terminal');
        const inputBox = document.getElementById('inputBox');
        const sendBtn = document.getElementById('sendBtn');
        const statusDiv = document.getElementById('status');
        
        const term = new Terminal({
            cursorBlink: true,
            fontSize: 14,
            fontFamily: 'Menlo, "DejaVu Sans Mono", Consolas, "Lucida Console", monospace',
            theme: {
                background: '#0c0c0c',
                foreground: '#d4d4d4',
                cursor: '#d4d4d4',
            },
            rows: 30, // Default, will be adjusted by fit addon
        });
        
        const fitAddon = new FitAddon.FitAddon();
        term.loadAddon(fitAddon);
        term.open(terminalDiv);
        
        function fitTerminal() {
            try {
                fitAddon.fit();
            } catch (e) {
                console.error("Fit addon error:", e);
            }
        }
        
        // Fit terminal on load and on window resize
        window.addEventListener('load', fitTerminal);
        window.addEventListener('resize', fitTerminal);
        
        let ws;
        
        function connect() {
            ws = new WebSocket('ws://' + window.location.host + '/ws');
            
            ws.onopen = function() {
                statusDiv.textContent = '● 终端劫持已连接';
                statusDiv.className = 'status connected';
                fitTerminal(); // Fit again on connect
            };
            
            ws.onmessage = function(event) {
                const data = JSON.parse(event.data);
                if (data.type === 'terminal_data' && typeof data.content === 'string') {
                    term.write(data.content);
                }
            };
            
            ws.onclose = function() {
                statusDiv.textContent = '● 终端劫持连接断开';
                statusDiv.className = 'status disconnected';
                setTimeout(connect, 3000);
            };
            
            ws.onerror = function(error) {
                console.error('WebSocket Error: ', error);
                statusDiv.textContent = '● 终端劫持连接错误';
                statusDiv.className = 'status disconnected';
            };
        }
        
        function sendInput() {
            const input = inputBox.value.trim();
            if (input && ws && ws.readyState === WebSocket.OPEN) {
                fetch('/api/input', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({input: input})
                });
                inputBox.value = '';
            }
        }
        
        sendBtn.addEventListener('click', sendInput);
        inputBox.addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                sendInput();
            }
        });
        
        connect();
    </script>
</body>
</html>`
	wr.Header().Set("Content-Type", "text/html; charset=utf-8")
	wr.Write([]byte(html))
}

// handleWebSocket 处理WebSocket连接
func (w *ClaudeWarp) handleWebSocket(wr http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(wr, r, nil)
	if err != nil {
		log.Printf("WebSocket升级失败: %v", err)
		return
	}
	defer conn.Close()

	w.clientsMux.Lock()
	w.clients[conn] = true
	w.clientsMux.Unlock()

	defer func() {
		w.clientsMux.Lock()
		delete(w.clients, conn)
		w.clientsMux.Unlock()
	}()

	// 保持连接活跃
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// handleMessages 处理消息历史API
func (w *ClaudeWarp) handleMessages(wr http.ResponseWriter, r *http.Request) {
	w.messagesMux.RLock()
	data, _ := json.Marshal(w.messages)
	w.messagesMux.RUnlock()

	wr.Header().Set("Content-Type", "application/json")
	wr.Write(data)
}

// handleInputAPI 处理输入API
func (w *ClaudeWarp) handleInputAPI(wr http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(wr, "仅支持POST方法", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Input string `json:"input"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(wr, "无效的JSON", http.StatusBadRequest)
		return
	}

	// 发送到输入通道
	select {
	case w.inputChan <- req.Input:
		wr.WriteHeader(http.StatusOK)
	default:
		http.Error(wr, "输入队列已满", http.StatusServiceUnavailable)
	}
}

// cleanup 清理资源
func (w *ClaudeWarp) cleanup() {
	// 恢复终端状态 - 非常重要！
	if w.termState != nil {
		if err := term.Restore(int(os.Stdin.Fd()), w.termState); err != nil {
			log.Printf("恢复终端状态失败: %v", err)
		}
		w.termState = nil
	}

	// 停止窗口大小监听
	if w.resizeChan != nil {
		signal.Stop(w.resizeChan)
		close(w.resizeChan)
		w.resizeChan = nil
	}

	// 清理管道
	if w.outputWriter != nil {
		w.outputWriter.Close()
		w.outputWriter = nil
	}
	if w.inputWriter != nil {
		w.inputWriter.Close()
		w.inputWriter = nil
	}

	// 关闭PTY
	if w.ptmx != nil {
		w.ptmx.Close()
		w.ptmx = nil
	}

	// 终止Claude进程
	if w.claudeCmd != nil && w.claudeCmd.Process != nil {
		w.claudeCmd.Process.Kill()
		w.claudeCmd.Process.Wait() // 等待进程真正结束
		w.claudeCmd = nil
	}

	// 关闭通道
	if w.inputChan != nil {
		close(w.inputChan)
		w.inputChan = nil
	}
}
