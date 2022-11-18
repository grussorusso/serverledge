#!/bin/sh
docker run -d --rm -p 8086:8086 --name InfluxDb     -e DOCKER_INFLUXDB_INIT_MODE=setup \
            -e DOCKER_INFLUXDB_INIT_USERNAME=my-user \
            -e DOCKER_INFLUXDB_INIT_PASSWORD=my-password \
            -e DOCKER_INFLUXDB_INIT_ORG=serverledge \
            -e DOCKER_INFLUXDB_INIT_BUCKET=completions \
      -e DOCKER_INFLUXDB_INIT_ADMIN_TOKEN=my-token \
      influxdb