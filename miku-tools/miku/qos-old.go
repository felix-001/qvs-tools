package miku

import (
	"bytes"
	"mikutool/config"
	"mikutool/public/util"

	//"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client 客户端B
type Client struct {
	// 服务器A的地址
	serverAddr string
	// TCP连接
	conn net.Conn
	// 数据库连接
	db *sql.DB
	// 是否已连接
	connected bool
}

// NewClient 创建客户端
func NewClient(serverAddr, dbDSN string) (*Client, error) {
	// 连接数据库
	/*
		db, err := sql.Open("mysql", dbDSN)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}

		// 测试数据库连接
		if err := db.Ping(); err != nil {
			return nil, fmt.Errorf("failed to ping database: %w", err)
		}
	*/

	client := &Client{
		serverAddr: serverAddr,
		//db:         db,
	}

	return client, nil
}

// Start 启动客户端
func (c *Client) Start() error {
	// 连接到服务器
	if err := c.connectToServer(); err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	// 启动心跳
	go c.startHeartbeat()

	// 处理消息
	return c.handleMessages()
}

// connectToServer 连接到服务器
func (c *Client) connectToServer() error {
	var err error
	c.conn, err = net.DialTimeout("tcp", c.serverAddr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to dial server: %w", err)
	}

	c.connected = true
	log.Printf("Connected to server at %s", c.serverAddr)
	return nil
}

// startHeartbeat 启动心跳
func (c *Client) startHeartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !c.connected {
				// 尝试重连
				if err := c.connectToServer(); err != nil {
					log.Printf("Failed to reconnect: %v", err)
					continue
				}
			}

			// 发送心跳
			if err := c.sendHeartbeat(); err != nil {
				log.Printf("Failed to send heartbeat: %v", err)
				c.connected = false
				c.conn.Close()
			}
		}
	}
}

// sendHeartbeat 发送心跳
func (c *Client) sendHeartbeat() error {
	protocolMsg := util.ProtocolMessage{
		Type:      util.Heartbeat,
		MessageID: fmt.Sprintf("hb_%d", time.Now().UnixNano()),
		Data:      []byte("ping"),
		Headers:   make(map[string]string),
	}

	data, err := util.EncodeMessage(protocolMsg)
	if err != nil {
		return err
	}

	_, err = c.conn.Write(data)
	return err
}

// handleMessages 处理消息
func (c *Client) handleMessages() error {
	buffer := make([]byte, 8192)

	for {
		if !c.connected {
			// 等待重连
			time.Sleep(5 * time.Second)
			if err := c.connectToServer(); err != nil {
				log.Printf("Failed to reconnect: %v", err)
				continue
			}
		}

		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		n, err := c.conn.Read(buffer)
		if err != nil {
			log.Printf("Error reading from server: %v", err)
			c.connected = false
			c.conn.Close()
			continue
		}

		if err := c.handleMessage(buffer[:n]); err != nil {
			log.Printf("Error handling message: %v", err)
		}
	}
}

// handleMessage 处理消息
func (c *Client) handleMessage(data []byte) error {
	msg, err := util.DecodeMessage(data)
	if err != nil {
		return fmt.Errorf("decode message error: %w", err)
	}

	switch msg.Type {
	case util.HTTPRequest:
		go c.processHTTPRequest(msg)

	case util.Heartbeat:
		log.Printf("Heartbeat received from server")

	default:
		log.Printf("Unknown message type: %d", msg.Type)
	}

	return nil
}

// processHTTPRequest 处理HTTP请求
func (c *Client) processHTTPRequest(protocolMsg util.ProtocolMessage) {
	var requestData util.HTTPRequestData
	if err := json.Unmarshal(protocolMsg.Data, &requestData); err != nil {
		log.Printf("Failed to unmarshal request data: %v", err)
		c.sendErrorResponse(protocolMsg.MessageID, http.StatusBadRequest, "Invalid request data")
		return
	}

	log.Printf("Processing request: %s %s", requestData.Method, requestData.Path)

	// 根据路径处理不同的请求
	switch {
	case requestData.Path == "/" || requestData.Path == "/query":
		c.handleQueryRequest(protocolMsg.MessageID, &requestData)
	case requestData.Path == "/tables":
		c.handleTablesRequest(protocolMsg.MessageID, &requestData)
	case strings.HasPrefix(requestData.Path, "/table/"):
		tableName := strings.TrimPrefix(requestData.Path, "/table/")
		c.handleTableDataRequest(protocolMsg.MessageID, &requestData, tableName)
	default:
		c.sendErrorResponse(protocolMsg.MessageID, http.StatusNotFound, "Page not found")
	}
}

// handleQueryRequest 处理查询请求
func (c *Client) handleQueryRequest(messageID string, requestData *util.HTTPRequestData) {
	// 从查询参数中获取SQL
	query := ""
	if requestData.Method == "GET" {
		// 解析URL参数
		if strings.Contains(requestData.Path, "?") {
			parsedURL, err := url.Parse(requestData.Path)
			if err == nil {
				query = parsedURL.Query().Get("sql")
			}
		}
	} else if requestData.Method == "POST" {
		// 从POST body中获取SQL
		query = string(requestData.Body)
	}

	if query == "" {
		// 显示查询表单
		html := c.generateQueryForm("")
		c.sendHTMLResponse(messageID, http.StatusOK, html)
		return
	}

	// 执行SQL查询
	start := time.Now()
	results, err := c.executeSQLQuery(query)
	duration := time.Since(start)

	if err != nil {
		html := c.generateQueryForm(fmt.Sprintf("Error: %v", err))
		c.sendHTMLResponse(messageID, http.StatusOK, html)
		return
	}

	// 生成结果HTML
	html := c.generateQueryResults(query, results, duration)
	c.sendHTMLResponse(messageID, http.StatusOK, html)
}

// handleTablesRequest 处理表列表请求
func (c *Client) handleTablesRequest(messageID string, requestData *util.HTTPRequestData) {
	tables, err := c.getTableList()
	if err != nil {
		c.sendErrorResponse(messageID, http.StatusInternalServerError, fmt.Sprintf("Failed to get tables: %v", err))
		return
	}

	html := c.generateTablesHTML(tables)
	c.sendHTMLResponse(messageID, http.StatusOK, html)
}

// handleTableDataRequest 处理表数据请求
func (c *Client) handleTableDataRequest(messageID string, requestData *util.HTTPRequestData, tableName string) {
	query := fmt.Sprintf("SELECT * FROM %s LIMIT 100", tableName)

	start := time.Now()
	results, err := c.executeSQLQuery(query)
	duration := time.Since(start)

	if err != nil {
		c.sendErrorResponse(messageID, http.StatusInternalServerError, fmt.Sprintf("Failed to query table: %v", err))
		return
	}

	html := c.generateTableDataHTML(tableName, results, duration)
	c.sendHTMLResponse(messageID, http.StatusOK, html)
}

// executeSQLQuery 执行SQL查询
func (c *Client) executeSQLQuery(query string) (*QueryResult, error) {
	// 安全检查：只允许SELECT语句
	trimmedQuery := strings.TrimSpace(strings.ToUpper(query))
	if !strings.HasPrefix(trimmedQuery, "SELECT") {
		return nil, fmt.Errorf("only SELECT queries are allowed")
	}

	rows, err := c.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 获取列信息
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	// 准备数据
	var data [][]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		data = append(data, values)
	}

	return &QueryResult{
		Columns: columns,
		Data:    data,
		Count:   len(data),
	}, nil
}

// getTableList 获取表列表
func (c *Client) getTableList() ([]string, error) {
	rows, err := c.db.Query("SHOW TABLES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}

	return tables, nil
}

// sendHTMLResponse 发送HTML响应
func (c *Client) sendHTMLResponse(messageID string, statusCode int, html string) {
	responseData := util.HTTPResponseData{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type": "text/html; charset=utf-8",
		},
		Body: []byte(html),
	}

	data, err := json.Marshal(responseData)
	if err != nil {
		log.Printf("Failed to marshal response data: %v", err)
		return
	}

	protocolMsg := util.ProtocolMessage{
		Type:      util.HTTPResponse,
		MessageID: messageID,
		Data:      data,
		Headers:   make(map[string]string),
	}

	c.sendResponse(protocolMsg)
}

// sendErrorResponse 发送错误响应
func (c *Client) sendErrorResponse(messageID string, statusCode int, message string) {
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Error</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .error { color: #d32f2f; background: #ffebee; padding: 20px; border-radius: 4px; }
    </style>
</head>
<body>
    <div class="error">
        <h2>Error %d</h2>
        <p>%s</p>
    </div>
</body>
</html>`, statusCode, message)

	c.sendHTMLResponse(messageID, statusCode, html)
}

// sendResponse 发送响应
func (c *Client) sendResponse(protocolMsg util.ProtocolMessage) {
	data, err := util.EncodeMessage(protocolMsg)
	if err != nil {
		log.Printf("Failed to encode response: %v", err)
		return
	}

	if _, err := c.conn.Write(data); err != nil {
		log.Printf("Failed to send response: %v", err)
		c.connected = false
		c.conn.Close()
	}
}

// QueryResult 查询结果
type QueryResult struct {
	Columns []string        `json:"columns"`
	Data    [][]interface{} `json:"data"`
	Count   int             `json:"count"`
}

// generateQueryForm 生成查询表单
func (c *Client) generateQueryForm(errorMsg string) string {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>SQL Query</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .container { max-width: 1000px; margin: 0 auto; }
        textarea { width: 100%%; height: 100px; margin: 10px 0; padding: 10px; border: 1px solid #ddd; border-radius: 4px; }
        button { background: #007bff; color: white; padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; }
        button:hover { background: #0056b3; }
        .error { color: #d32f2f; background: #ffebee; padding: 10px; margin: 10px 0; border-radius: 4px; }
        .links { margin: 20px 0; }
        .links a { margin-right: 15px; color: #007bff; text-decoration: none; }
        .links a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <div class="container">
        <h1>SQL Query Interface</h1>
        
        <div class="links">
            <a href="/tables">Show Tables</a>
            <a href="/query">New Query</a>
        </div>
        
        {{if .Error}}
        <div class="error">{{.Error}}</div>
        {{end}}
        
        <form method="post" action="/query">
            <label for="sql">SQL Query (SELECT only):</label><br>
            <textarea name="sql" id="sql" placeholder="SELECT * FROM users LIMIT 10"></textarea><br>
            <button type="submit">Execute Query</button>
        </form>
    </div>
</body>
</html>`

	t, _ := template.New("query").Parse(tmpl)
	var buf bytes.Buffer
	t.Execute(&buf, map[string]interface{}{"Error": errorMsg})
	return buf.String()
}

// generateQueryResults 生成查询结果
func (c *Client) generateQueryResults(query string, result *QueryResult, duration time.Duration) string {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Query Results</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .container { max-width: 100%; margin: 0 auto; }
        .query { background: #f5f5f5; padding: 15px; margin: 10px 0; border-radius: 4px; }
        table { border-collapse: collapse; width: 100%%; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background: #f2f2f2; }
        tr:nth-child(even) { background: #f9f9f9; }
        .info { margin: 10px 0; color: #666; }
        .links { margin: 20px 0; }
        .links a { margin-right: 15px; color: #007bff; text-decoration: none; }
        .links a:hover { text-decoration: underline; }
        .no-results { text-align: center; color: #666; margin: 20px 0; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Query Results</h1>
        
        <div class="links">
            <a href="/tables">Show Tables</a>
            <a href="/query">New Query</a>
        </div>
        
        <div class="query">
            <strong>Query:</strong><br>
            <code>{{.Query}}</code>
        </div>
        
        <div class="info">
            {{.Count}} rows returned | Execution time: {{.Duration}}
        </div>
        
        {{if .HasData}}
        <div style="overflow-x: auto;">
            <table>
                <thead>
                    <tr>
                        {{range .Columns}}
                        <th>{{.}}</th>
                        {{end}}
                    </tr>
                </thead>
                <tbody>
                    {{range .Data}}
                    <tr>
                        {{range .}}
                        <td>{{.}}</td>
                        {{end}}
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </div>
        {{else}}
        <div class="no-results">No results found</div>
        {{end}}
    </div>
</body>
</html>`

	data := map[string]interface{}{
		"Query":    query,
		"Count":    result.Count,
		"Duration": duration.String(),
		"HasData":  result.Count > 0,
		"Columns":  result.Columns,
		"Data":     result.Data,
	}

	t, _ := template.New("results").Parse(tmpl)
	var buf bytes.Buffer
	t.Execute(&buf, data)
	return buf.String()
}

// generateTablesHTML 生成表列表HTML
func (c *Client) generateTablesHTML(tables []string) string {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Database Tables</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .container { max-width: 800px; margin: 0 auto; }
        table { border-collapse: collapse; width: 100%%; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 12px; text-align: left; }
        th { background: #f2f2f2; }
        tr:hover { background: #f5f5f5; }
        .links { margin: 20px 0; }
        .links a { margin-right: 15px; color: #007bff; text-decoration: none; }
        .links a:hover { text-decoration: underline; }
        .count { text-align: center; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Database Tables</h1>
        
        <div class="links">
            <a href="/tables">Show Tables</a>
            <a href="/query">New Query</a>
        </div>
        
        <div class="count">Found {{.Count}} tables</div>
        
        <table>
            <thead>
                <tr>
                    <th>Table Name</th>
                    <th>Actions</th>
                </tr>
            </thead>
            <tbody>
                {{range .Tables}}
                <tr>
                    <td>{{.}}</td>
                    <td>
                        <a href="/table/{{.}}">View Data</a>
                    </td>
                </tr>
                {{end}}
            </tbody>
        </table>
    </div>
</body>
</html>`

	data := map[string]interface{}{
		"Tables": tables,
		"Count":  len(tables),
	}

	t, _ := template.New("tables").Parse(tmpl)
	var buf bytes.Buffer
	t.Execute(&buf, data)
	return buf.String()
}

// generateTableDataHTML 生成表数据HTML
func (c *Client) generateTableDataHTML(tableName string, result *QueryResult, duration time.Duration) string {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Table Data: {{.TableName}}</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .container { max-width: 100%%; margin: 0 auto; }
        .info { background: #f5f5f5; padding: 15px; margin: 10px 0; border-radius: 4px; }
        table { border-collapse: collapse; width: 100%%; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background: #f2f2f2; }
        tr:nth-child(even) { background: #f9f9f9; }
        .links { margin: 20px 0; }
        .links a { margin-right: 15px; color: #007bff; text-decoration: none; }
        .links a:hover { text-decoration: underline; }
        .no-results { text-align: center; color: #666; margin: 20px 0; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Table Data: {{.TableName}}</h1>
        
        <div class="links">
            <a href="/tables">Show Tables</a>
            <a href="/query">New Query</a>
            <a href="/table/{{.TableName}}">Refresh</a>
        </div>
        
        <div class="info">
            Showing first {{.Count}} rows | Execution time: {{.Duration}}
            <br><small>Limited to 100 rows. Use query interface for more data.</small>
        </div>
        
        {{if .HasData}}
        <div style="overflow-x: auto;">
            <table>
                <thead>
                    <tr>
                        {{range .Columns}}
                        <th>{{.}}</th>
                        {{end}}
                    </tr>
                </thead>
                <tbody>
                    {{range .Data}}
                    <tr>
                        {{range .}}
                        <td>{{.}}</td>
                        {{end}}
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </div>
        {{else}}
        <div class="no-results">No data found in table</div>
        {{end}}
    </div>
</body>
</html>`

	data := map[string]interface{}{
		"TableName": tableName,
		"Count":     result.Count,
		"Duration":  duration.String(),
		"HasData":   result.Count > 0,
		"Columns":   result.Columns,
		"Data":      result.Data,
	}

	t, _ := template.New("tabledata").Parse(tmpl)
	var buf bytes.Buffer
	t.Execute(&buf, data)
	return buf.String()
}

const (
	httpStatusOK                  = 200
	httpStatusBadRequest          = 400
	httpStatusNotFound            = 404
	httpStatusInternalServerError = 500
)

func QosOld(config *config.Config) {
	// 配置服务器地址和数据库连接
	serverAddr := "localhost:58089" // Server A的TCP端口
	dbDSN := "user:password@tcp(localhost:3306)/database?charset=utf8mb4&parseTime=True&loc=Local"

	log.Println("Starting proxy client B...")

	client, err := NewClient(serverAddr, dbDSN)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	if err := client.Start(); err != nil {
		log.Fatalf("Failed to start client: %v", err)
	}
}
