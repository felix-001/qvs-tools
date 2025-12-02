package miku

/*
import (
	//"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mikutool/config"
	"net"
	"net/http"
	"sync"
	"time"

	"proxy/protocol"
)

// Server 代理服务器A
type Server struct {
	// TCP监听端口，用于接收客户端B的连接
	tcpPort int
	// HTTP监听端口，用于接收浏览器请求
	httpPort int
	// 活跃的客户端连接
	clients map[string]*ClientConnection
	// 客户端连接锁
	clientsMutex sync.RWMutex
	// 等待响应的请求
	pendingRequests map[string]*PendingRequest
	// 请求锁
	requestsMutex sync.RWMutex
}

// ClientConnection 客户端连接
type ClientConnection struct {
	ID       string
	Conn     net.Conn
	LastSeen time.Time
}

// PendingRequest 等待响应的请求
type PendingRequest struct {
	Chan     chan *protocol.HTTPResponseData
	Deadline time.Time
}

// NewServer 创建服务器
func NewProxyServer(tcpPort, httpPort int) *Server {
	return &Server{
		tcpPort:         tcpPort,
		httpPort:        httpPort,
		clients:         make(map[string]*ClientConnection),
		pendingRequests: make(map[string]*PendingRequest),
	}
}

// Start 启动服务器
func (s *Server) Start() error {
	// 启动TCP服务器接收客户端连接
	go s.startTCPServer()

	// 启动HTTP服务器接收浏览器请求
	return s.startHTTPServer()
}

// startTCPServer 启动TCP服务器
func (s *Server) startTCPServer() {
	addr := fmt.Sprintf(":%d", s.tcpPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to start TCP server on port %d: %v", s.tcpPort, err)
	}
	defer listener.Close()

	log.Printf("TCP server listening on port %d", s.tcpPort)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		go s.handleTCPConnection(conn)
	}
}

// handleTCPConnection 处理TCP连接
func (s *Server) handleTCPConnection(conn net.Conn) {
	defer conn.Close()

	clientID := fmt.Sprintf("client_%d", time.Now().UnixNano())
	client := &ClientConnection{
		ID:       clientID,
		Conn:     conn,
		LastSeen: time.Now(),
	}

	// 添加到客户端列表
	s.clientsMutex.Lock()
	s.clients[clientID] = client
	s.clientsMutex.Unlock()

	log.Printf("Client connected: %s from %s", clientID, conn.RemoteAddr())

	// 处理连接
	buffer := make([]byte, 4096)
	for {
		conn.SetReadDeadline(time.Now().Add(300 * time.Second)) // 5分钟超时

		n, err := conn.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading from client %s: %v", clientID, err)
			}
			break
		}

		// 处理消息
		if err := s.handleMessage(clientID, buffer[:n]); err != nil {
			log.Printf("Error handling message from client %s: %v", clientID, err)
			break
		}

		client.LastSeen = time.Now()
	}

	// 移除客户端
	s.clientsMutex.Lock()
	delete(s.clients, clientID)
	s.clientsMutex.Unlock()

	log.Printf("Client disconnected: %s", clientID)
}

// handleMessage 处理消息
func (s *Server) handleMessage(clientID string, data []byte) error {
	msg, err := protocol.DecodeMessage(data)
	if err != nil {
		return fmt.Errorf("decode message error: %w", err)
	}

	switch msg.Type {
	case protocol.HTTPResponse:
		// 处理HTTP响应
		var responseData protocol.HTTPResponseData
		if err := json.Unmarshal(msg.Data, &responseData); err != nil {
			return fmt.Errorf("unmarshal response data error: %w", err)
		}

		s.requestsMutex.Lock()
		if req, exists := s.pendingRequests[msg.MessageID]; exists {
			select {
			case req.Chan <- &responseData:
			default:
			}
			delete(s.pendingRequests, msg.MessageID)
		}
		s.requestsMutex.Unlock()

	case protocol.Heartbeat:
		// 心跳响应
		log.Printf("Heartbeat received from client %s", clientID)

	default:
		log.Printf("Unknown message type: %d from client %s", msg.Type, clientID)
	}

	return nil
}

// startHTTPServer 启动HTTP服务器
func (s *Server) startHTTPServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleHTTPRequest)

	addr := fmt.Sprintf(":%d", s.httpPort)
	log.Printf("HTTP server listening on port %d", s.httpPort)

	return http.ListenAndServe(addr, mux)
}

// handleHTTPRequest 处理HTTP请求
func (s *Server) handleHTTPRequest(w http.ResponseWriter, r *http.Request) {
	// 检查是否有可用的客户端
	s.clientsMutex.RLock()
	if len(s.clients) == 0 {
		s.clientsMutex.RUnlock()
		http.Error(w, "No available clients", http.StatusServiceUnavailable)
		return
	}

	// 获取第一个可用客户端
	var client *ClientConnection
	for _, c := range s.clients {
		client = c
		break
	}
	s.clientsMutex.RUnlock()

	if client == nil {
		http.Error(w, "No available clients", http.StatusServiceUnavailable)
		return
	}

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 构建HTTP请求数据
	requestData := protocol.HTTPRequestData{
		Method:  r.Method,
		Path:    r.URL.Path,
		Headers: make(map[string]string),
		Body:    body,
	}

	// 复制headers
	for key, values := range r.Header {
		if len(values) > 0 {
			requestData.Headers[key] = values[0]
		}
	}

	// 编码请求数据
	requestDataBytes, err := json.Marshal(requestData)
	if err != nil {
		http.Error(w, "Failed to marshal request data", http.StatusInternalServerError)
		return
	}

	// 生成消息ID
	messageID := fmt.Sprintf("req_%d", time.Now().UnixNano())

	// 创建协议消息
	protocolMsg := protocol.ProtocolMessage{
		Type:      protocol.HTTPRequest,
		MessageID: messageID,
		Data:      requestDataBytes,
		Headers:   make(map[string]string),
	}

	// 编码协议消息
	protocolMsgBytes, err := protocol.EncodeMessage(protocolMsg)
	if err != nil {
		http.Error(w, "Failed to encode protocol message", http.StatusInternalServerError)
		return
	}

	// 创建等待响应的channel
	responseChan := make(chan *protocol.HTTPResponseData, 1)
	s.requestsMutex.Lock()
	s.pendingRequests[messageID] = &PendingRequest{
		Chan:     responseChan,
		Deadline: time.Now().Add(60 * time.Second), // 60秒超时
	}
	s.requestsMutex.Unlock()

	// 发送请求到客户端
	if _, err := client.Conn.Write(protocolMsgBytes); err != nil {
		http.Error(w, "Failed to send request to client", http.StatusServiceUnavailable)
		return
	}

	// 等待响应
	select {
	case responseData := <-responseChan:
		// 设置响应头
		for key, value := range responseData.Headers {
			w.Header().Set(key, value)
		}
		w.WriteHeader(responseData.StatusCode)
		w.Write(responseData.Body)

	case <-time.After(60 * time.Second):
		// 超时
		s.requestsMutex.Lock()
		delete(s.pendingRequests, messageID)
		s.requestsMutex.Unlock()
		http.Error(w, "Request timeout", http.StatusGatewayTimeout)
	}
}

// CleanupExpiredRequests 清理过期请求
func (s *Server) CleanupExpiredRequests() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		s.requestsMutex.Lock()
		for id, req := range s.pendingRequests {
			if now.After(req.Deadline) {
				close(req.Chan)
				delete(s.pendingRequests, id)
			}
		}
		s.requestsMutex.Unlock()
	}
}

func Proxy(conf *config.Config) {
	if conf.Port != 0 {
		conf.Port = 58088
	}
	server := NewProxyServer(58089, conf.Port) // TCP监听58089，HTTP监听58088

	// 启动清理协程
	go server.CleanupExpiredRequests()

	log.Println("Starting proxy server A...")
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

*/
