package debug

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type PrometheusMetrics struct {
	Registry              *prometheus.Registry
	RelayedHeadersCounter *prometheus.CounterVec
	RelayedChainsCounter  *prometheus.CounterVec
	FailedHeadersCounter  *prometheus.CounterVec
	FailedChainsCounter   *prometheus.CounterVec
}

func NewPrometheusMetrics() *PrometheusMetrics {
	headerLabels := []string{"src_chain", "dst_chain"}
	chainLabels := []string{"src_chain", "dst_chain"}
	registry := prometheus.NewRegistry()
	registerer := promauto.With(registry)
	return &PrometheusMetrics{
		Registry: registry,
		RelayedHeadersCounter: registerer.NewCounterVec(prometheus.CounterOpts{
			Name: "cosmos_relayer_relayed_headers",
			Help: "The total number of relayed headers",
		}, headerLabels),
		RelayedChainsCounter: registerer.NewCounterVec(prometheus.CounterOpts{
			Name: "cosmos_relayer_relayed_chains",
			Help: "The total number of relayed chains",
		}, chainLabels),
		FailedHeadersCounter: registerer.NewCounterVec(prometheus.CounterOpts{
			Name: "cosmos_relayer_failed_headers",
			Help: "The total number of headers that are failed to be relayed",
		}, headerLabels),
		FailedChainsCounter: registerer.NewCounterVec(prometheus.CounterOpts{
			Name: "cosmos_relayer_failed_chains",
			Help: "The total number of chains that are failed to be relayed",
		}, chainLabels),
	}
}
