package pkg

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)


var (
	FanoutTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name:      "total_count",
		Help:      "Total number of messages received via fanout",
	}, []string{"queue_name"})

	FanoutSuccess = promauto.NewCounterVec(prometheus.CounterOpts{
		Name:      "success_count",
		Help:      "Total number of successful messages submitted via fanout",
	}, []string{"queue_name"})

	FanoutFailure = promauto.NewCounterVec(prometheus.CounterOpts{
		Name:      "failure_count",
		Help:      "Total number of messages failed to be submitted via fanout",
	}, []string{"queue_name"})

	FanoutPendingRequests = promauto.NewGauge(prometheus.GaugeOpts{
		Name:      "pending_requests_gauge",
		Help:      "Number of pending operations",
	})

	InvalidRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name:      "invalid_requestcount",
		Help:      "Total number of requests that were not accepted by the fanout",
	}, []string{})
)

