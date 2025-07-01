package qvs

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"encoding/json"
	"net/http"
)

// Message 定义了企业微信消息的结构
type Message struct {
	MsgType string `json:"msgtype"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text"`
}

// sendWeChatMessage 向企业微信发送消息
func sendWeChatMessage(webhookURL string, message Message) error {
	jsonData, err := json.Marshal(message)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonData))
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

// getTotalFileSize 获取指定目录下所有文件的大小总和
func getTotalFileSize(dirPath string) (int64, error) {
	var totalSize int64
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	return totalSize, err
}

func Tcpdump() {
	dirPath := "/root/liyq"
	webhookURL := "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=" + Conf.SendKey

	// 每隔10秒执行一次
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			totalSize, err := getTotalFileSize(dirPath)
			if err != nil {
				log.Printf("Error getting total file size: %v", err)
				continue
			}

			if totalSize > 3000 {
				message := Message{
					MsgType: "text",
					Text: struct {
						Content string `json:"content"`
					}{
						Content: fmt.Sprintf("Alert: Total file size in %s is %d bytes, which exceeds 1000 bytes.", dirPath, totalSize),
					},
				}
				err = sendWeChatMessage(webhookURL, message)
				if err != nil {
					log.Printf("Error sending WeChat message: %v", err)
				} else {
					//log.Println("WeChat message sent successfully.")
				}
			}
		}
	}
}
