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


## Prometheus Integration

Various Prometheus configurations can be considered to scrape Serverledge
metrics:

- A centralized Prometheus server in the Cloud (likely not scalable...)
- A Prometheus server in each Edge zone
- A Prometheus server in the Cloud with a Prometheus Agent on each Serverledge
  node (details below)

### Example: Prometheus Agent + Cloud

As regards the last option, it requires Prometheus instances to use the
following (minimal) configuration.

In the Serverledge node,
Prometheus must be started with `--enable-feature=agent` and the following
lines in the configuration:

	remote_write:
	  - url: "http://<prometheus_cloud_host>:9091/api/v1/write"

Example configuration:

	global:
	  scrape_interval: 15s # Set the scrape interval to every 15 seconds. Default is every 1 minute.
	  evaluation_interval: 15s # Evaluate rules every 15 seconds. The default is every 1 minute.
	  # scrape_timeout is set to the global default (10s).

	# A scrape configuration containing exactly one endpoint to scrape:
	scrape_configs:
	  - job_name: "serverledge"
	    # metrics_path defaults to '/metrics'
	    # scheme defaults to 'http'.
	    static_configs:
	      - targets: ["<serverledge_host>:2112"]

	remote_write:
	  - url: "http://<prometheus_cloud_host>:9091/api/v1/write"

In the Cloud, Prometheus must be started with `--web.enable-remote-write-receiver`.
Example configuration:

	global:
	  scrape_interval: 15s # Set the scrape interval to every 15 seconds. Default is every 1 minute.
	  evaluation_interval: 15s # Evaluate rules every 15 seconds. The default is every 1 minute.

Example script to launch both Prometheus instances on the same host (for
testing):

	docker run \
	    --name prom \
	    -d \
	    -p 9090:9090 \
	    -v $(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml \
	    prom/prometheus --enable-feature=agent \
		--config.file=/etc/prometheus/prometheus.yml

	docker run \
	    --name promRemote \
	    -d\
	    \
	    -p 9091:9090 \
	    -v $(pwd)/prometheus_remote.yml:/etc/prometheus/prometheus.yml \
	    prom/prometheus --web.enable-remote-write-receiver \
		--config.file=/etc/prometheus/prometheus.yml

### References

- [Prometheus Agent Mode](https://prometheus.io/blog/2021/11/16/agent/)
- [Prometheus + Go](https://prometheus.io/docs/guides/go-application/)
