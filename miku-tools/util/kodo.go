package util

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/qbox/bo-sdk/base/xlog.v1"
	"github.com/qbox/bo-sdk/sdk/qconf/appg"
	"github.com/qbox/bo-sdk/sdk/qconf/qconfapi"
	"github.com/qbox/pili/base/qiniu/api/auth/digest"
	urlescape "github.com/qiniu/x/url"
)

//"github.com/qbox/linking/internal/qvs.v1"

var (
	app appg.Client
)

func SignResource() {
	//qvs.NewKodo()
	qc := qconfapi.New(&Conf.AccountCfg)
	app = appg.Client{Conn: qc}
	if Conf.Uid == "" {
		log.Println("need uid")
		return
	}
	if Conf.Key == "" {
		log.Println("need key")
		return
	}
	if Conf.Domain == "" {
		log.Println("need domain")
		return
	}
	uidInt, err := strconv.ParseUint(Conf.Uid, 10, 32)
	if err != nil {
		log.Println(err)
		return
	}
	//key := fmt.Sprintf("%s/%s/%s", Conf.Bucket, Conf.Ns, Conf.Key)
	addr, err := SignURL(context.Background(), uint32(uidInt), "http", Conf.Domain, Conf.Key, 1000)
	if err != nil {
		log.Println("SignUrl err", err)
		return
	}
	fmt.Println(addr)
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
