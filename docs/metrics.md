# Metrics

The metrics system must be enabled via `metrics.enabled`.
If enabled, metrics are exposed at `http://localhost:2112/metrics`.

You can check that the metrics system is working without starting a Prometheus
server:

	$ curl 127.0.0.1:2112/metrics 


## Available metrics

A few metrics are currently exposed (just for demonstration purposes):

- `sedge_completed_total`: number of completed invocations (Counter, per function)
- `sedge_exectime`: execution time for each function (Histogram, per function)

## References

- [Prometheus Agent Mode](https://prometheus.io/blog/2021/11/16/agent/)
- [Prometheus + Go](https://prometheus.io/docs/guides/go-application/)
