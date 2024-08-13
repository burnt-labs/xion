#!/bin/sh

docker compose rm -f -s -v
docker volume rm -f devnet_shared
docker image rm burnt/xion:develop 2>/dev/null
