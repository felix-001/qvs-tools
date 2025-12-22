package qvs

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/smtp"
	"strings"
	"time"
)

type MailConfig struct {
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	From         string `json:"from"`
	UseTLS       bool   `json:"use_tls"`
}

type MailMessage struct {
	To      []string `json:"to"`
	CC      []string `json:"cc"`
	BCC     []string `json:"bcc"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
	IsHTML  bool     `json:"is_html"`
}

// SendMail 发送邮件
func SendMail(mailConfig *MailConfig, message *MailMessage) error {
	if mailConfig == nil {
		return fmt.Errorf("邮件配置不能为空")
	}
	if message == nil {
		return fmt.Errorf("邮件内容不能为空")
	}
	if len(message.To) == 0 {
		return fmt.Errorf("收件人不能为空")
	}

	// 构建邮件内容
	var content strings.Builder
	content.WriteString("To: " + strings.Join(message.To, ","))
	if len(message.CC) > 0 {
		content.WriteString("\nCc: " + strings.Join(message.CC, ","))
	}
	content.WriteString("\nSubject: " + message.Subject)
	if message.IsHTML {
		content.WriteString("\nMIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n")
	} else {
		content.WriteString("\nMIME-version: 1.0;\nContent-Type: text/plain; charset=\"UTF-8\";\n\n")
	}
	content.WriteString(message.Body)

	// 构建SMTP地址
	addr := fmt.Sprintf("%s:%d", mailConfig.SMTPHost, mailConfig.SMTPPort)

	// 认证信息
	auth := smtp.PlainAuth("", mailConfig.Username, mailConfig.Password, mailConfig.SMTPHost)

	// 发送邮件
	var err error
	if mailConfig.UseTLS {
		// 使用TLS加密连接
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         mailConfig.SMTPHost,
		}
		
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("TLS连接失败: %v", err)
		}
		defer conn.Close()
		
		client, err := smtp.NewClient(conn, mailConfig.SMTPHost)
		if err != nil {
			return fmt.Errorf("创建SMTP客户端失败: %v", err)
		}
		defer client.Quit()
		
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP认证失败: %v", err)
		}
		
		if err = client.Mail(mailConfig.From); err != nil {
			return fmt.Errorf("设置发件人失败: %v", err)
		}
		
		// 添加所有收件人（To、CC、BCC）
		recipients := append(message.To, message.CC...)
		recipients = append(recipients, message.BCC...)
		for _, recipient := range recipients {
			if err = client.Rcpt(recipient); err != nil {
				return fmt.Errorf("添加收件人失败: %v", err)
			}
		}
		
		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("获取数据写入器失败: %v", err)
		}
		defer w.Close()
		
		_, err = w.Write([]byte(content.String()))
		if err != nil {
			return fmt.Errorf("写入邮件内容失败: %v", err)
		}
	} else {
		// 普通连接
		err = smtp.SendMail(addr, auth, mailConfig.From, 
			append(append(message.To, message.CC...), message.BCC...), 
			[]byte(content.String()))
	}
	
	if err != nil {
		return fmt.Errorf("发送邮件失败: %v", err)
	}
	
	log.Printf("邮件已成功发送至: %s", strings.Join(message.To, ","))
	return nil
}

// SendReportMail 发送报告邮件
func SendReportMail(mailConfig *MailConfig, reportType, content string, recipients []string) error {
	if mailConfig == nil {
		return fmt.Errorf("邮件配置不能为空")
	}
	if len(recipients) == 0 {
		return fmt.Errorf("收件人不能为空")
	}
	
	subject := fmt.Sprintf("【%s】质量报告 - %s", reportType, getCurrentTime())
	
	message := &MailMessage{
		To:      recipients,
		Subject: subject,
		Body:    content,
		IsHTML:  true,
	}
	
	return SendMail(mailConfig, message)
}

// SendAlertMail 发送告警邮件
func SendAlertMail(mailConfig *MailConfig, alertType, message string, recipients []string) error {
	if mailConfig == nil {
		return fmt.Errorf("邮件配置不能为空")
	}
	if len(recipients) == 0 {
		return fmt.Errorf("收件人不能为空")
	}
	
	subject := fmt.Sprintf("【%s】告警通知 - %s", alertType, getCurrentTime())
	
	alertContent := fmt.Sprintf(`
	<html>
	<head>
		<style>
			body { font-family: Arial, sans-serif; }
			.alert { background-color: #f8d7da; border: 1px solid #f5c6cb; color: #721c24; padding: 15px; margin-bottom: 20px; border-radius: 4px; }
			.header { color: #721c24; font-size: 18px; font-weight: bold; }
			.content { margin-top: 15px; }
		</style>
	</head>
	<body>
		<div class="alert">
			<div class="header">告警类型: %s</div>
			<div class="content">
				<p>%s</p>
			</div>
		</div>
		<div>
			<p>请及时处理该告警信息。</p>
		</div>
	</body>
	</html>
	`, alertType, message)
	
	alertMessage := &MailMessage{
		To:      recipients,
		Subject: subject,
		Body:    alertContent,
		IsHTML:  true,
	}
	
	return SendMail(mailConfig, alertMessage)
}

// getCurrentTime 获取当前时间字符串
func getCurrentTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}