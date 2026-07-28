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

func genConnectID() (string, error) {
	b := make([]byte, 5)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (m *Miku) reproDoRegister(client *http.Client, registerURL string, req StreamRegisterRequest) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		log.Printf("[register] marshal err: %v", err)
		return
	}
	log.Printf("[register] req connectId=%s", req.ConnectId)
	resp, err := client.Post(registerURL, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		log.Printf("[register] err: %v", err)
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("[register] resp=%s", string(respBody))
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

	log.Printf("开始循环: count=%d, bucket=%s, key=%s, node=%s", loopCount, bucket, key, node)
	log.Println("流程: register -> sleep 20ms -> unregister || sleep 3ms -> register -> sleep 30ms -> 读 Redis")
	log.Printf("register=%s", registerURL)
	log.Printf("unregister=%s", unregisterURL)
	log.Printf("redis=%s:%d key=%s", redisHost, redisPort, redisKey)
	log.Println()

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
		}

		done := make(chan struct{}, 2)
		go func() {
			m.reproDoUnregister(client, unregisterURL, unregisterReq)
			done <- struct{}{}
		}()
		time.Sleep(3 * time.Millisecond)
		go func() {
			m.reproDoRegister(client, registerURL, registerReq)
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
			log.Printf("发现 Redis key 为空，退出循环。round=%d, old=%s, new=%s", i, connectIDOld, connectIDNew)
			os.Exit(0)
		}
		for k, v := range result {
			log.Printf("  %s: %s", k, v)
		}
		log.Println()
	}

	log.Printf("循环结束，共执行 %d 次，Redis key 一直非空。", loopCount)
	os.Exit(1)
}
