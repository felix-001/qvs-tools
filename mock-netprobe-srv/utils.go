package main

import (
	"sort"

	"github.com/qbox/mikud-live/common/model"
)

func (s *NetprobeSrv) findNode(nodeId string) *model.RtNode {
	for _, node := range s.nodes {
		if node.Id == nodeId {
			return node
		}
	}
	return nil
}

func SortFloatMap(m map[string]float64) []Pair {
	pairs := []Pair{}
	for k, v := range m {
		pairs = append(pairs, Pair{Key: k, Val: v})
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Val < pairs[j].Val
	})
	return pairs
}
