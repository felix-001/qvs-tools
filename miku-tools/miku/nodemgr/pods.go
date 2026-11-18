package nodemgr

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"mikutool/config"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	edgecoreDBPath         = "/var/lib/kubeedge/edgecore.db"
	podsQsshUser           = "qboxserver" // 登录节点只用普通账号，禁止 root
	podsQueryTimeout       = 60 * time.Second
	podsDefaultConcurrency = 20
	podsStreamdName        = "streamd"
	podsErrOutputLimit     = 4000 // 失败时最多回显的远端输出字节数
)

// edgecorePodsScript 在节点上读取 edgecore.db 里的 pod 记录。
// 节点上不保证有 sqlite3 工具，因此优先使用 python3/python 的 sqlite3 模块，
// 再退化为 sqlite3 命令行；输出格式固定为：
//
//	COUNT:<pod 数量>
//	POD:<pod 名>
//	ERR:<错误信息>
//
// 任何一步失败都必须把远端真实的错误信息带回来，便于定位。
const edgecorePodsScript = `DB="/var/lib/kubeedge/edgecore.db"
TMPDB=""
ERRF=""
MISSING=""
cleanup() {
  if [ -n "$TMPDB" ]; then rm -f "$TMPDB"; fi
  if [ -n "$ERRF" ]; then rm -f "$ERRF"; fi
}
trap cleanup EXIT
if [ ! -r "$DB" ]; then
  TMPDB="/tmp/.edgecore.$$.db"
  ERRF="/tmp/.edgecore.$$.err"
  if sudo -n cp "$DB" "$TMPDB" 2>"$ERRF"; then
    chmod 0644 "$TMPDB" 2>/dev/null
    DB="$TMPDB"
  else
    echo "ERR:db not readable by $(id -un 2>/dev/null): $(tr '\n' ' ' < "$ERRF" 2>/dev/null)"
    rm -f "$ERRF"
    ERRF=""
    exit 0
  fi
fi
for PY in python3 python; do
  if command -v "$PY" >/dev/null 2>&1; then
    if "$PY" -c 'import sqlite3' >/dev/null 2>&1; then
      "$PY" - "$DB" <<'PYEOF' 2>&1
import json, sqlite3, sys, traceback

def main():
    db = sys.argv[1]
    conn = sqlite3.connect("file:%s?mode=ro" % db, uri=True)
    rows = conn.execute("select key, value from meta where type = ?", ("pod",)).fetchall()
    names = set()
    for key, value in rows:
        name = ""
        obj = None
        if value:
            try:
                obj = json.loads(value)
            except Exception:
                obj = None
        if isinstance(obj, dict):
            md = obj.get("metadata")
            if isinstance(md, dict):
                name = md.get("name") or ""
            if not name:
                name = obj.get("name") or ""
        if not name and key:
            name = str(key).split("/")[-1]
        if name:
            names.add(name)
    sys.stdout.write("COUNT:%d\n" % len(names))
    for name in sorted(names):
        sys.stdout.write("POD:%s\n" % name)

try:
    main()
except Exception:
    sys.stdout.write("ERR:%s failed: %s\n" % (sys.executable, " | ".join(traceback.format_exc().splitlines())))
    sys.stdout.write("ERR:db path: %s\n" % sys.argv[1])
PYEOF
      exit 0
    fi
    MISSING="$MISSING; $PY($("$PY" -V 2>&1)) 缺少 sqlite3 模块"
  fi
done
if command -v sqlite3 >/dev/null 2>&1; then
  OUT=$(sqlite3 -readonly "$DB" "select key from meta where type = 'pod'" 2>&1) || {
    echo "ERR:sqlite3 failed: $(printf '%s' "$OUT" | tr '\n' ' ')"
    exit 0
  }
  echo "COUNT:$(printf '%s' "$OUT" | grep -c . 2>/dev/null)"
  printf '%s\n' "$OUT" | awk -F/ 'NF > 0 && $NF != "" {print "POD:" $NF}'
  exit 0
fi
echo "ERR:no usable sqlite reader on node ($(id -un 2>/dev/null))${MISSING}; no sqlite3 command"
`

// edgecorePodsHook 便于测试时替换远端执行逻辑
var edgecorePodsHook = queryEdgecorePodsRemote

// PrintStreamdDupPods 并发登录所有静态节点，查询 edgecore.db 中的 pod 列表，
// 若同一节点存在至少两个 pod 名包含 "streamd"，则打印节点 host 与 pod 名。
func (m *NodeMgr) PrintStreamdDupPods(conf *config.Config) {
	// qssh 无 TTY 会改走 WAN root@；script -c 只吃一个命令字符串。
	//out, execErr := exec.Command("script", "-q", "-e", "-c", "qssh qboxserver@jf22 ls /tmp", "/dev/null").CombinedOutput()
	//log.Println("logs: ", string(out), " err: ", execErr)
	//return
	type nodeTarget struct {
		host   string
		nodeId string
	}

	targets := make([]nodeTarget, 0, len(m.allNodesMap))
	for nodeId, node := range m.allNodesMap {
		if node == nil || node.IsDynamic {
			continue
		}
		// 登录节点必须使用 nodes 的 host 字段，而不是 map 的 key(id)
		host := strings.TrimSpace(node.HostName)
		if host == "" {
			log.Printf("[Pods] 静态节点缺少 host 字段，跳过: nodeId=%s", nodeId)
			continue
		}
		targets = append(targets, nodeTarget{host: host, nodeId: nodeId})
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].host < targets[j].host })

	total := len(targets)
	if total == 0 {
		log.Println("[Pods] 没有可用的静态节点")
		return
	}

	concurrency := podsDefaultConcurrency
	if conf != nil && conf.N > 0 {
		concurrency = conf.N
	}
	if concurrency > total {
		concurrency = total
	}
	loginUser := podsQsshUser
	log.Printf("[Pods] 静态节点总数: %d, 并发数: %d, 登录账号: %s, db: %s",
		total, concurrency, loginUser, edgecoreDBPath)

	pr := newPodsProgress(int64(total))
	jobs := make(chan nodeTarget)
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range jobs {
				pods, err := edgecorePodsHook(loginUser, t.host)
				if err != nil {
					pr.resolve(fmt.Sprintf("[失败] host=%s nodeId=%s err=%v", t.host, t.nodeId, err), false, true)
					continue
				}
				streamdPods := make([]string, 0, 2)
				for _, pod := range pods {
					if strings.Contains(strings.ToLower(pod), podsStreamdName) {
						streamdPods = append(streamdPods, pod)
					}
				}
				if len(streamdPods) >= 2 {
					pr.resolve(fmt.Sprintf("[命中] nodeId=%s host=%s pods=%v",
						t.nodeId, t.host, streamdPods), true, false)
					continue
				}
				pr.resolve("", false, false)
			}
		}()
	}

	for _, t := range targets {
		jobs <- t
	}
	close(jobs)
	wg.Wait()

	pr.finish()
}

// queryEdgecorePodsRemote 按手工执行的形式发起查询：
//
//	qssh qboxserver@<host> "<command>"
//
// 整条命令交给本地 shell 执行，双引号由本地 shell 剥离，qssh 收到的参数与手敲完全一致。
// 只做一次最朴素的调用，不添加任何 qssh 选项、不做路由重试。
func queryEdgecorePodsRemote(loginUser, host string) ([]string, error) {
	if loginUser == "" {
		loginUser = podsQsshUser
	}
	target := host
	if !strings.Contains(target, "@") {
		target = loginUser + "@" + target
	}
	line := buildEdgecorePodsCommand()
	//log.Printf("[Pods] 执行远端查询: qssh %s %s ", target, line)
	pods, _, err := runEdgecorePodsQL(target, line)
	return pods, err
}

// buildEdgecorePodsCommand 构造远端命令，形式与手工执行一致的一条简单管道：
//
//	echo '<base64>' | base64 -d | sh
//
// 命令里不出现任何 shell 变量（$F/$?/$rc 等），也不做多段 && / ; 串联，
// 避免被 qssh 转发过程中的中间层 shell 干扰。
func buildEdgecorePodsCommand() string {
	payload := base64.StdEncoding.EncodeToString([]byte(edgecorePodsScript))
	return fmt.Sprintf("echo '%s' | base64 -d | sh", payload)
}

// runEdgecorePodsQL 用本地 sh 执行完整命令行（含引号），并解析输出
func runEdgecorePodsQL(target, line string) ([]string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), podsQueryTimeout)
	defer cancel()

	// qssh 无 TTY 会改走 WAN root@；script -c 只吃一个命令字符串，不加任何 qssh 选项。
	// 远端命令必须整体加引号，否则 | 会被 script 的本地 shell 吃掉。
	cmd := fmt.Sprintf("qssh %s %q", target, line)
	out, execErr := exec.CommandContext(ctx, "script", "-q", "-e", "-c", cmd, "/dev/null").CombinedOutput()
	//log.Println("[Pods] qssh 执行结果: target=", target, " line=", line, " err=", execErr)
	output := strings.TrimSpace(string(out))

	if ctx.Err() == context.DeadlineExceeded {
		return nil, output, fmt.Errorf("执行超时(%s)", podsQueryTimeout)
	}
	if output == "" {
		// 零输出通常是外层命令本身没跑起来（例如命令被中间层 shell 破坏）
		return nil, output, fmt.Errorf("远端零输出, 退出码错误=%v, 命令: %s",
			execErr, truncateOutput(line, podsErrOutputLimit))
	}

	var (
		pods     []string
		errMsgs  []string
		hasCount bool
	)
	//log.Println("[Pods] 远端输出:\n" + truncateOutput(output, podsErrOutputLimit))
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "POD:"):
			podName := strings.TrimSpace(strings.TrimPrefix(line, "POD:"))
			if podName != "" {
				pods = append(pods, podName)
			}
		case strings.HasPrefix(line, "COUNT:"):
			hasCount = true
		case strings.HasPrefix(line, "ERR:"):
			errMsgs = append(errMsgs, strings.TrimSpace(strings.TrimPrefix(line, "ERR:")))
		}
	}
	if len(errMsgs) > 0 {
		return nil, output, fmt.Errorf("%s", truncateOutput(strings.Join(errMsgs, " | "), podsErrOutputLimit))
	}
	if !hasCount && len(pods) == 0 {
		return nil, output, fmt.Errorf("未获取到 pod 数据, 输出: %s", truncateOutput(output, podsErrOutputLimit))
	}
	if execErr != nil {
		// 有数据但仍返回非零退出码，记录以定位问题
		log.Printf("[Pods] qssh 返回非零退出码: %v", execErr)
	}
	return pods, output, nil
}

func truncateOutput(output string, limit int) string {
	if limit <= 0 || len(output) <= limit {
		return output
	}
	return output[:limit] + "...(truncated)"
}

// podsProgress 打印实时进度：完成数/总数、命中数、失败数
type podsProgress struct {
	mu          sync.Mutex
	interactive bool
	total       int64
	done        int64
	hits        int64
	fails       int64
}

func newPodsProgress(total int64) *podsProgress {
	p := &podsProgress{total: total}
	if fi, err := os.Stdout.Stat(); err == nil {
		p.interactive = fi.Mode()&os.ModeCharDevice != 0
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.interactive {
		p.renderLocked()
	}
	return p
}

func (p *podsProgress) renderLocked() {
	fmt.Printf("\r\033[K[进度] %d/%d 完成, 命中 %d, 失败 %d", p.done, p.total, p.hits, p.fails)
}

// resolve 完成一个节点的处理。line 非空时输出该行结果，再刷新进度。
func (p *podsProgress) resolve(line string, hit bool, failed bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.done++
	if hit {
		p.hits++
	}
	if failed {
		p.fails++
	}
	if line != "" {
		if p.interactive {
			fmt.Printf("\r\033[K%s\n", line)
		} else {
			fmt.Println(line)
		}
	}
	if p.interactive {
		p.renderLocked()
		return
	}
	fmt.Printf("[进度] %d/%d 完成, 命中 %d, 失败 %d\n", p.done, p.total, p.hits, p.fails)
}

func (p *podsProgress) finish() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.interactive {
		fmt.Println()
	}
	log.Printf("[Pods] 全部完成: 总数 %d, 完成 %d, 命中 %d, 失败 %d",
		p.total, p.done, p.hits, p.fails)
}
