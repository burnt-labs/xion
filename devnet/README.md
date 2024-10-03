# Xion Devnet

Xion Devnet is a multi validator sandbox environment orchestrated with docker compose

## Prerequisites

- [docker](https://www.docker.com/)
- [docker compose](https://github.com/docker/compose)

## Running Devnet

Start Devnet
(From the root of this project)
```sh
cd ./devnet
NUM_VALIDATORS=3 docker compose up
```

Adjust the number of validators (up to 10) per your needs.

By default this will build a container using the local repository state
and create a local xion network with three validators.

Stop Devnet:

```sh
docker compose stop
```

After stopping you may resume from the previous height with'

```sh
docker compose start
```

Remove all/reset:

```sh
docker compose rm -f -s -v
docker volume rm devnet_shared
docker network rm devnet_default
```

Rebuild the container image used

```sh
docker compose build
```
