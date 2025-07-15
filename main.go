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
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

// Message 表示Claude交互消息
type Message struct {
	Type      string    `json:"type"`      // "output", "input", "error"
	Content   string    `json:"content"`   // 消息内容
	Timestamp time.Time `json:"timestamp"` // 时间戳
}

// ClaudeWarp 主要结构体
type ClaudeWarp struct {
	claudeCmd    *exec.Cmd               // Claude子进程
	ptmx         *os.File                // PTY主端
	messages     []Message               // 消息历史
	clients      map[*websocket.Conn]bool // WebSocket客户端
	clientsMux   sync.RWMutex            // 客户端锁
	messagesMux  sync.RWMutex            // 消息锁
	inputChan    chan string             // Web输入通道
	outputReader *io.PipeReader          // 输出管道读端
	outputWriter *io.PipeWriter          // 输出管道写端
	inputReader  *io.PipeReader          // 输入管道读端
	inputWriter  *io.PipeWriter          // 输入管道写端
	resizeChan   chan os.Signal          // 窗口大小变化通道
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

	// 启动输入输出劫持
	go warp.hijackIO()

	// 启动Web服务器
	go warp.startWebServer(*port)

	// 等待信号
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("\n👋 ClaudeWarp 正在关闭...")
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
	fmt.Println()
}

// startClaude 启动Claude子进程并设置PTY劫持
func (w *ClaudeWarp) startClaude(cmdStr string) error {
	// 创建Claude命令
	w.claudeCmd = exec.Command("sh", "-c", cmdStr)
	
	// 继承当前进程的所有环境变量（包括代理设置）
	w.claudeCmd.Env = os.Environ()
	
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
	// 最简单的劫持方法：完全透明的双向代理
	
	// 输出代理：PTY -> stdout + Web
	go func() {
		// 使用MultiWriter同时写入stdout和Web监控
		webWriter := &webWriter{warp: w}
		multiWriter := io.MultiWriter(os.Stdout, webWriter)
		
		// 直接复制，完全透明
		io.Copy(multiWriter, w.ptmx)
	}()

	// 输入代理：stdin -> PTY (完全透明)
	go func() {
		// 直接复制stdin到PTY，不做任何干预
		io.Copy(w.ptmx, os.Stdin)
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
}

// webWriter 实现io.Writer接口，用于Web界面监控
type webWriter struct {
	warp *ClaudeWarp
}

func (w *webWriter) Write(p []byte) (n int, err error) {
	// 发送到Web界面
	if len(p) > 0 {
		content := string(p)
		w.warp.addMessage("output", content)
	}
	return len(p), nil
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
    <title>ClaudeWarp - Session Hijacker</title>
    <style>
        body {
            font-family: 'Courier New', monospace;
            margin: 0;
            padding: 20px;
            background-color: #1e1e1e;
            color: #d4d4d4;
        }
        .container {
            max-width: 1200px;
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
        .session-box {
            background-color: #252526;
            border: 1px solid #3e3e42;
            border-radius: 5px;
            height: 500px;
            overflow-y: auto;
            padding: 10px;
            margin-bottom: 20px;
        }
        .message {
            margin-bottom: 10px;
            padding: 5px;
            border-radius: 3px;
        }
        .message.output {
            background-color: #0e639c33;
            border-left: 3px solid #0e639c;
        }
        .message.input {
            background-color: #16825d33;
            border-left: 3px solid #16825d;
        }
        .message.error {
            background-color: #f1494933;
            border-left: 3px solid #f14949;
        }
        .timestamp {
            font-size: 0.8em;
            color: #888;
            margin-right: 10px;
        }
        .input-section {
            display: flex;
            gap: 10px;
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
            <h1>🔍 ClaudeWarp Session Hijacker</h1>
            <div id="status" class="status disconnected">● 连接中...</div>
        </div>
        
        <div class="info-box">
            <strong>💡 劫持模式:</strong> Claude在控制台正常运行，此界面监控所有交互并允许远程输入
        </div>
        
        <div id="messages" class="session-box"></div>
        
        <div class="input-section">
            <input type="text" id="inputBox" class="input-box" placeholder="远程输入到Claude..." />
            <button id="sendBtn" class="send-btn">发送</button>
        </div>
    </div>

    <script>
        const messagesDiv = document.getElementById('messages');
        const inputBox = document.getElementById('inputBox');
        const sendBtn = document.getElementById('sendBtn');
        const statusDiv = document.getElementById('status');
        
        let ws;
        
        function connect() {
            ws = new WebSocket('ws://localhost:' + window.location.port + '/ws');
            
            ws.onopen = function() {
                statusDiv.textContent = '● 劫持已连接';
                statusDiv.className = 'status connected';
                loadHistory();
            };
            
            ws.onmessage = function(event) {
                const message = JSON.parse(event.data);
                addMessage(message);
            };
            
            ws.onclose = function() {
                statusDiv.textContent = '● 劫持连接断开';
                statusDiv.className = 'status disconnected';
                setTimeout(connect, 3000); // 3秒后重连
            };
            
            ws.onerror = function(error) {
                statusDiv.textContent = '● 劫持连接错误';
                statusDiv.className = 'status disconnected';
            };
        }
        
        function addMessage(message) {
            const messageDiv = document.createElement('div');
            messageDiv.className = 'message ' + message.type;
            
            const timestamp = new Date(message.timestamp).toLocaleTimeString();
            messageDiv.innerHTML = '<span class="timestamp">' + timestamp + '</span>' + message.content;
            
            messagesDiv.appendChild(messageDiv);
            messagesDiv.scrollTop = messagesDiv.scrollHeight;
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
        
        function loadHistory() {
            fetch('/api/messages')
                .then(response => response.json())
                .then(messages => {
                    messagesDiv.innerHTML = '';
                    messages.forEach(addMessage);
                });
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
	// 停止窗口大小监听
	if w.resizeChan != nil {
		signal.Stop(w.resizeChan)
		close(w.resizeChan)
	}
	
	// 清理管道
	if w.outputWriter != nil {
		w.outputWriter.Close()
	}
	if w.inputWriter != nil {
		w.inputWriter.Close()
	}
	
	// 关闭PTY
	if w.ptmx != nil {
		w.ptmx.Close()
	}
	
	// 终止Claude进程
	if w.claudeCmd != nil && w.claudeCmd.Process != nil {
		w.claudeCmd.Process.Kill()
	}
	
	// 关闭通道
	close(w.inputChan)
}