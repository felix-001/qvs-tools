package util

import (
	"encoding/json"
	"fmt"
)

// MessageType 消息类型
type MessageType int

const (
	// HTTPRequest HTTP请求消息
	HTTPRequest MessageType = iota + 1
	// HTTPResponse HTTP响应消息
	HTTPResponse
	// Heartbeat 心跳消息
	Heartbeat
)

// ProtocolMessage 协议消息
type ProtocolMessage struct {
	Type      MessageType       `json:"type"`
	MessageID string            `json:"messageId"`
	Data      []byte            `json:"data"`
	Headers   map[string]string `json:"headers"`
}

// HTTPRequestData HTTP请求数据
type HTTPRequestData struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers"`
	Body    []byte            `json:"body"`
}

// HTTPResponseData HTTP响应数据
type HTTPResponseData struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       []byte            `json:"body"`
}

// EncodeMessage 编码消息
func EncodeMessage(msg ProtocolMessage) ([]byte, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal message error: %w", err)
	}

	// 添加长度前缀
	length := len(data)
	result := make([]byte, 4+length)
	result[0] = byte(length >> 24)
	result[1] = byte(length >> 16)
	result[2] = byte(length >> 8)
	result[3] = byte(length)
	copy(result[4:], data)

	return result, nil
}

// DecodeMessage 解码消息
func DecodeMessage(data []byte) (ProtocolMessage, error) {
	if len(data) < 4 {
		return ProtocolMessage{}, fmt.Errorf("data too short")
	}

	length := int(data[0])<<24 | int(data[1])<<16 | int(data[2])<<8 | int(data[3])
	if len(data) < 4+length {
		return ProtocolMessage{}, fmt.Errorf("data incomplete")
	}

	var msg ProtocolMessage
	err := json.Unmarshal(data[4:4+length], &msg)
	if err != nil {
		return ProtocolMessage{}, fmt.Errorf("unmarshal message error: %w", err)
	}

	return msg, nil
}
