package util

import (
	"log"
	"middle-source-analysis/config"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/qbox/mikud-live/cmd/monitor/common/consts"
)

var (
	DynIpStatusMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: consts.MikuSchedPrometheusNamespace,
			Subsystem: consts.InterfaceDataSubsystem,
			Name:      "dyn_node_ip",
			Help:      "动态节点ip监控",
		}, []string{"status"})
)

func DynIpMonitor(ipStatusMap map[string]int, conf *config.Config) {
	DynIpStatusMetric.Reset()
	for status, cnt := range ipStatusMap {
		DynIpStatusMetric.With(prometheus.Labels{"status": status}).Set(float64(cnt))
	}
	err := push.New(conf.PrometheusAddr, "miku_sched").
		Grouping("instance", "sched").Collector(DynIpStatusMetric).Add()
	if err != nil {
		log.Println("dynIpMonitor err", err)
	}
}
