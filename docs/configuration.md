# Configuration #

This page provides a (partial) list of configuration options that can
be specified in Serverledge configuration files.
All the supported configuration keys are defined in `internal/config/keys.go`.

## Configuration files

You can provide a configuration file using YAML or TOML syntax. Depending on the
chosen format, the default file name will be `serverledge-conf.yaml` or
`serverledge-conf.toml`. The file can be either placed in `/etc/serverledge`,
in the user `$HOME` directory, or in the working directory where the server is
started.

Alternatively, you can indicate a specific configuration file when starting the
server:

	$ bin/serverledge <config file>


## Frequently used options

| Configuration key  | Description   | Example value(s) |
| -------------      | ------------- | -----------------|
| `etcd.address` | Hostname and port of the Etcd server acting as the Global Registry. | `127.0.0.1:2379` | 
| `api.port` |Port number for the API server. | 1323| 
| `cloud.server.url` |URL prefix for the remote Cloud node API. | `http://127.0.0.1:1326` | 
| `factory.images.refresh` |Forces function runtime container images to be pulled from the Internet the first time they are used (to update them), even if they are available on the host.| `true` | 
| `container.pool.memory` |Maximum amount of memory (in MB) that the container pool can use (must be not greater than the total memory available in the host).|4096| 
| `janitor.interval` |Activation interval (in seconds) for the janitor thread that checks for expired containers.| 60| 
| `container.expiration` |Expiration time (in seconds) for idle containers. | 600|
| `registry.area` |Geographic area where this node is located.| `ROME`| 
| `registry.udp.port` |UPD port used for peer-to-peer Edge monitoring.|| 
| `scheduler.policy` |Scheduling policy to use. Possible values: `default`, `localonly`, `edgeonly`, `cloudonly`.|| 

<!-- TODO:
| `container.pool.cpus` ||| 
| `cache.size` ||| 
| `cache.cleanup` ||| 
| `cache.expiration` ||| 
| `scheduler.queue.capacity` ||| 
| `metrics.enabled` ||| 
| `metrics.prometheus.host` ||| 
| `metrics.prometheus.port` ||| 
| `registry.nearby.interval` ||| 
| `registry.monitoring.interval` |||
| `registry.ttl` ||| 
-->
