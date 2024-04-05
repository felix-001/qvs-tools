package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

func streamAnalysis(streamid string) {
	usedNodes := []string{}
	nodes := GetDynamicNodesData()
	log.Println("total nodes:", len(nodes))
	for i, node := range nodes {
		if !CheckNode(node) {
			log.Println("skip node:", node.Id)
			continue
		}
		streams := GetNodeStreams(node.Id)
		if streams == nil {
			continue
		}
		for _, stream := range streams.Streams {
			if stream.Key == streamid {
				usedNodes = append(usedNodes, node.Id)
				break
			}
		}
		log.Println("idx:", i)
	}
	log.Println(usedNodes, len(usedNodes))
}

func douyuSourceUrl() {
	expireTime := time.Now().Unix() + 600
	hexTime := strconv.FormatInt(expireTime, 16)
	raw := fmt.Sprintf("%s%s%s", os.Args[2], os.Args[1], hexTime)
	hash := md5.Sum([]byte(raw))
	txSecret := hex.EncodeToString([]byte(hash[:]))
	originUrl := fmt.Sprintf("http://%s/%s/%s.flv?txSecret=%s&txTime=%s", "qngin.douyucdn.cn",
		"douyu", os.Args[1], txSecret, hexTime)
	log.Println(originUrl)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	//streamAnalysis("85894rmovieChow_4000h")
	douyuSourceUrl()
}
