# Xion Daemon

The Xion Daemon is scaffolded off of [CosmWasm/wasmd](https://github.com/CosmWasm/wasmd)
rather than being scaffolded with ignite in order to more easily achieve
compatibility with the latest cosmos-sdk and CosmWasm releases.





## Running integration tests

### Prerequisites
* [docker](https://docs.docker.com/get-docker/)
* [heighliner](https://github.com/strangelove-ventures/heighliner)

### Build and run
At the root of the project, run:
```bash 
heighliner build -c xion --local --no-cache --no-build-cache && XION_IMAGE=xion:local make test-integration
```

> **Note**
> This will take some time (10+ minutes) to run the as it will need to build the docker image and pull dependencies.

The final line of output should read as follows if successful:
```bash
ok      integration_tests       164.191s
```
