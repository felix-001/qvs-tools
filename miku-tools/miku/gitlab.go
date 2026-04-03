package miku

import (
	"fmt"
	"log"
	"mikutool/config"
	"os"
	"os/exec"
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

func OenConfEdit(conf *config.Config) {
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
