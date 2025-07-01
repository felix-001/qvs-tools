package node

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	localUtil "middle-source-analysis/util"

	"github.com/qbox/mikud-live/common/model"
	publicUtil "github.com/qbox/mikud-live/common/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func K8s() {
	config, err := clientcmd.BuildConfigFromFlags("", Conf.KubeCfg)
	if err != nil {
		return
	}
	cli, err := kubernetes.NewForConfig(config)
	if err != nil {
		return
	}
	resp, err := cli.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return
	}
	//fmt.Println(resp)
	var notReadys []string
	for _, node := range resp.Items {
		for _, status := range node.Status.Conditions {
			if status.Type != corev1.NodeReady {
				continue
			}
			if status.Status == corev1.ConditionTrue {
				continue
			}
			//notReadys = append(notReadys, node.Name)
			if _, ok := AllNodesMap[node.Name]; !ok {
				continue
			}
			notReadys = append(notReadys, node.Name)
			break
		}
	}
	//fmt.Println(notReadys)
	fmt.Println("cnt:", len(notReadys))

	var wg sync.WaitGroup
	//wg.Add(len(notReadys))
	ch := make(chan string, len(notReadys))
	for _, nodeId := range notReadys {
		node, ok := AllNodesMap[nodeId]
		if !ok {
			continue
		}
		for _, ipInfo := range node.Ips {
			if publicUtil.IsPrivateIP(ipInfo.Ip) {
				continue
			}
			if !localUtil.IsPublicIPAddress(ipInfo.Ip) {
				continue
			}
			wg.Add(1)
			go func(nodeIdIn string, nodeIn *model.RtNode) {
				defer wg.Done()
				addr := fmt.Sprintf("%s:%d", ipInfo.Ip, node.StreamdPorts.Http)
				if CheckServer(addr) {
					ch <- node.MachineId
					return
				}
			}(nodeId, node)
			break
		}
	}

	go func() {
		wg.Wait() // 等待所有goroutine完成
		close(ch) // 所有goroutine完成后关闭channel
	}()

	var nodeIds []string
	for {
		nodeId, ok := <-ch
		if !ok {
			break
		}
		nodeIds = append(nodeIds, nodeId)
	}
	fmt.Println("final cnt:", len(nodeIds))
	fmt.Println(nodeIds)
}

func CheckServer(addr string) bool {
	_, err := net.DialTimeout("tcp", addr, time.Duration(1*time.Second))
	return err == nil
}
