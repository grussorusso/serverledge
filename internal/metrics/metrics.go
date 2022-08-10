package metrics

import (
	"log"

	"net/http"

	"github.com/grussorusso/serverledge/internal/config"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var Enabled bool
var registry = prometheus.NewRegistry()

func Init() {
	if config.GetBool(config.METRICS_ENABLED, false) {
		log.Println("Metrics enabled.")
		Enabled = true
	} else {
		log.Println("Metrics disabled.")
		Enabled = false
		return
	}

	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true})
	http.Handle("/metrics", handler)
	http.ListenAndServe(":2112", nil)
}

// Example
//var (
//	opsProcessed = promauto.NewCounter(prometheus.CounterOpts{
//		Name: "myapp_processed_ops_total",
//		Help: "The total number of processed events",
//	})
//)
//
//func recordMetrics() {
//	registry.Register(opsProcessed)
//	go func() {
//		for {
//			opsProcessed.Inc()
//			time.Sleep(2 * time.Second)
//		}
//	}()
//}
