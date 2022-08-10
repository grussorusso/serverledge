# Metrics

The metrics system must be enabled via `metrics.enabled`.
If enabled, metrics are exposed at `http://localhost:2112/metrics`.

You can check that the metrics system is working without starting a Prometheus
server:

	$ curl 127.0.0.1:2112/metrics 

Example output:

	# HELP sedge_completed_total The total number of completed function invocations
	# TYPE sedge_completed_total counter
	sedge_completed_total 2


## References

- [Prometheus Agent Mode](https://prometheus.io/blog/2021/11/16/agent/)
- [Prometheus + Go](https://prometheus.io/docs/guides/go-application/)
