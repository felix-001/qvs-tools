package miku

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

type streamUnregisterRequest struct {
	Bucket    string `json:"bucket"`
	Key       string `json:"key"`
	Node      string `json:"nodeId"`
	ConnectId string `json:"connectId"`
	Url       string `json:"url"`
	RawUrl    string `json:"rawUrl"`
}

type streamRegisterResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	ConnectId string `json:"connectId"`
}

func genConnectID() (string, error) {
	b := make([]byte, 5)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (m *Miku) reproDoRegister(client *http.Client, registerURL string, req StreamRegisterRequest) string {
	reqBody, err := json.Marshal(req)
	if err != nil {
		log.Printf("[register] marshal err: %v", err)
		return ""
	}
	log.Printf("[register] req connectId=%s masterKey=%s", req.ConnectId, req.Master)
	resp, err := client.Post(registerURL, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		log.Printf("[register] err: %v", err)
		return ""
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("[register] resp=%s", string(respBody))

	var parsed streamRegisterResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		log.Printf("[register] unmarshal resp err: %v", err)
		return ""
	}
	return parsed.Code
}

func (m *Miku) reproDoUnregister(client *http.Client, unregisterURL string, req streamUnregisterRequest) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		log.Printf("[unregister] marshal err: %v", err)
		return
	}
	log.Printf("[unregister] req connectId=%s", req.ConnectId)
	resp, err := client.Post(unregisterURL, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		log.Printf("[unregister] err: %v", err)
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("[unregister] resp=%s", string(respBody))
}

func (m *Miku) reproHGetAll(ctx context.Context, rdb redis.UniversalClient, key string) (map[string]string, error) {
	return rdb.HGetAll(ctx, key).Result()
}

func (m *Miku) ReproRegisterUnregisterRace() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	conf := m.conf
	loopCount := conf.Loop
	if loopCount <= 0 {
		log.Println("循环次数必须是正整数，请通过 -loop 指定")
		os.Exit(1)
	}

	bucket := conf.Bucket
	if bucket == "" {
		bucket = "liyqtest"
	}
	key := conf.Stream
	if key == "" || key == "teststream" {
		key = "livetest"
	}
	node := conf.Node
	if node == "" {
		node = "a1a2561a-2b2d-3d1b-a119-1d15d7793e5a-vdn-jscz-dls-1-9"
	}
	domain := conf.Domain
	if domain == "" || domain == "push.liyqtest.com" {
		domain = "test.push.qiniuapi.com"
	}
	schedIP := conf.SchedIp
	if schedIP == "" {
		schedIP = "10.34.146.62"
	}
	redisHost := conf.Host
	if redisHost == "" {
		redisHost = "10.34.91.37"
	}
	redisPort := conf.Port
	if redisPort == 0 {
		redisPort = 8200
	}

	pushURL := conf.Url
	if pushURL == "" {
		pushURL = fmt.Sprintf("http://%s/%s/xxx.flv", domain, bucket)
	}

	registerURL := fmt.Sprintf("http://%s:6060/api/v1/streamregister", schedIP)
	unregisterURL := fmt.Sprintf("http://%s:6060/api/v1/streamunregister", schedIP)
	redisKey := fmt.Sprintf("stream_sched_%s:%s", bucket, key)

	rdb := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:      []string{fmt.Sprintf("%s:%d", redisHost, redisPort)},
		MaxRetries: 1,
	})
	defer rdb.Close()

	client := &http.Client{Timeout: 10 * time.Second}
	ctx := context.Background()

	sleepMs := conf.SleepMs
	if sleepMs < 0 {
		sleepMs = 0
	}

	log.Printf("开始循环: count=%d, bucket=%s, key=%s, node=%s, sleep_ms=%d", loopCount, bucket, key, node, sleepMs)
	log.Printf("流程: register || sleep %dms -> unregister -> sleep 30ms -> 读 Redis", sleepMs)
	log.Printf("判定: new=1000 且 Redis 空 => HIT 真竞态; new!=1000 且 Redis 空 => 假阳性跳过")
	log.Printf("register=%s", registerURL)
	log.Printf("unregister=%s", unregisterURL)
	log.Printf("redis=%s:%d key=%s", redisHost, redisPort, redisKey)
	log.Println()

	falsePositive := 0

	for i := 1; i <= loopCount; i++ {
		connectIDOld, err := genConnectID()
		if err != nil {
			log.Fatalf("生成 connectId 失败: %v", err)
		}
		connectIDNew, err := genConnectID()
		if err != nil {
			log.Fatalf("生成 connectId 失败: %v", err)
		}

		log.Printf("========== round %d/%d old=%s new=%s ==========", i, loopCount, connectIDOld, connectIDNew)

		m.reproDoRegister(client, registerURL, StreamRegisterRequest{
			Bucket:    bucket,
			Key:       key,
			Node:      node,
			Url:       pushURL,
			RawUrl:    pushURL,
			Type:      "live",
			ConnectId: connectIDOld,
			Domain:    domain,
			Master:    key, // 模拟 lived: req.Master = key
		})
		time.Sleep(20 * time.Millisecond)

		unregisterReq := streamUnregisterRequest{
			Bucket:    bucket,
			Key:       key,
			Node:      node,
			ConnectId: connectIDOld,
			Url:       pushURL,
			RawUrl:    pushURL,
		}
		registerReq := StreamRegisterRequest{
			Bucket:    bucket,
			Key:       key,
			Node:      node,
			Url:       pushURL,
			RawUrl:    pushURL,
			Type:      "live",
			ConnectId: connectIDNew,
			Domain:    domain,
			Master:    key, // 关键：绕过 CheckStream 的 Stream is online，进入 Set/Del 竞态窗口
		}

		var newCode string
		done := make(chan struct{}, 2)
		go func() {
			newCode = m.reproDoRegister(client, registerURL, registerReq)
			done <- struct{}{}
		}()
		time.Sleep(time.Duration(sleepMs) * time.Millisecond)
		go func() {
			m.reproDoUnregister(client, unregisterURL, unregisterReq)
			done <- struct{}{}
		}()
		<-done
		<-done

		time.Sleep(30 * time.Millisecond)

		result, err := m.reproHGetAll(ctx, rdb, redisKey)
		if err != nil {
			log.Printf("[redis] HGETALL %s err: %v", redisKey, err)
		}
		log.Printf("[redis] HGETALL %s =>", redisKey)
		if len(result) == 0 {
			log.Println("(empty)")
			// 真阳性：new 注册成功(1000) 但 Redis 被 old unregister Del 掉
			if newCode == "1000" {
				log.Printf("HIT race: new register success but redis empty. round=%d, old=%s, new=%s", i, connectIDOld, connectIDNew)
				os.Exit(0)
			}
			// 假阳性：new 未成功写入（如 1037），old Del 后 Redis 空
			falsePositive++
			log.Printf("skip false positive: newCode=%s redis empty. fp=%d round=%d", newCode, falsePositive, i)
			log.Println()
			continue
		}
		for k, v := range result {
			log.Printf("  %s: %s", k, v)
		}
		log.Println()
	}

	log.Printf("循环结束，共执行 %d 次，Redis key 一直非空（或未打中真竞态，假阳性 %d 次）。", loopCount, falsePositive)
	os.Exit(1)
}
