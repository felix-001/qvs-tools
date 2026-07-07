package util

import (
	"context"
	"fmt"
	"io"
	"log"
	"mikutool/config"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/qbox/bo-sdk/base/xlog.v1"
	"github.com/qbox/bo-sdk/sdk/qconf/appg"
	"github.com/qbox/bo-sdk/sdk/qconf/qconfapi"
	"github.com/qbox/pili/base/qiniu/api/auth/digest"
	"github.com/qbox/pili/base/qiniupkg.com/api.v7/kodo"
	ipdb "github.com/qbox/pili/common/ipdb.v1"
	urlescape "github.com/qiniu/x/url"
)

//"github.com/qbox/linking/internal/qvs.v1"

var (
	app appg.Client
)

func SignResource(conf *config.Config) {
	//qvs.NewKodo()
	qc := qconfapi.New(&conf.AccountCfg)
	app = appg.Client{Conn: qc}
	if conf.Uid == "" {
		log.Println("need uid")
		return
	}
	if conf.Key == "" {
		log.Println("need key")
		return
	}
	if conf.Domain == "" {
		log.Println("need domain")
		return
	}
	uidInt, err := strconv.ParseUint(conf.Uid, 10, 32)
	if err != nil {
		log.Println(err)
		return
	}
	//key := fmt.Sprintf("%s/%s/%s", Conf.Bucket, Conf.Ns, Conf.Key)
	addr, err := SignURL(context.Background(), uint32(uidInt), "http", conf.Domain, conf.Key, 1000)
	if err != nil {
		log.Println("SignUrl err", err)
		return
	}
	fmt.Println(addr)
}

// DownloadIpdb 使用 ipdb 配置(ipdb.ips_source_param)中的 remote_url 从 kodo 下载 ipdb 文件到 /tmp 目录。
// 参考 pili ipdb.v1 中 cityWorker.loadRemote 的实现，通过 kodo.Client 生成私有下载链接后拉取文件。
func DownloadIpdb(conf *config.Config) {
	if len(conf.IPDB.IP) == 0 {
		log.Println("no ipdb config, check ipdb.ips_source_param in mikutool.yaml")
		return
	}
	for name, param := range conf.IPDB.IP {
		if param == nil || param.RemoteUrl == "" {
			log.Printf("ipdb [%s] has no remote_url, skip\n", name)
			continue
		}
		if err := downloadIpdbFromKodo(name, param); err != nil {
			log.Printf("download ipdb [%s] err: %v\n", name, err)
			continue
		}
	}
}

func downloadIpdbFromKodo(name string, param *ipdb.IPSourceReqParam) error {
	u, err := url.Parse(param.RemoteUrl)
	if err != nil {
		return fmt.Errorf("parse remote_url %s err: %w", param.RemoteUrl, err)
	}
	if u.Path == "" || u.Path == "/" {
		return fmt.Errorf("remote_url %s has no key path", param.RemoteUrl)
	}

	key := u.Path[1:]
	baseUrl := kodo.MakeBaseUrl(u.Host, key)
	kc := kodo.New(0, &kodo.Config{
		AccessKey: param.AK,
		SecretKey: param.SK,
		RSHost:    baseUrl,
	})

	deadline := time.Now().Add(time.Second * 3600).Unix()
	policy := &kodo.GetPolicy{
		Expires: uint32(deadline),
	}
	privateUrl := kc.MakePrivateUrl(kc.RSHost, policy)
	privateAccessUrl := fmt.Sprintf("%s&now=%d", privateUrl, time.Now().Unix())
	log.Printf("ipdb [%s] load privateAccessUrl: %s\n", name, privateAccessUrl)

	dst := filepath.Join("/tmp", fmt.Sprintf("%s.ipdb", name))

	client := &http.Client{Timeout: 60 * time.Second}
	retry := int(param.RemoteRetry)
	if retry <= 0 {
		retry = 1
	}

	var lastErr error
	for i := 0; i < retry; i++ {
		if lastErr = httpDownloadFile(client, privateAccessUrl, dst); lastErr != nil {
			log.Printf("download ipdb [%s] attempt %d/%d err: %v\n", name, i+1, retry, lastErr)
			continue
		}
		log.Printf("download ipdb [%s] success, saved to %s\n", name, dst)
		return nil
	}
	return lastErr
}

func httpDownloadFile(client *http.Client, downloadUrl, dst string) error {
	resp, err := client.Get(downloadUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("http.Get %s resp.StatusCode %d", downloadUrl, resp.StatusCode)
	}

	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()

	n, err := io.Copy(f, resp.Body)
	if err != nil {
		return fmt.Errorf("write %s err: %w", dst, err)
	}
	log.Printf("downloaded %d bytes to %s\n", n, dst)
	return nil
}

func SignURL(ctx context.Context, uid uint32, scheme, domain, key string, ttl int64) (string, error) {

	u := fmt.Sprintf("%s://%s/%s", scheme, domain, urlescape.EscapeEx(key, urlescape.EncodePath))

	ak, sk, err := app.GetAkSk(xlog.FromContextSafe(ctx), uid)
	if err != nil {
		return "", err
	}

	u += fmt.Sprintf("?e=%d", time.Now().Unix()+ttl)

	mac := &digest.Mac{AccessKey: ak, SecretKey: []byte(sk)}

	token := mac.Sign([]byte(u))

	return fmt.Sprintf("%s&token=%s", u, token), nil
}
