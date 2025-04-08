package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// sendWeChatAlert 向企业微信发送告警
func sendWeChatAlert(message string) error {
	webhookURL := "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=59b3d537-8160-4870-ba92-31245c4b4729" // 替换为你的企业微信Webhook URL

	// 构建消息内容
	msg := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": message,
		},
	}

	// 将消息内容转换为JSON
	jsonMsg, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// 发送HTTP POST请求
	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonMsg))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send message, status code: %d", resp.StatusCode)
	}

	return nil
}
