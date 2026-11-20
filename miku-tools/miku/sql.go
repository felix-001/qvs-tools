package miku

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	_ "embed"
	"mikutool/config"
	"mikutool/public/util"
)

//go:embed sql/html/index.html
var sqlPageHTML string

const (
	defaultSQLCatalog = "hive_miku"
	defaultSQLSchema  = "miku"
	defaultSQLPort    = 8081
	sqlQueryTimeout   = 3 * time.Minute
)

// SQLServer SQL 查询页面 (前端构造 SQL，后端执行 trino 查询)
type SQLServer struct {
	config *config.Config
	port   int

	mu      sync.Mutex
	running map[string]context.CancelFunc // requestId -> cancel
}

func NewSQLServer(cfg *config.Config) *SQLServer {
	port := defaultSQLPort
	if cfg != nil && cfg.Port > 0 {
		port = cfg.Port
	}
	return &SQLServer{config: cfg, port: port, running: make(map[string]context.CancelFunc)}
}

// register 登记一个运行中的查询，便于外部强制取消
func (s *SQLServer) register(id string, cancel context.CancelFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.running[id] = cancel
}

func (s *SQLServer) unregister(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.running, id)
}

// cancelQuery 取消指定查询，返回是否找到了对应的运行中查询
func (s *SQLServer) cancelQuery(id string) bool {
	s.mu.Lock()
	cancel, ok := s.running[id]
	s.mu.Unlock()
	if !ok {
		return false
	}
	cancel()
	return true
}

// cancelAll 取消全部运行中的查询
func (s *SQLServer) cancelAll() int {
	s.mu.Lock()
	cancels := make([]context.CancelFunc, 0, len(s.running))
	for _, c := range s.running {
		cancels = append(cancels, c)
	}
	s.mu.Unlock()
	for _, c := range cancels {
		c()
	}
	return len(cancels)
}

func (s *SQLServer) StartServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.homeHandler)
	mux.HandleFunc("/api/v1/catalogs", s.catalogsHandler)
	mux.HandleFunc("/api/v1/schemas", s.schemasHandler)
	mux.HandleFunc("/api/v1/tables", s.tablesHandler)
	mux.HandleFunc("/api/v1/columns", s.columnsHandler)
	mux.HandleFunc("/api/v1/query", s.queryHandler)
	mux.HandleFunc("/api/v1/query/cancel", s.cancelQueryHandler)

	fmt.Println("SQL 查询页面启动成功！")
	fmt.Printf("请访问: http://localhost:%d\n", s.port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", s.port), mux))
}

func (s *SQLServer) homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(sqlPageHTML))
}

// catalogsHandler 返回 trino 中所有 catalog(database)
func (s *SQLServer) catalogsHandler(w http.ResponseWriter, r *http.Request) {
	rs, err := util.TrinoQueryRows("", "", "SHOW CATALOGS")
	if err != nil {
		writeJSONError(w, err)
		return
	}
	names := rs.FirstColumn()
	if len(names) == 0 {
		names = []string{defaultSQLCatalog}
	}
	writeJSON(w, map[string]any{
		"catalogs": names,
		"default":  defaultSQLCatalog,
	})
}

// schemasHandler 返回指定 catalog 下的 schema
func (s *SQLServer) schemasHandler(w http.ResponseWriter, r *http.Request) {
	catalog := queryParam(r, "catalog", defaultSQLCatalog)
	sql := fmt.Sprintf("SHOW SCHEMAS FROM %s", quoteIdent(catalog))
	rs, err := util.TrinoQueryRows("", "", sql)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, map[string]any{
		"schemas": rs.FirstColumn(),
		"default": defaultSQLSchema,
	})
}

// tablesHandler 列出 schema 下的所有表
func (s *SQLServer) tablesHandler(w http.ResponseWriter, r *http.Request) {
	catalog := queryParam(r, "catalog", defaultSQLCatalog)
	schema := queryParam(r, "schema", defaultSQLSchema)
	sql := fmt.Sprintf(`SELECT table_name FROM %s.information_schema.tables
WHERE table_schema = '%s' ORDER BY table_name`, quoteIdent(catalog), escSQL(schema))
	rs, err := util.TrinoQueryRows("", "", sql)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, map[string]any{"tables": rs.FirstColumn()})
}

// columnsHandler 返回表的全部字段名(供前端复选框展示)
func (s *SQLServer) columnsHandler(w http.ResponseWriter, r *http.Request) {
	catalog := queryParam(r, "catalog", defaultSQLCatalog)
	schema := queryParam(r, "schema", defaultSQLSchema)
	table := queryParam(r, "table", "")
	if table == "" {
		writeJSON(w, map[string]any{"columns": []any{}})
		return
	}
	sql := fmt.Sprintf(`SELECT column_name, data_type FROM %s.information_schema.columns
WHERE table_schema = '%s' AND table_name = '%s' ORDER BY ordinal_position`,
		quoteIdent(catalog), escSQL(schema), escSQL(table))
	rs, err := util.TrinoQueryRowsCtx(r.Context(), "", "", sql)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	columns := make([]map[string]string, 0, len(rs.Rows))
	for _, row := range rs.Rows {
		if len(row) < 2 {
			continue
		}
		columns = append(columns, map[string]string{
			"name": fmt.Sprint(row[0]),
			"type": fmt.Sprint(row[1]),
		})
	}
	writeJSON(w, map[string]any{"columns": columns})
}

type sqlQueryRequest struct {
	Catalog   string `json:"catalog"`
	Schema    string `json:"schema"`
	SQL       string `json:"sql"`
	RequestID string `json:"request_id"`
}

// queryHandler 执行前端构造的 SQL
func (s *SQLServer) queryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req sqlQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, fmt.Errorf("请求解析失败: %v", err))
		return
	}
	req.SQL = strings.TrimSpace(req.SQL)
	if req.SQL == "" {
		writeJSONError(w, fmt.Errorf("SQL 不能为空"))
		return
	}
	if req.Catalog == "" {
		req.Catalog = defaultSQLCatalog
	}
	if req.RequestID == "" {
		req.RequestID = fmt.Sprintf("q_%d", time.Now().UnixNano())
	}

	ctx, cancel := context.WithTimeout(r.Context(), sqlQueryTimeout)
	s.register(req.RequestID, cancel)
	defer func() {
		s.unregister(req.RequestID)
		cancel()
	}()

	start := time.Now()
	rs, err := util.TrinoQueryRowsCtx(ctx, req.Catalog, req.Schema, req.SQL)
	elapsed := time.Since(start).Milliseconds()
	if err != nil {
		if ctx.Err() == context.Canceled || util.IsQueryCancelled(err) {
			log.Printf("sql query cancelled, requestId=%s elapsed=%dms sql=%s", req.RequestID, elapsed, req.SQL)
			writeJSON(w, map[string]any{
				"error":      "查询已取消",
				"cancelled":  true,
				"request_id": req.RequestID,
				"elapsed_ms": elapsed,
			})
			return
		}
		log.Printf("sql query failed, catalog=%s schema=%s err=%v sql=%s", req.Catalog, req.Schema, err, req.SQL)
		writeJSONError(w, err)
		return
	}
	log.Printf("sql query done, rows=%d elapsed=%dms sql=%s", len(rs.Rows), elapsed, req.SQL)

	writeJSON(w, map[string]any{
		"columns":    rs.Columns,
		"rows":       rs.Rows,
		"row_count":  len(rs.Rows),
		"elapsed_ms": elapsed,
		"sql":        req.SQL,
		"request_id": req.RequestID,
	})
}

// cancelQueryHandler 强制停止正在执行的查询
// POST /api/v1/query/cancel  {"request_id":"..."}；request_id 为空时取消全部运行中的查询
func (s *SQLServer) cancelQueryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		RequestID string `json:"request_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	req.RequestID = strings.TrimSpace(req.RequestID)

	if req.RequestID == "" {
		n := s.cancelAll()
		log.Printf("sql cancel all: %d running queries", n)
		writeJSON(w, map[string]any{"cancelled": n > 0, "count": n})
		return
	}
	ok := s.cancelQuery(req.RequestID)
	count := 0
	if ok {
		count = 1
	}
	log.Printf("sql cancel requestId=%s found=%v", req.RequestID, ok)
	writeJSON(w, map[string]any{"cancelled": ok, "count": count, "request_id": req.RequestID})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, err error) {
	writeJSON(w, map[string]any{"error": err.Error()})
}

func queryParam(r *http.Request, key, def string) string {
	v := strings.TrimSpace(r.URL.Query().Get(key))
	if v == "" {
		return def
	}
	return v
}

// quoteIdent 双引号包裹标识符，避免特殊字符或关键字冲突
func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// escSQL 转义字符串字面量中的单引号
func escSQL(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// SQL 启动 SQL 查询页面
func SQL(cfg *config.Config) {
	NewSQLServer(cfg).StartServer()
}
