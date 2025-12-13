package miku

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"mikutool/config"
	"net/http"
	"net/url"
	"strings"
)

func HlsPlay(conf *config.Config) {
	// 创建自定义HTTP客户端，用于捕获重定向
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// 打印重定向信息
			log.Printf("Redirecting to: %s (from %s)", req.URL.String(), via[0].URL.String())
			return nil // 允许重定向
		},
	}

	// Make initial HTTP request
	resp, err := client.Get(conf.Url)
	if err != nil {
		log.Printf("Error fetching initial URL: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v\n", err)
		return
	}

	// Log the initial response status
	log.Printf("Initial request to %s returned status code: %d", conf.Url, resp.StatusCode)
	log.Println("raw m3u8:", string(body))

	// Process each line
	scanner := bufio.NewScanner(strings.NewReader(string(body)))
	for scanner.Scan() {
		line := scanner.Text()
		if isURL(line) {
			// Replace domain
			modifiedURL, err := replaceDomain(line, conf.Domain)
			if err != nil {
				log.Printf("Error modifying URL: %v\n", err)
				continue
			}

			// Make request to modified URL
			log.Println("url: ", modifiedURL)
			resp, err := client.Get(modifiedURL)
			if err != nil {
				log.Printf("Error fetching modified URL %s: %v\n", modifiedURL, err)
				continue
			}
			defer resp.Body.Close()

			// 检查是否有重定向链
			if resp.Request.URL.String() != modifiedURL {
				log.Printf("Final URL after redirects: %s (original: %s)", resp.Request.URL.String(), modifiedURL)
			}

			fmt.Printf("URL: %s, Status Code: %d\n", modifiedURL, resp.StatusCode)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error scanning content: %v\n", err)
	}
}

func isURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func replaceDomain(originalURL, newDomain string) (string, error) {
	u, err := url.Parse(originalURL)
	if err != nil {
		return "", err
	}

	// Split host into parts
	parts := strings.Split(u.Host, ":")
	u.Host = newDomain
	if len(parts) > 1 {
		u.Host += ":" + parts[1] // preserve port if exists
	}

	return u.String(), nil
}
