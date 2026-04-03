package miku

import (
	"fmt"
	"log"
	"mikutool/config"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func GetOldPackageName(conf *config.Config) (string, error) {
	// 读取CSV文件内容
	data, err := os.ReadFile(conf.Path)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %w", err)
	}

	// 将内容按行分割
	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		return "", fmt.Errorf("文件内容为空")
	}

	// 解析第一行，获取旧的包名
	firstLine := strings.TrimSpace(lines[1])
	if firstLine == "" {
		return "", fmt.Errorf("第一行为空")
	}

	// 按逗号分割，获取第3个字段（索引为2）
	fields := strings.Split(firstLine, ",")
	if len(fields) < 3 {
		return "", fmt.Errorf("第一行字段不足3个")
	}

	oldPackageName := strings.TrimSpace(fields[2])
	return oldPackageName, nil
}

func UpdatePackageName(conf *config.Config, packageName string) error {
	if conf.Path == "" {
		conf.Path = fmt.Sprintf("/Users/liyuanquan/workspace/deploy/floy/miku-%s/miku-%s.csv", conf.Bin, conf.Bin)
	}

	oldPackageName, err := GetOldPackageName(conf)
	if err != nil {
		log.Printf("获取旧包名失败: %v\n", err)
		return fmt.Errorf("获取旧包名失败: %w", err)
	}

	if oldPackageName == "" {
		log.Printf("旧包名为空\n")
		return fmt.Errorf("旧包名为空")
	}

	log.Printf("替换包名: %s -> %s\n", oldPackageName, packageName)

	data, err := os.ReadFile(conf.Path)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	newContent := strings.ReplaceAll(string(data), oldPackageName, packageName)

	err = os.WriteFile(conf.Path, []byte(newContent), 0644)
	if err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	fmt.Printf("文件 %s 中的包名已全部替换\n", conf.Path)
	return nil
}

func ResetGitLab() {
	path := "/Users/liyuanquan/workspace/deploy/"

	// 定义要执行的 git 命令
	commands := []string{
		"git fetch upstream",
		"git reset --hard upstream/master",
		"git status",
	}

	// 依次执行每个命令
	for _, cmdStr := range commands {
		log.Printf("执行命令：%s\n", cmdStr)

		// 创建命令
		cmd := exec.Command("bash", "-c", cmdStr)

		// 设置工作目录
		cmd.Dir = path

		// 执行命令并获取输出
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("命令执行失败：%v\n", err)
		}

		// 输出命令执行结果
		fmt.Printf("命令：%s\n", cmdStr)
		fmt.Printf("输出:\n%s\n", string(output))
		if err != nil {
			fmt.Printf("错误：%v\n\n", err)
		} else {
			fmt.Println()
		}
	}
}

func OpenConfEdit(conf *config.Config) {
	path := fmt.Sprintf("/Users/liyuanquan/workspace/deploy/floy/miku-%s/env_common/_package/%s.json", conf.Bin, conf.Bin)
	log.Printf("打开配置文件：%s\n", path)

	// 使用 open 命令（macOS）打开文件
	cmd := exec.Command("open", path)

	err := cmd.Start()
	if err != nil {
		log.Printf("打开文件失败：%v\n", err)
		return
	}
	log.Printf("正在打开文件：%s\n", path)
}

func GitlabPR(conf *config.Config, prTitle string) {
	// 构建工作目录路径
	workDir := fmt.Sprintf("/Users/liyuanquan/workspace/deploy/floy/miku-%s", conf.Bin)

	// 定义要执行的 git 命令
	commands := []string{
		"git add .",
		fmt.Sprintf("git commit -m \"%s\"", prTitle),
		//"git push -f",
	}

	// 依次执行每个命令
	for _, cmdStr := range commands {
		log.Printf("执行命令：%s\n", cmdStr)

		// 创建命令
		cmd := exec.Command("bash", "-c", cmdStr)

		// 设置工作目录
		cmd.Dir = workDir

		// 执行命令并获取输出
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("命令执行失败：%v\n", err)
			fmt.Printf("命令：%s\n", cmdStr)
			fmt.Printf("输出:\n%s\n", string(output))
			fmt.Printf("错误：%v\n\n", err)
			return
		}

		// 输出命令执行结果
		fmt.Printf("命令：%s\n", cmdStr)
		fmt.Printf("输出:\n%s\n\n", string(output))
	}

	// 解析 push 命令的输出，提取 MR URL
	lastOutput := getLastCommandOutput(workDir, "git push -f")
	log.Println("lastOutput:", lastOutput)
	mrURL := extractGitLabMRURL(lastOutput)

	if mrURL != "" {
		log.Printf("找到合并请求 URL: %s\n", mrURL)

		// 使用 open 命令打开 URL
		cmd := exec.Command("open", mrURL)
		err := cmd.Start()
		if err != nil {
			log.Printf("打开 URL 失败：%v\n", err)
			return
		}
		log.Printf("正在打开合并请求页面：%s\n", mrURL)
	} else {
		log.Printf("未找到合并请求 URL\n")
	}
}

// getLastCommandOutput 获取最后一次 git push 的输出
func getLastCommandOutput(workDir string, cmdStr string) string {
	cmd := exec.Command("bash", "-c", cmdStr)
	cmd.Dir = workDir
	output, _ := cmd.CombinedOutput()
	return string(output)
}

// extractGitLabMRURL 从 git push 输出中提取 GitLab 合并请求 URL
func extractGitLabMRURL(output string) string {
	// 查找包含 "To create a merge request for" 的行
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if strings.Contains(line, "To create a merge request for") {
			// 下一行应该包含 URL
			if i+1 < len(lines) {
				urlLine := strings.TrimSpace(lines[i+1])
				// 使用正则表达式提取 URL
				re := regexp.MustCompile(`https?://[^\s]+`)
				match := re.FindString(urlLine)
				if match != "" {
					// 移除可能的反引号
					match = strings.Trim(match, "`")
					return match
				}
			}
		}
	}
	return ""
}
