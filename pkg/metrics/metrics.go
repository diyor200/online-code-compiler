package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	// total executions
	TotalExecutions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "compiler_executions_total",
			Help: "Total number of compiler executions",
		}, []string{"language", "status"},
	)

	// execution duration
	ExecutionDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "compiler_execution_duration_seconds",
			Help:    "Compiler execution duration in seconds",
			Buckets: []float64{0.5, 1, 2, 5, 10, 20, 30},
		}, []string{"language"},
	)

	// active executions
	ActiveExecutions = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "compiler_active_executions",
			Help: "Number of active compiler executions",
		},
	)

	//container creation failure
	ContainerFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "container_creation_failures_total",
			Help: "Number of container creation failures",
		},
		[]string{"language", "stage"},
	)

	// rate limit hits
	RateLimitHits = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "compiler_rate_limit_hits_total",
			Help: "Total number of rate limit rejections",
		},
	)
)

func RegisterMetrics() {
	prometheus.MustRegister(
		TotalExecutions,
		ExecutionDuration,
		ActiveExecutions,
		ContainerFailures,
		RateLimitHits,
	)
}
