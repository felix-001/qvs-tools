package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"time"
)

func main() {
	app := os.Args[1]
	stream := os.Args[2]
	key := os.Args[3]
	domain := os.Args[4]

	expireTime := time.Now().Unix() + int64(600)
	hexTime := strconv.FormatInt(expireTime, 16)
	raw := fmt.Sprintf("%s%s%s", key, stream, hexTime)
	hash := md5.Sum([]byte(raw))
	txSecret := hex.EncodeToString([]byte(hash[:]))
	originUrl := fmt.Sprintf("http://%s/%s/%s.flv?txSecret=%s&txTime=%s", domain,
		app, stream, txSecret, hexTime)
	fmt.Println("url: ", originUrl)
}
