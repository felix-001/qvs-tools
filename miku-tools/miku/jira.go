package miku

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	JIRA_URL  = "http://jira.qiniu.io"
	USERNAME  = "liyuanquan"
	PASSWORD  = "Swayinwind*1987" // 注意：生产环境建议使用 API Token
	ISSUE_KEY = "MIKU-2293"
)

func GetIssue() {
	// 1. 创建 Basic Auth 认证头
	credentials := fmt.Sprintf("%s:%s", USERNAME, PASSWORD)
	encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
	authHeader := fmt.Sprintf("Basic %s", encoded)

	// 2. 创建请求
	url := fmt.Sprintf("%s/rest/api/2/issue/%s", JIRA_URL, ISSUE_KEY)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("创建请求失败: %v\n", err)
		return
	}

	// 3. 设置请求头
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")

	// 4. 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// 5. 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("读取响应失败: %v\n", err)
		return
	}

	// 6. 检查状态码
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("请求异常，状态码: %d, 响应: %s\n", resp.StatusCode, string(body))
		return
	}

	// 7. 解析 JSON（可选：定义结构体解析，或直接打印）
	// 方式一：直接打印原始 JSON
	//fmt.Println("响应内容:", string(body))
	fmt.Println(string(body))

	// 方式二：解析为 map（灵活但不类型安全）
	/*
	   var result map[string]interface{}
	   if err := json.Unmarshal(body, &result); err != nil {
	           fmt.Printf("JSON解析失败: %v\n", err)
	           return
	   }
	   fmt.Printf("解析结果: %+v\n", result)
	*/

	// 方式三：定义结构体解析（类型安全，推荐生产使用）
	/*
	   var issue Issue
	   if err := json.Unmarshal(body, &issue); err != nil {
	           fmt.Printf("JSON解析失败: %v\n", err)
	           return
	   }
	   fmt.Printf("Issue Key: %s, Summary: %s\n", issue.Key, issue.Fields.Summary)
	*/
}

// Issue 结构体（按需扩展字段）
type Issue struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Fields struct {
		Summary     string `json:"summary"`
		Description string `json:"description"`
		Status      struct {
			Name string `json:"name"`
		} `json:"status"`
	} `json:"fields"`
}

// CreateIssueRequest 创建 issue 的请求结构体
type CreateIssueRequest struct {
	Fields struct {
		Project struct {
			Key string `json:"key"`
		} `json:"project"`
		Summary     string `json:"summary"`
		Description string `json:"description"`
		IssueType   struct {
			Name string `json:"name"`
		} `json:"issuetype"`
	} `json:"fields"`
}

// CreateIssue 创建 JIRA issue
func CreateIssue(projectKey, summary, description, issueType string) {
	// 1. 创建 Basic Auth 认证头
	credentials := fmt.Sprintf("%s:%s", USERNAME, PASSWORD)
	encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
	authHeader := fmt.Sprintf("Basic %s", encoded)

	// 2. 构建请求体
	reqBody := CreateIssueRequest{}
	//reqBody.Fields.Project.Key = projectKey
	reqBody.Fields.Summary = summary
	reqBody.Fields.Description = description
	reqBody.Fields.IssueType.Name = issueType

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Printf("JSON 编码失败: %v\n", err)
		return
	}

	// 3. 创建请求
	url := fmt.Sprintf("%s/rest/api/2/issue", JIRA_URL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("创建请求失败: %v\n", err)
		return
	}

	// 4. 设置请求头
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")

	// 5. 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// 6. 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("读取响应失败: %v\n", err)
		return
	}

	// 7. 检查状态码
	if resp.StatusCode != http.StatusCreated {
		fmt.Printf("创建 issue 失败，状态码: %d, 响应: %s\n", resp.StatusCode, string(body))
		return
	}

	// 8. 解析响应
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("JSON解析失败: %v\n", err)
		return
	}

	fmt.Printf("创建 issue 成功! Key: %s, ID: %s\n", result["key"], result["id"])
}
