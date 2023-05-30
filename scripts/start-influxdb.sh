#!/bin/sh
docker run -d --rm -p 8086:8086 --name InfluxDb     -e DOCKER_INFLUXDB_INIT_MODE=setup \
            -e DOCKER_INFLUXDB_INIT_USERNAME=user \
            -e DOCKER_INFLUXDB_INIT_PASSWORD=password \
            -e DOCKER_INFLUXDB_INIT_ORG=serverledge \
            -e DOCKER_INFLUXDB_INIT_BUCKET=completions \
      -e DOCKER_INFLUXDB_INIT_ADMIN_TOKEN=serverledge \
      influxdb