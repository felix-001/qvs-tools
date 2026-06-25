package miku

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"mikutool/config"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	ansiCSI2  = regexp.MustCompile(`\x1b\[\??[0-9;]*[a-zA-Z~]`)
	ansiOSC2  = regexp.MustCompile(`\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)`)
	ansiParen = regexp.MustCompile(`(?i)\x1b[\(\)][A-Z0-9]`)
	ansiMisc  = regexp.MustCompile(`\x1b[>=<~A-Za-z]`)
	ansiLeft  = regexp.MustCompile(`\x1b`)
)

type wsExecSettings struct {
	wsHost        string
	adminHost     string
	usertokenPath string
	auth          string
	authFile      string
	loginURL      string
	targetURL     string
	userFile      string
	passFile      string
	totpBin       string
	totpProfile   string
	timeoutSec    int
	columns       int
	rows          int
}

// WsExec 通过 GoTTY WebSocket 在远程节点执行命令。
func (m *Miku) WsExec(cfg *config.Config) {
	if cfg.Node == "" {
		log.Fatal("Usage: miku -cmd ssh -node <nodeId> [-query <command>] [-t <timeout_sec>]")
	}

	cmd := cfg.Query
	if cmd == "" {
		cmd = "hostname"
	}

	settings := resolveWsExecSettings(cfg)
	output, err := wsExecOnNode(settings, cfg.Node, cmd)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(output)
}

func resolveWsExecSettings(cfg *config.Config) wsExecSettings {
	ws := cfg.WsExec
	s := wsExecSettings{
		wsHost:        ws.WsHost,
		adminHost:     ws.AdminHost,
		usertokenPath: ws.UsertokenPath,
		auth:          strings.TrimSpace(ws.Auth),
		authFile:      ws.AuthFile,
		loginURL:      ws.LoginURL,
		targetURL:     ws.TargetURL,
		userFile:      ws.UserFile,
		passFile:      ws.PassFile,
		totpBin:       ws.TotpBin,
		totpProfile:   ws.TotpProfile,
		timeoutSec:    ws.TimeoutSec,
		columns:       ws.Columns,
		rows:          ws.Rows,
	}
	if s.wsHost == "" {
		s.wsHost = "jarvis-ws-inner-v2.niulinkcloud.com"
	}
	if s.adminHost == "" {
		s.adminHost = "linkcloud-admin.qiniu.io"
	}
	if s.usertokenPath == "" {
		s.usertokenPath = "/api/proxy/jarvis/v1/webshell/usertoken"
	}
	if s.authFile == "" {
		s.authFile = "/usr/local/etc/linkcloud_auth.txt"
	}
	if s.loginURL == "" {
		s.loginURL = defaultLinkcloudLoginURL
	}
	if s.targetURL == "" {
		s.targetURL = defaultLinkcloudTargetURL
	}
	if s.userFile == "" {
		s.userFile = "~/.secrets/linkcloud_user"
	}
	if s.passFile == "" {
		s.passFile = "~/.secrets/linkcloud_pass"
	}
	if s.totpBin == "" {
		s.totpBin = defaultTotpBin()
	}
	if s.totpProfile == "" {
		s.totpProfile = defaultTotpProfile()
	}
	if s.timeoutSec == 0 {
		s.timeoutSec = 30
	}
	if s.columns == 0 {
		s.columns = 500
	}
	if s.rows == 0 {
		s.rows = 50
	}
	if cfg.T != "" {
		if n, err := strconv.Atoi(cfg.T); err == nil && n > 0 {
			s.timeoutSec = n
		}
	}
	return s
}

type usertokenResp struct {
	Token string `json:"token"`
}

func fetchWsToken(s wsExecSettings) (string, error) {
	auth, fromEnv, err := getInitialAuth(s)
	if err != nil {
		return "", err
	}
	body, status, err := callUsertoken(s, auth)
	if err != nil {
		return "", err
	}
	if status == http.StatusUnauthorized && !fromEnv {
		log.Println("[auth] usertoken returned 401, refreshing admin token then retrying once")
		auth, err = refreshAuth(s)
		if err != nil {
			return "", err
		}
		body, status, err = callUsertoken(s, auth)
		if err != nil {
			return "", err
		}
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("usertoken HTTP %d: %s", status, truncate(body, 200))
	}
	var parsed usertokenResp
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return "", fmt.Errorf("usertoken response not JSON: %s", truncate(body, 200))
	}
	if parsed.Token == "" {
		return "", fmt.Errorf("usertoken response missing \"token\" field: %s", truncate(body, 200))
	}
	return parsed.Token, nil
}

func getInitialAuth(s wsExecSettings) (auth string, fromEnv bool, err error) {
	if v := strings.TrimSpace(os.Getenv("LINKCLOUD_AUTH")); v != "" {
		return v, true, nil
	}
	if s.auth != "" {
		return s.auth, false, nil
	}
	if cached, readErr := os.ReadFile(s.authFile); readErr == nil {
		if v := strings.TrimSpace(string(cached)); v != "" {
			return v, false, nil
		}
	}
	fresh, err := refreshAuth(s)
	if err != nil {
		return "", false, err
	}
	return fresh, false, nil
}

func refreshAuth(s wsExecSettings) (string, error) {
	log.Println("[auth] refreshing via LinkCloud SSO login")
	fresh, err := getLinkcloudAuth(s)
	if err != nil {
		return "", err
	}
	if fresh == "" {
		return "", errors.New("linkcloud auth extraction produced empty result")
	}
	if err := writeCachedAuth(s.authFile, fresh); err != nil {
		return "", err
	}
	return fresh, nil
}

func writeCachedAuth(path, auth string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.TrimSpace(auth)+"\n"), 0o600)
}

func callUsertoken(s wsExecSettings, auth string) (body string, status int, err error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "https://"+s.adminHost+s.usertokenPath, nil)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Authorization", auth)
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("usertoken request: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", resp.StatusCode, err
	}
	return string(data), resp.StatusCode, nil
}

func wsExecOnNode(s wsExecSettings, nodeID, cmd string) (string, error) {
	wsToken, err := fetchWsToken(s)
	if err != nil {
		return "", fmt.Errorf("failed to fetch WS token: %w", err)
	}

	suffix := fmt.Sprintf("%d_%s", time.Now().UnixMilli(), randomAlphaNum(6))
	startMarker := "___START_" + suffix + "___"
	endMarker := "___END_" + suffix + "___"

	wsURL := fmt.Sprintf("wss://%s/wsv1/web2server?nodeID=%s&token=%s",
		s.wsHost, nodeID, url.QueryEscape(wsToken))

	dialer := websocket.Dialer{
		TLSClientConfig:  &tls.Config{MinVersion: tls.VersionTLS12},
		HandshakeTimeout: time.Duration(s.timeoutSec) * time.Second,
	}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return "", fmt.Errorf("websocket dial: %w", err)
	}
	defer conn.Close()

	var (
		mu        sync.Mutex
		outputBuf strings.Builder
		done      = make(chan struct{})
	)

	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			mu.Lock()
			outputBuf.Write(message)
			buf := outputBuf.String()
			hasEnd := strings.Count(buf, endMarker) >= 2
			mu.Unlock()
			if hasEnd {
				time.Sleep(300 * time.Millisecond)
				close(done)
				return
			}
		}
	}()

	pingStop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_ = conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second))
			case <-pingStop:
				return
			case <-done:
				return
			}
		}
	}()
	defer close(pingStop)

	send := func(payload string) error {
		return conn.WriteMessage(websocket.BinaryMessage, []byte(payload))
	}

	resize, _ := json.Marshal(map[string]int{"columns": s.columns, "rows": s.rows})
	if err := send("1" + string(resize)); err != nil {
		return "", fmt.Errorf("send resize: %w", err)
	}
	time.Sleep(800 * time.Millisecond)
	if err := send("0\x03"); err != nil {
		return "", fmt.Errorf("send ctrl-c: %w", err)
	}
	time.Sleep(500 * time.Millisecond)
	if err := send(fmt.Sprintf("0stty cols %d rows %d\n", s.columns, s.rows)); err != nil {
		return "", fmt.Errorf("send stty: %w", err)
	}
	time.Sleep(500 * time.Millisecond)

	b64 := base64.StdEncoding.EncodeToString([]byte(cmd))
	wrapped := fmt.Sprintf("echo %s && eval \"$(echo %s | base64 -d)\"; echo %s\n", startMarker, b64, endMarker)
	if err := send("0" + wrapped); err != nil {
		return "", fmt.Errorf("send command: %w", err)
	}

	timer := time.NewTimer(time.Duration(s.timeoutSec) * time.Second)
	defer timer.Stop()

	select {
	case <-done:
	case <-readDone:
	case <-timer.C:
		mu.Lock()
		partial := outputBuf.String()
		mu.Unlock()
		if partial != "" {
			log.Printf("[TIMEOUT after %ds]", s.timeoutSec)
			log.Printf("[Partial]:\n%s", truncate(stripANSI(partial), 800))
		}
		return "", fmt.Errorf("[TIMEOUT after %ds]", s.timeoutSec)
	}

	mu.Lock()
	output := outputBuf.String()
	mu.Unlock()

	result := extractWsExecOutput(output, startMarker, endMarker)
	if result == "" && strings.Count(stripANSI(output), endMarker) < 2 {
		return "", fmt.Errorf("command output not captured (terminal may have wrapped the command; got partial echo)")
	}
	return result, nil
}

func extractWsExecOutput(output, startMarker, endMarker string) string {
	clean := stripANSI(output)
	s := strings.LastIndex(clean, startMarker)
	e := strings.LastIndex(clean, endMarker)

	var result string
	if s >= 0 && e > s {
		result = strings.TrimSpace(clean[s+len(startMarker) : e])
	} else if e >= 0 {
		result = strings.TrimSpace(clean[:e])
	} else {
		result = strings.TrimSpace(clean)
	}

	// 只捕获到命令回显里的 marker（折行导致命令未执行）时会误提取 eval/base64 片段
	if strings.Contains(result, "base64 -d") || strings.Contains(result, `eval "$(echo`) {
		return ""
	}
	return result
}

func stripANSI(text string) string {
	text = ansiCSI2.ReplaceAllString(text, "")
	text = ansiOSC2.ReplaceAllString(text, "")
	text = ansiParen.ReplaceAllString(text, "")
	text = ansiMisc.ReplaceAllString(text, "")
	text = ansiLeft.ReplaceAllString(text, "")
	return strings.ReplaceAll(text, "\r", "")
}

func randomAlphaNum(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
