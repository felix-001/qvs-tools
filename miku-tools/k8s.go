package main

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/qbox/mikud-live/common/model"
	publicUtil "github.com/qbox/mikud-live/common/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func (s *Parser) K8s() {
	config, err := clientcmd.BuildConfigFromFlags("", s.conf.KubeCfg)
	if err != nil {
		s.logger.Error().Err(err).Msg("BuildConfigFromFlags")
		return
	}
	cli, err := kubernetes.NewForConfig(config)
	if err != nil {
		s.logger.Error().Err(err).Msg("NewForConfig")
		return
	}
	resp, err := cli.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		s.logger.Error().Err(err).Msg("k8s list")
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
			if _, ok := s.allNodesMap[node.Name]; !ok {
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
		node, ok := s.allNodesMap[nodeId]
		if !ok {
			continue
		}
		s.logger.Info().Str("node", nodeId).Msg("notReadys")
		for _, ipInfo := range node.Ips {
			if publicUtil.IsPrivateIP(ipInfo.Ip) {
				continue
			}
			if !IsPublicIPAddress(ipInfo.Ip) {
				continue
			}
			wg.Add(1)
			go func(nodeIdIn string, nodeIn *model.RtNode) {
				defer wg.Done()
				addr := fmt.Sprintf("%s:%d", ipInfo.Ip, node.StreamdPorts.Http)
				if CheckServer(addr) {
					ch <- node.MachineId
					s.logger.Info().Str("node", nodeIdIn).Str("machine", nodeIn.MachineId).Msg("online")
					return
				}
				s.logger.Info().Str("node", nodeIdIn).Str("addr", addr).Str("machine", nodeIn.MachineId).Msg("offline")
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
