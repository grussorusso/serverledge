#!/bin/sh
docker run \
    --name prometheusLocal \
    -d --rm \
    -p 9090:9090 \
    -v $(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml \
    prom/prometheus:v2.37.1  \
	--config.file=/etc/prometheus/prometheus.yml
