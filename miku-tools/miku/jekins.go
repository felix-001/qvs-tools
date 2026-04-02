package miku

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"mikutool/config"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// 配置结构体
type Config struct {
	BaseURL    string            `yaml:"base_url"`
	Username   string            `yaml:"username"`
	Password   string            `yaml:"password"`
	OtpSecret  string            `yaml:"otp_secret"`
	OtpURL     string            `yaml:"otp_url"`
	ApiToken   string            `yaml:"api_token"`
	TestJob    string            `yaml:"test_job"`
	ProdJob    string            `yaml:"prod_job"`
	ProdParams map[string]string `yaml:"prod_params"`
}

// 构建信息结构体
type BuildInfo struct {
	Number int    `json:"number"`
	URL    string `json:"url"`
	Result string `json:"result"`
}

// 队列项目结构体
type QueueItem struct {
	Executable *BuildInfo `json:"executable"`
	Why        string     `json:"why"`
}

// 参数定义结构体
type ParameterDefinition struct {
	Name                  string `json:"name"`
	Description           string `json:"description"`
	DefaultParameterValue struct {
		Value interface{} `json:"value"`
	} `json:"defaultParameterValue"`
}

// Jenkins客户端
type JenkinsClient struct {
	config        *Config
	client        *http.Client
	crumbField    string
	crumb         string
	jar           http.CookieJar
	mutex         sync.Mutex
	baseURLParsed *url.URL
}

// 命令行参数
type CLIArgs struct {
	Command      string
	PRs          []string
	Bins         []string
	Branch       string
	Service      string
	ListServices bool
	JobPath      string
	BuildNumber  int
}

// TOTP实现
func generateTOTP(secret string, digits int, period int) (string, error) {
	// 从URL中提取密钥
	if strings.HasPrefix(secret, "otpauth://") {
		re := regexp.MustCompile(`[?&]secret=([A-Z2-7a-z]+)`)
		matches := re.FindStringSubmatch(secret)
		if len(matches) < 2 {
			return "", fmt.Errorf("cannot extract secret from otpauth URL")
		}
		secret = matches[1]
	}

	// 清理base32密钥
	secret = strings.ToUpper(strings.TrimSpace(secret))

	// 添加填充
	if len(secret)%8 != 0 {
		secret += strings.Repeat("=", 8-len(secret)%8)
	}

	// 解码base32
	key, err := base32.StdEncoding.DecodeString(secret)
	if err != nil {
		return "", fmt.Errorf("failed to decode base32 secret: %w", err)
	}

	// 计算计数器
	counter := uint64(time.Now().Unix()) / uint64(period)

	// 转换为8字节大端序
	msg := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		msg[i] = byte(counter)
		counter >>= 8
	}

	// 计算HMAC-SHA1
	mac := hmac.New(sha1.New, key)
	mac.Write(msg)
	hash := mac.Sum(nil)

	// 动态截取偏移
	offset := hash[19] & 0x0F
	code := ((int(hash[offset]) & 0x7F) << 24) |
		((int(hash[offset+1]) & 0xFF) << 16) |
		((int(hash[offset+2]) & 0xFF) << 8) |
		(int(hash[offset+3]) & 0xFF)

	// 取模得到6位数
	code = code % 1000000
	return fmt.Sprintf("%0*d", digits, code), nil
}

// 加载配置文件
func loadConfig() (*Config, error) {
	searchPaths := []string{
		filepath.Join(os.Getenv("HOME"), ".claude", "skills", "jenkins-credentials.yaml"),
		filepath.Join(".", "jenkins-credentials.yaml"),
		filepath.Join("..", "jenkins-credentials.yaml"),
	}

	// 如果$HOME未设置，使用当前用户目录
	if os.Getenv("HOME") == "" {
		if u, err := user.Current(); err == nil {
			searchPaths[0] = filepath.Join(u.HomeDir, ".claude", "skills", "jenkins-credentials.yaml")
		}
	}

	var configPath string
	for _, path := range searchPaths {
		if _, err := os.Stat(path); err == nil {
			configPath = path
			break
		}
	}

	if configPath == "" {
		return nil, errors.New(`no configuration file found
Please create jenkins-credentials.yaml with your credentials:
  Global: ~/.claude/skills/jenkins-credentials.yaml
  Local:  .claude/skills/jenkins/jenkins-credentials.yaml`)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &config, nil
}

// 创建Jenkins客户端
func NewJenkinsClient(cfg *Config) (*JenkinsClient, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	parsedURL, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		Jar:     jar,
	}

	return &JenkinsClient{
		config:        cfg,
		client:        client,
		jar:           jar,
		baseURLParsed: parsedURL,
	}, nil
}

// 登录到Jenkins
func (jc *JenkinsClient) Login() error {
	if jc.config.ApiToken != "" {
		fmt.Printf("[jenkins] Using API token auth (user=%s)\n", jc.config.Username)
		return nil
	}

	fmt.Println("[jenkins] Logging in with LDAP credentials...")

	var otpCode string
	if jc.config.OtpSecret != "" || jc.config.OtpURL != "" {
		secret := jc.config.OtpSecret
		if secret == "" {
			secret = jc.config.OtpURL
		}
		code, err := generateTOTP(secret, 6, 30)
		if err != nil {
			return fmt.Errorf("failed to generate OTP: %w", err)
		}
		otpCode = code
		fmt.Printf("[jenkins] Generated OTP: %s\n", otpCode)
	}

	loginURL := jc.buildURL("/j_acegi_security_check")
	formData := url.Values{
		"j_username": {jc.config.Username},
		"j_password": {jc.config.Password},
		"j_otp":      {otpCode},
		"from":       {"/"},
		"Submit":     {"Sign in"},
	}

	req, err := http.NewRequest("POST", loginURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := jc.client.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 302 {
		return fmt.Errorf("login failed with status: %d", resp.StatusCode)
	}

	fmt.Println("[jenkins] Login successful")
	return jc.fetchCrumb()
}

// 获取CSRF令牌
func (jc *JenkinsClient) fetchCrumb() error {
	crumbURL := jc.buildURL("/crumbIssuer/api/json")

	req, err := http.NewRequest("GET", crumbURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create crumb request: %w", err)
	}

	if jc.config.ApiToken != "" {
		auth := base64.StdEncoding.EncodeToString(
			[]byte(fmt.Sprintf("%s:%s", jc.config.Username, jc.config.ApiToken)),
		)
		req.Header.Set("Authorization", "Basic "+auth)
	}

	resp, err := jc.client.Do(req)
	if err != nil {
		return fmt.Errorf("crumb request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		var result struct {
			CrumbRequestField string `json:"crumbRequestField"`
			Crumb             string `json:"crumb"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("failed to decode crumb response: %w", err)
		}

		jc.crumbField = result.CrumbRequestField
		jc.crumb = result.Crumb
		fmt.Println("[jenkins] CSRF crumb obtained")
	} else {
		fmt.Printf("[jenkins] Crumb not available (status=%d), proceeding without it\n", resp.StatusCode)
	}

	return nil
}

// 构建请求头
func (jc *JenkinsClient) buildHeaders() http.Header {
	headers := make(http.Header)

	if jc.config.ApiToken != "" {
		auth := base64.StdEncoding.EncodeToString(
			[]byte(fmt.Sprintf("%s:%s", jc.config.Username, jc.config.ApiToken)),
		)
		headers.Set("Authorization", "Basic "+auth)
	}

	if jc.crumb != "" && jc.crumbField != "" {
		headers.Set(jc.crumbField, jc.crumb)
	}

	return headers
}

// 构建URL
func (jc *JenkinsClient) buildURL(path string) string {
	return strings.TrimSuffix(jc.config.BaseURL, "/") + "/" + strings.TrimPrefix(path, "/")
}

// 触发构建
func (jc *JenkinsClient) TriggerBuild(jobPath string, params map[string]string) (int, error) {
	var buildURL string
	if len(params) > 0 {
		buildURL = jc.buildURL("/" + strings.TrimPrefix(jobPath, "/") + "/buildWithParameters")
	} else {
		buildURL = jc.buildURL("/" + strings.TrimPrefix(jobPath, "/") + "/build")
	}

	var reqBody io.Reader
	if len(params) > 0 {
		formData := url.Values{}
		for k, v := range params {
			formData.Set(k, v)
		}
		log.Println("params:", formData.Encode())
		reqBody = strings.NewReader(formData.Encode())
	}

	req, err := http.NewRequest("POST", buildURL, reqBody)
	if err != nil {
		return -1, fmt.Errorf("failed to create build request: %w", err)
	}

	headers := jc.buildHeaders()
	if len(params) > 0 {
		headers.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	log.Printf("headers: %v\n", headers)
	for k, v := range headers {
		req.Header[k] = v
	}

	resp, err := jc.client.Do(req)
	if err != nil {
		return -1, fmt.Errorf("build request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return -1, fmt.Errorf("trigger failed (status=%d): %s", resp.StatusCode, string(body[:min(500, len(body))]))
	}

	// 从Location头中提取队列ID
	location := resp.Header.Get("Location")
	if location == "" {
		fmt.Println("[jenkins] Build queued (no queue ID in response)")
		return -1, nil
	}

	re := regexp.MustCompile(`/queue/item/(\d+)/`)
	matches := re.FindStringSubmatch(location)
	if len(matches) < 2 {
		fmt.Println("[jenkins] Build queued (could not parse queue ID)")
		return -1, nil
	}

	queueID, _ := strconv.Atoi(matches[1])
	fmt.Printf("[jenkins] Build queued (queue_id=%d)\n", queueID)
	fmt.Printf("[jenkins] Queue URL: %s/queue/item/%d/\n", jc.config.BaseURL, queueID)
	return queueID, nil
}

// 获取队列项目
func (jc *JenkinsClient) GetQueueItem(queueID int) (*BuildInfo, error) {
	queueURL := jc.buildURL(fmt.Sprintf("/queue/item/%d/api/json", queueID))

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	for attempt := 0; attempt < 30; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "GET", queueURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create queue request: %w", err)
		}

		headers := jc.buildHeaders()
		for k, v := range headers {
			req.Header[k] = v
		}

		resp, err := jc.client.Do(req)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		if resp.StatusCode == 200 {
			var item QueueItem
			if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
				resp.Body.Close()
				time.Sleep(3 * time.Second)
				continue
			}
			resp.Body.Close()

			if item.Executable != nil {
				return item.Executable, nil
			}

			why := item.Why
			if why == "" {
				why = "queued"
			}
			fmt.Printf("[jenkins] Waiting for build to start... (%s)\n", why)
		} else {
			resp.Body.Close()
		}

		time.Sleep(3 * time.Second)
	}

	return nil, errors.New("timed out waiting for build to start")
}

// 获取构建状态
func (jc *JenkinsClient) GetBuildStatus(jobPath string, buildNumber int) (map[string]interface{}, error) {
	buildURL := jc.buildURL(fmt.Sprintf("/%s/%d/api/json", strings.TrimPrefix(jobPath, "/"), buildNumber))

	log.Println(buildURL)
	req, err := http.NewRequest("GET", buildURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create status request: %w", err)
	}

	headers := jc.buildHeaders()
	for k, v := range headers {
		req.Header[k] = v
	}

	resp, err := jc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("status request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get build status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode status response: %w", err)
	}

	return result, nil
}

// 列出任务参数
func (jc *JenkinsClient) ListJobParams(jobPath string) ([]ParameterDefinition, error) {
	jobURL := jc.buildURL(fmt.Sprintf("/%s/api/json?tree=property[parameterDefinitions[name,description,defaultParameterValue[value]]]",
		strings.TrimPrefix(jobPath, "/")))

	req, err := http.NewRequest("GET", jobURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create params request: %w", err)
	}

	headers := jc.buildHeaders()
	for k, v := range headers {
		req.Header[k] = v
	}

	resp, err := jc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("params request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get job params: %d", resp.StatusCode)
	}

	var result struct {
		Property []struct {
			ParameterDefinitions []ParameterDefinition `json:"parameterDefinitions"`
		} `json:"property"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode params response: %w", err)
	}

	var params []ParameterDefinition
	for _, prop := range result.Property {
		params = append(params, prop.ParameterDefinitions...)
	}

	return params, nil
}

// 解析命令行参数
func parseArgs(cmdArgs []string) (*CLIArgs, error) {
	args := &CLIArgs{}

	if len(cmdArgs) < 1 {
		return nil, errors.New("no command specified")
	}

	args.Command = cmdArgs[0]

	switch args.Command {
	case "test":
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		var prs, bins string
		fs.StringVar(&prs, "pr", "", "PR numbers separated by space")
		fs.StringVar(&bins, "bin", "", "Service names separated by space")
		fs.Parse(cmdArgs[1:])

		if prs != "" {
			args.PRs = strings.Fields(prs)
		}
		if bins != "" {
			args.Bins = strings.Fields(bins)
		}

	case "prod":
		fs := flag.NewFlagSet("prod", flag.ContinueOnError)
		fs.StringVar(&args.Branch, "branch", "", "Branch name")
		fs.StringVar(&args.Service, "service", "", "Service name")
		fs.BoolVar(&args.ListServices, "list-services", false, "List available services")
		fs.Parse(cmdArgs[1:])

	case "status":
		fs := flag.NewFlagSet("status", flag.ContinueOnError)
		var job, build string
		fs.StringVar(&job, "job", "", "Job name")
		fs.StringVar(&build, "build", "", "Build number")
		fs.Parse(cmdArgs[1:])

		args.JobPath = job
		if build != "" {
			if n, err := strconv.Atoi(build); err == nil {
				args.BuildNumber = n
			}
		}

	default:
		return nil, fmt.Errorf("unknown command: %s", args.Command)
	}

	return args, nil
}

// 测试命令
func CmdTest(client *JenkinsClient, args *CLIArgs, config *Config) (int, error) {
	if /*len(args.PRs) == 0 ||*/ len(args.Bins) == 0 {
		return 0, errors.New("--pr <pr...> and --bin <service...> are required\nExample: test --pr 3103 3067 --bin agent lived")
	}

	jobName := config.TestJob
	if jobName == "" {
		jobName = "mikud-dev-jfcs-deploy"
	}
	jobPath := "job/" + jobName

	params := map[string]string{
		"env": "dev",
		"PR":  strings.Join(args.PRs, " "),
		"BIN": strings.Join(args.Bins, " "),
	}

	fmt.Printf("[jenkins] Triggering test deploy: PR=%s BIN=%s\n", params["PR"], params["BIN"])

	queueID, err := client.TriggerBuild(jobPath, params)
	if err != nil {
		return 0, fmt.Errorf("failed to trigger build: %w", err)
	}
	buildNum := 0
	if queueID > 0 {
		fmt.Println("[jenkins] Waiting for build to start...")
		buildInfo, err := client.GetQueueItem(queueID)
		if err != nil {
			return 0, fmt.Errorf("failed to get queue item: %w", err)
		}

		if buildInfo != nil {
			buildNum = buildInfo.Number
			fmt.Printf("[jenkins] Build #%d started: %s\n", buildNum, buildInfo.URL)
		}
	}

	return buildNum, nil
}

// 生产命令
func cmdProd(client *JenkinsClient, args *CLIArgs, config *Config) (int, error) {
	jobName := config.ProdJob
	if jobName == "" {
		jobName = "mikud-live-module-pipeline"
	}
	jobPath := "view/Mikud/job/" + jobName

	if err := client.Login(); err != nil {
		log.Printf("login failed: %v\n", err)
		return 0, fmt.Errorf("login failed: %w", err)
	}

	if args.ListServices {
		log.Printf("[jenkins] Fetching parameters for %s...\n", jobName)
		params, err := client.ListJobParams(jobPath)
		if err != nil {
			return 0, fmt.Errorf("failed to list job params: %w", err)
		}

		log.Println("[jenkins] Available parameters:")
		for _, p := range params {
			defaultStr := ""
			if p.DefaultParameterValue.Value != nil {
				defaultStr = fmt.Sprintf(" (default: %v)", p.DefaultParameterValue.Value)
			}
			log.Printf("  %s: %s%s\n", p.Name, p.Description, defaultStr)
		}
		return 0, nil
	}

	if args.Branch == "" || args.Service == "" {
		return 0, errors.New("--branch and --service are required for production deploy\nUse --list-services to see available service names")
	}

	params := map[string]string{
		"BRANCH": args.Branch,
		//"SERVICE": args.Service,
		"BIN": args.Service,
	}

	// 合并额外参数
	if config.ProdParams != nil {
		for k, v := range config.ProdParams {
			params[k] = v
		}
	}

	log.Printf("[jenkins] Triggering production pipeline: job=%s, branch=%s, service=%s\n",
		jobName, args.Branch, args.Service)
	log.Printf("[jenkins] Full URL: %s/view/Mikud/job/%s/\n", config.BaseURL, jobName)

	queueID, err := client.TriggerBuild(jobPath, params)
	if err != nil {
		log.Printf("failed to trigger build: %v\n", err)
		return 0, fmt.Errorf("failed to trigger build: %w", err)
	}

	buildNum := 0
	if queueID > 0 {
		fmt.Println("[jenkins] Waiting for build to start...")
		buildInfo, err := client.GetQueueItem(queueID)
		if err != nil {
			log.Printf("failed to get queue item: %v\n", err)
			return 0, fmt.Errorf("failed to get queue item: %w", err)
		}

		if buildInfo != nil {
			log.Printf("Build #%d started: %s\n", buildInfo.Number, buildInfo.URL)
			buildNum = buildInfo.Number
		}
	}

	return buildNum, nil
}

type Status struct {
	InProgress  bool
	PackageName string
}

// 状态命令
func cmdStatus(client *JenkinsClient, args *CLIArgs) (*Status, error) {
	if args.JobPath == "" || args.BuildNumber == 0 {
		return nil, errors.New("--job and --build are required")
	}

	jobPath := "job/" + args.JobPath

	if err := client.Login(); err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	info, err := client.GetBuildStatus(jobPath, args.BuildNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get build status: %w", err)
	}
	status := &Status{}

	if info != nil {
		result := "IN_PROGRESS"
		status.InProgress = true
		if r, ok := info["result"].(string); ok {
			if r == "SUCCESS" {
				status.InProgress = false
			}
			result = r
		}

		duration := 0.0
		if d, ok := info["duration"].(float64); ok {
			duration = d / 1000
		}

		url := ""
		if u, ok := info["url"].(string); ok {
			url = u
		}

		fmt.Printf("[jenkins] Build #%d: %s (%.0fs)\n", args.BuildNumber, result, duration)
		fmt.Printf("[jenkins] URL: %s\n", url)

		if actions, ok := info["actions"].([]interface{}); ok {
			for _, action := range actions {
				if a, ok := action.(map[string]interface{}); ok {
					if params, ok := a["parameters"].([]interface{}); ok && len(params) > 0 {
						fmt.Println("[jenkins] Build parameters:")
						for _, param := range params {
							if p, ok := param.(map[string]interface{}); ok {
								name := p["name"].(string)
								value := ""
								if v, ok := p["value"]; ok {
									value = fmt.Sprintf("%v", v)
								}
								if name == "PACKAGE_NAME" {
									status.PackageName = value
								}
								fmt.Printf("  %s = %s\n", name, value)
							}
						}
					}
				}
			}
		}
	} else {
		fmt.Println("[jenkins] Build not found")
	}

	return status, nil
}

func Run(cmdArgs []string) error {
	if len(cmdArgs) < 1 {
		fmt.Print(`Jenkins Trigger Tool - Go version

Usage:
  # Trigger test environment (mikud-dev-jfcs-deploy)
  jenkins-trigger test --pr "3103 3067" --bin "agent lived"

  # Trigger production pipeline (mikud-live-module-pipeline)
  jenkins-trigger prod --branch <branch> --service <service>

  # List available production services
  jenkins-trigger prod --list-services

  # Check job status
  jenkins-trigger status --job <job_name> --build <build_number>
`)
		return nil
	}

	// 加载配置
	config, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 解析命令行参数
	args, err := parseArgs(cmdArgs)
	if err != nil {
		return fmt.Errorf("failed to parse args: %w", err)
	}

	// 创建客户端
	client, err := NewJenkinsClient(config)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// 执行命令
	var cmdErr error
	switch args.Command {
	case "test":
		_, cmdErr = CmdTest(client, args, config)
	case "prod":
		_, cmdErr = cmdProd(client, args, config)
	case "status":
		cmdStatus(client, args)
	}

	return cmdErr
}

func RunTest(conf *config.Config) {
	cfg := &Config{
		BaseURL: "https://jenkins.qiniu.io",
		TestJob: "mikud-dev-jfcs-deploy",
	}
	client, err := NewJenkinsClient(cfg)
	if err != nil {
		fmt.Printf("failed to create client: %v\n", err)
		return
	}
	args := &CLIArgs{
		PRs:  []string{},
		Bins: []string{"lived"},
	}
	buildNum, err := CmdTest(client, args, cfg)
	if err != nil {
		fmt.Printf("failed to trigger test: %v\n", err)
		return
	}
	fmt.Printf("build number: %d\n", buildNum)
}

func GetStatus(conf *config.Config) {
	cfg := &Config{
		BaseURL: "https://jenkins.qiniu.io",
		TestJob: "mikud-dev-jfcs-deploy",
	}
	client, err := NewJenkinsClient(cfg)
	if err != nil {
		fmt.Printf("failed to create client: %v\n", err)
		return
	}
	jobName := cfg.TestJob
	if jobName == "" {
		jobName = "mikud-dev-jfcs-deploy"
	}
	args := &CLIArgs{
		JobPath:     jobName,
		BuildNumber: 1047,
	}
	cmdStatus(client, args)
}

func Build(conf *config.Config) {
	if conf.User == "iqiyi" {
		conf.User = "liyuanquan"
	}
	if conf.ApiKey == "" {
		log.Println("api key is empty")
		return
	}
	cfg := &Config{
		BaseURL:  "https://jenkins.qiniu.io",
		Username: conf.User,
		ApiToken: conf.ApiKey,
		TestJob:  "mikud-dev-jfcs-deploy",
	}
	log.Println("api token:", conf.ApiKey)
	log.Println("bin:", conf.Bin)
	client, err := NewJenkinsClient(cfg)
	if err != nil {
		log.Printf("failed to create client: %v\n", err)
		return
	}
	args := &CLIArgs{
		Service: conf.Bin,
		Bins:    []string{conf.Bin},
		Branch:  "main",
	}
	buildNum := 0
	if conf.Env == "online" {
		buildNum, err = cmdProd(client, args, cfg)
		if err != nil {
			log.Printf("failed to trigger prod: %v\n", err)
			return
		}
	} else {
		buildNum, err = CmdTest(client, args, cfg)
		if err != nil {
			log.Printf("failed to trigger test: %v\n", err)
			return
		}
	}
	log.Println("build number:", buildNum)
}
