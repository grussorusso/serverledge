#!/bin/sh
if [ "$#" != 1 ] || ([ "$1" != "podman" ] && [ "$1" != "docker" ]); then
        echo "[Error]: Please specify a container manager flag. Supported options: 'podman' || 'docker'"
        echo "Usage: ./start-prometheus.sh [OPTION]"
        exit 10
fi

$1 run \
    --name prometheusLocal \
    -d --rm \
    -p 9090:9090 \
    -v $(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml \
    prom/prometheus:v2.37.1  \
	--config.file=/etc/prometheus/prometheus.yml
