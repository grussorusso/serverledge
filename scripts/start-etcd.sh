#!/bin/sh
if [ "$#" != 1 ] || ([ "$1" != "podman" ] && [ "$1" != "docker" ]); then
        echo "[Error]: Please specify a container manager flag. Supported options: 'podman' || 'docker'"
        echo "Usage: ./start-etcd.sh [OPTION]"
        exit 10
fi

$1 run -d --rm --name Etcd-server \
    --publish 2379:2379 \
    --publish 2380:2380 \
    --env ALLOW_NONE_AUTHENTICATION=yes \
    --env ETCD_ADVERTISE_CLIENT_URLS=http://192.168.1.105:2379 \
    docker.io/bitnami/etcd:latest
