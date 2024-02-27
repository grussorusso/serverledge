#!/bin/sh
docker run -d --rm --name Etcd-server \
    --publish 2379:2379 \
    --publish 2380:2380 \
    --env ALLOW_NONE_AUTHENTICATION=yes \
    --env ETCD_ADVERTISE_CLIENT_URLS=http://localhost:2379 \
    bitnami/etcd:latest

docker run -d --rm -p 8086:8086 --name InfluxDb     -e DOCKER_INFLUXDB_INIT_MODE=setup \
            -e DOCKER_INFLUXDB_INIT_USERNAME=user \
            -e DOCKER_INFLUXDB_INIT_PASSWORD=password \
            -e DOCKER_INFLUXDB_INIT_ORG=serverledge \
            -e DOCKER_INFLUXDB_INIT_BUCKET=completions \
      -e DOCKER_INFLUXDB_INIT_ADMIN_TOKEN=serverledge \
      influxdb

docker run -d -p 2500:2500 --rm --name Solver ferrarally/serverledge-solver