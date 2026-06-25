package miku

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

const (
	defaultLinkcloudLoginURL = "https://sso.qiniu.io/?client_id=linkcloud-admin&redirect=" +
		"https%3A%2F%2Flinkcloud-admin.qiniu.io%2Fuser%2Fsign-in%2Fsso%3Fredirect%3D" +
		"%252Fnode-operation%252Fresource%252Fnode%253FqueryTimeType%253Dservice%2526" +
		"filterRepelled%253Dtrue%2526filterDestroyed%253Dfalse%2526sortKey%253Dservice%2526" +
		"sortType%253Ddesc"
	defaultLinkcloudTargetURL = "https://linkcloud-admin.qiniu.io/node-operation/resource/node" +
		"?queryTimeType=service&filterRepelled=true&filterDestroyed=false&sortKey=service&sortType=desc"
)

func getLinkcloudAuth(s wsExecSettings) (string, error) {
	username, err := readSecretFile(s.userFile)
	if err != nil {
		return "", fmt.Errorf("read user file: %w", err)
	}
	password, err := readSecretFile(s.passFile)
	if err != nil {
		return "", fmt.Errorf("read pass file: %w", err)
	}
	totp, err := getTotpCode(s.totpBin, s.totpProfile)
	if err != nil {
		return "", fmt.Errorf("get totp: %w", err)
	}

	var (
		mu           sync.Mutex
		capturedAuth string
	)

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.NoSandbox,
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, timeoutCancel := context.WithTimeout(ctx, 2*time.Minute)
	defer timeoutCancel()

	chromedp.ListenTarget(ctx, func(ev interface{}) {
		req, ok := ev.(*network.EventRequestWillBeSent)
		if !ok {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if capturedAuth != "" {
			return
		}
		for k, v := range req.Request.Headers {
			if !strings.EqualFold(k, "authorization") {
				continue
			}
			auth, ok := v.(string)
			if ok && len(auth) > 10 {
				capturedAuth = auth
			}
		}
	})

	tasks := chromedp.Tasks{
		network.Enable(),
		chromedp.Navigate(s.loginURL),
		chromedp.WaitVisible(`//input`, chromedp.BySearch),
		fillInputByName("LDAP 用户名", username),
		fillInputByName("LDAP 登录密码", password),
		fillInputByName("请输入TOTP验证码", totp),
		chromedp.Click(`//button[contains(normalize-space(.), "立即登录")]`, chromedp.BySearch),
		chromedp.Sleep(3 * time.Second),
		chromedp.Navigate(s.targetURL),
		chromedp.Sleep(5 * time.Second),
	}
	if err := chromedp.Run(ctx, tasks); err != nil {
		return "", fmt.Errorf("linkcloud SSO login: %w", err)
	}

	mu.Lock()
	auth := capturedAuth
	mu.Unlock()
	if auth == "" {
		return "", errors.New("did not capture Authorization header from any request")
	}
	return auth, nil
}

func fillInputByName(name, value string) chromedp.Action {
	xpath := fmt.Sprintf(
		`(//label[contains(normalize-space(.), %q)]/following::input[1] | //input[@aria-label=%q] | //input[@placeholder=%q])[1]`,
		name, name, name,
	)
	return chromedp.SetValue(xpath, value, chromedp.BySearch)
}

func readSecretFile(path string) (string, error) {
	data, err := os.ReadFile(expandHome(path))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func getTotpCode(bin, profile string) (string, error) {
	out, err := exec.Command(bin, profile).Output()
	if err != nil {
		return "", fmt.Errorf("run %s %s: %w", bin, profile, err)
	}
	code := strings.TrimSpace(string(out))
	if code == "" {
		return "", errors.New("totp command produced empty output")
	}
	return code, nil
}

func expandHome(path string) string {
	if path == "" {
		return path
	}
	if path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		if len(path) == 1 || path[1] == '/' {
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return path
}

func defaultTotpBin() string {
	home, _ := os.UserHomeDir()
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Documents/QiNiuWork/code_base/totp")
	}
	return filepath.Join(home, "totp")
}

func defaultTotpProfile() string {
	if runtime.GOOS == "darwin" {
		return "linkcloud"
	}
	return "Linkcloud"
}
