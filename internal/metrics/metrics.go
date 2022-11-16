package metrics

import (
	"fmt"
	"log"

	"net/http"

	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/node"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var Enabled bool
var registry = prometheus.NewRegistry()
var nodeIdentifier string

func Init() {
	if config.GetBool(config.METRICS_ENABLED, false) {
		log.Println("Metrics enabled.")
		Enabled = true
	} else {
		Enabled = false
		return
	}

	nodeIdentifier = node.NodeIdentifier
	registerGlobalMetrics()

	//portNumber := config.GetInt(config.METRICS_PROMETHEUS_PORT, 2112)
	//host := config.GetString(config.METRICS_PROMETHEUS_HOST, "")

	portNumber := config.GetInt(config.METRICS_PORT, 2112)

	addr := fmt.Sprintf(":%d", portNumber)

	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true})
	http.Handle("/metrics", handler)
	http.ListenAndServe(addr, nil)
}

// Global metrics
var (
	CompletedInvocations = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sedge_completed_total",
		Help: "The total number of completed function invocations",
	}, []string{"node", "function"})

	CompletedInvocationsOffloaded = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sedge_completed_offloaded_total",
		Help: "The total number of completed offloaded function invocations",
	}, []string{"node", "function"})

	Arrivals = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "arrivals_total",
		Help: "The total number of arrivals",
	}, []string{"node", "function", "class"})

	ExecutionTimes = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "sedge_exectime",
		Help: "Function duration",
	},
		[]string{"node", "function"})

	ExecutionTimesOffloaded = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "sedge_exectime_offloaded",
		Help: "Function duration if offloaded",
	},
		[]string{"node", "function"})

	InitTime = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "sedge_init_time",
		Help:    "Function init time",
		Buckets: initBuckets,
	},
		[]string{"node", "function"})

	ColdStarts = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sedge_cold_starts",
		Help: "The total number of cold starts",
	}, []string{"node", "function"})

	InitTimeOffloaded = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "sedge_init_time_offloaded",
		Help:    "Function init time",
		Buckets: initBuckets,
	},
		[]string{"node", "function"})

	ColdStartsOffloaded = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sedge_cold_starts_offloaded",
		Help: "The total number of offloaded cold starts",
	}, []string{"node", "function"})

	OffloadTime = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "sedge_offload_time",
		Help:    "Function offload time",
		Buckets: latencyBuckets,
	},
		[]string{"node", "function"})
)

// TODO add custom buckets
var durationBuckets = []float64{0.002, 0.005, 0.010, 0.02, 0.03, 0.05, 0.1, 0.15, 0.3, 0.6, 1.0}
var initBuckets = []float64{0.002, 0.005, 0.010, 0.02, 0.03, 0.05, 0.1, 0.15, 0.3, 0.6, 1.0}
var latencyBuckets = []float64{0.002, 0.005, 0.010, 0.02, 0.03, 0.05, 0.1, 0.15, 0.3, 0.6, 1.0}

func AddCompletedInvocation(funcName string) {
	CompletedInvocations.With(prometheus.Labels{"function": funcName, "node": nodeIdentifier}).Inc()
}

func AddCompletedInvocationOffloaded(funcName string) {
	CompletedInvocationsOffloaded.With(prometheus.Labels{"function": funcName, "node": nodeIdentifier}).Inc()
}

func AddArrivals(funcName string, className string) {
	Arrivals.With(prometheus.Labels{"function": funcName, "node": nodeIdentifier, "class": className}).Inc()
}

func AddColdStart(funcName string, duration float64) {
	ColdStarts.With(prometheus.Labels{"function": funcName, "node": nodeIdentifier}).Inc()
	InitTime.With(prometheus.Labels{"function": funcName, "node": nodeIdentifier}).Observe(duration)
}

func AddColdStartOffload(funcName string, duration float64) {
	ColdStartsOffloaded.With(prometheus.Labels{"function": funcName, "node": nodeIdentifier}).Inc()
	InitTimeOffloaded.With(prometheus.Labels{"function": funcName, "node": nodeIdentifier}).Observe(duration)
}

func AddOffloadingTime(funcName string, duration float64) {
	OffloadTime.With(prometheus.Labels{"function": funcName, "node": nodeIdentifier}).Observe(duration)
}

func AddFunctionDurationValue(funcName string, duration float64) {
	ExecutionTimes.With(prometheus.Labels{"function": funcName, "node": nodeIdentifier}).Observe(duration)
}

func AddFunctionDurationOffloadedValue(funcName string, duration float64) {
	ExecutionTimesOffloaded.With(prometheus.Labels{"function": funcName, "node": nodeIdentifier}).Observe(duration)
}

func registerGlobalMetrics() {
	registry.MustRegister(CompletedInvocations)
	registry.MustRegister(CompletedInvocationsOffloaded)
	registry.MustRegister(ExecutionTimes)
	registry.MustRegister(ExecutionTimesOffloaded)
	registry.MustRegister(Arrivals)
	registry.MustRegister(InitTime)
	registry.MustRegister(ColdStarts)
	registry.MustRegister(InitTimeOffloaded)
	registry.MustRegister(ColdStartsOffloaded)
	registry.MustRegister(OffloadTime)
}
