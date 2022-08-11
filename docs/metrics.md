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


## Prometheus Deployment

Various Prometheus configurations can be considered to scrape Serverledge
metrics:

- A centralized Prometheus server in the Cloud (likely not scalable...)
- A Prometheus server in each Edge zone
- A Prometheus server in the Cloud with a Prometheus Agent on each Serverledge
  node

As regards the last option, it requires Prometheus instances to use the
following (minimal) configuration.

In the Serverledge node,
Prometheus must be started with `--enable-feature=agent` and the following
lines in the configuration:

	remote_write:
	  - url: "http://<prometheus_cloud_host>:9091/api/v1/write"

In the Cloud, 
Prometheus must be started with `--web.enable-remote-write-receiver`.


## References

- [Prometheus Agent Mode](https://prometheus.io/blog/2021/11/16/agent/)
- [Prometheus + Go](https://prometheus.io/docs/guides/go-application/)
