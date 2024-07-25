package main

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/qbox/mikud-live/cmd/monitor/common/consts"
)

var (
	dynIpStatusMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: consts.MikuSchedPrometheusNamespace,
			Subsystem: consts.InterfaceDataSubsystem,
			Name:      "dyn_node_ip",
			Help:      "动态节点ip监控",
		}, []string{"status"})
)

func (s *Parser) dynIpMonitor(ipStatusMap map[string]int) {
	dynIpStatusMetric.Reset()
	for status, cnt := range ipStatusMap {
		dynIpStatusMetric.With(prometheus.Labels{"status": status}).Set(float64(cnt))
	}
	err := push.New(s.conf.PrometheusAddr, "miku_sched").
		Grouping("instance", "sched").Collector(dynIpStatusMetric).Add()
	if err != nil {
		log.Println("dynIpMonitor err", err)
	}
}
