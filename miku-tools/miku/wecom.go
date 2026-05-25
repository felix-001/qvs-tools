package miku

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// WeComRobot 企业微信机器人
type WeComRobot struct {
	WebhookURL string
}

// WeComMessage 企业微信消息
type WeComMessage struct {
	MsgType  string           `json:"msgtype"`
	Markdown WeComMarkdownMsg `json:"markdown"`
}

// WeComMarkdownMsg markdown消息
type WeComMarkdownMsg struct {
	Content string `json:"content"`
}

// NewWeComRobot 创建企业微信机器人
func NewWeComRobot(webhookURL string) *WeComRobot {
	return &WeComRobot{
		WebhookURL: webhookURL,
	}
}

// SendMarkdown 发送markdown消息
func (r *WeComRobot) SendMarkdown(content string) error {
	if r.WebhookURL == "" {
		return fmt.Errorf("webhook URL is empty")
	}

	msg := WeComMessage{
		MsgType: "markdown",
		Markdown: WeComMarkdownMsg{
			Content: content,
		},
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message failed: %v", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(r.WebhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("send request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	log.Printf("[WeCom] 告警消息发送成功")
	return nil
}

// SendPanicAlert 发送panic告警
func (r *WeComRobot) SendPanicAlert(node string, panicLog string) error {
	content := fmt.Sprintf(`## <font color="warning">【miku-sched Panic 告警】</font>

> **节点：** %s
> **时间：** %s
> **状态：** 发现 panic 日志，已备份

<font color="comment">最近 panic 日志：</font>

%s`, node, time.Now().Format("2006-01-02 15:04:05"), panicLog)

	return r.SendMarkdown(content)
}
