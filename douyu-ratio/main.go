package main

import "log"

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

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	streamAnalysis("85894rmovieChow_4000h")
}
