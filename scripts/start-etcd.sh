#!/bin/sh
docker ps 2>&1 > /dev/null || sudo systemctl start docker

docker run -d --rm --name Etcd-server \
    --publish 2379:2379 \
    --publish 2380:2380 \
    --env ALLOW_NONE_AUTHENTICATION=yes \
    --env ETCD_ADVERTISE_CLIENT_URLS=http://localhost:2379 \
    bitnami/etcd:3.5.14-debian-12-r1
