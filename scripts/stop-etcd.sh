#!/bin/sh
if [ "$#" != 1 ] || ([ "$1" != "podman" ] && [ "$1" != "docker" ]); then
        echo "[Error]: Please specify a container manager flag. Supported options: 'podman' || 'docker'"
        echo "Usage: ./stop-etcd.sh [OPTION]"
        exit 10
fi

$1 stop Etcd-server -t 0
