<p align="center">
  <img src="./docs/res/beast-logo.png" alt="Beast Logo">
</p>

<h3 align="center">Jeopardy-style CTF challenge deployment and management tool.</h3>

<p align="center">
 <a href="https://app.netlify.com/sites/beast-docs-sdslabs/deploys">
    <img src="https://api.netlify.com/api/v1/badges/bea0e0b4-30e1-4830-ba98-e484b51e4036/deploy-status" alt="Netlify Status" />
  </a>
 <a href="https://dev.azure.com/deepshpathak/deepshpathak/_build/latest?definitionId=1&branchName=master">
    <img src="https://dev.azure.com/deepshpathak/deepshpathak/_apis/build/status/sdslabs.beastv4?branchName=master" alt="Build Status" />
  </a>
 <a href="https://github.com/sdslabs/beastv4/blob/master/LICENSE.md">
    <img src="https://img.shields.io/badge/license-Apache-blue.svg" alt="Apache License" />
  </a>
</p>

## Contents

- [Overview](#overview)
- [Features](#features)
- [Supported Challenge](#supported-challenges)
- [Setup](#setup)
- [Tech Stack](#tech-stack)
- [Development](#development)
- [Contributing](#contributing)
- [Contact](#contact)

## Overview

Beast is a service that runs on your host(maybe a bare metal server or a cloud instance) and helps manage deployment, lifecycle, and health check of CTF challenges. It can also be used to host Jeopardy-style CTF competition.

Visit [beast.sdslabs.co](https://beast.sdslabs.co/) for the more details and documentation

If you're looking for the source code of playCTF, the frontend powered by Beast, visit https://github.com/sdslabs/playCTF. 

## Features

- Git based source of truth.
- Container based isolation
- Easy configuration
- SSH support for challenge instances
- Command line interface to perform actions and host competitions
- REST API interface for the entire ecosystem
- An optional automated health check service to periodically check the status of challenges and report if there is some sort of problem with one.
- Single source of truth for all the static content related to all the challenges making it easy to debug, monitor and manage
  static content through a single interface.
- Support for various notification channels like slack, discord.
- Single Go binary for the CLI and API, with Docker, PostgreSQL, and Redis as runtime dependencies.

For more details on the features, refer to [Features](./docs/Features.md)

## Supported Challenges

As of now beast support the following type of challenges:

- Service - A service hosted on beast container instance
- Web - Web based challenges for various languages including PHP, Python, Node.js etc.
- Static - Challenges with static files, this may include forensics challenges.
- Bare - Highly customisable challenges.
- Custom Dockerfile - Challenges that provide their own image build using `docker_context`.
- Docker Compose - Multi-service challenges that provide a Compose file using `docker_compose`.
- Instanced - Per-user containers for supported service and Docker Compose configurations.

## Setup

Build from this checkout to use the code described here. The setup guide covers Docker, PostgreSQL, Redis, and first-time configuration. [Setup guide](./docs/Setup.md)

## Tech Stack

Beast is written completely in Golang and comes with a clean REST API interface to trigger actions or interact with underlying functionalities.
The REST API server is implemented using `gin` go library and uses JWT as an authentication mechanism. Being written in go, Beast is compiled into
a single binary which can run on any linux distribution.

Beast uses Docker and Docker Compose to run and manage challenge workloads.

We use Swagger for automatic generation of API documentation and you can find the docs at `/api/docs/index.html` from beast server root.

To save the state of deployments, challenges, users, and submissions Beast uses PostgreSQL (`psql_config`) and Redis (`redis_config`) for caching and instanced-challenge state/port allocation.

## Development

Beast go version is under active development; follow the below instructions to get started.

- Make sure you have Docker, PostgreSQL, and Redis up and running.
- Install Go 1.23+ (`go.mod` targets `go 1.23.0`)
- Clone the repository.
- Run `make help` inside the repository root to inspect available targets.

```bash
$ go version
go version go1.23.0 linux/amd64

$ git clone git@github.com:sdslabs/beast.git

$ cd beast && make help
BEAST: An automated challenge deployment tool for backdoor

* build: Build Beast and copy binary to PATH set for go build binaries.
* dev: Run development environment with hot-reloading enabled
* check_format: Check for formatting errors using gofmt
* format: format the go files using go_fmt in the project directory.
* requirements: Build beast extra artifacts requirements
* test: Run the Go test suite
* test-examples: Validate all bundled challenge configurations
* tools: Set up required tools for Beast which includes - docker-enter, importenv
```

**Dependencies are tracked in `go.mod` (with optional `vendor/` sync via `go mod vendor`).**

### Building

To build Beast from Source use the Makefile provided.

- `make build`

This builds Beast into the Go binary directory (`go env GOBIN`, or the first `go env GOPATH` entry followed by `/bin`) and installs the tools needed for SSH access and provisioning. To run the separate static content server, build the image in [extras/static-content](./extras/static-content).

To run the API server for Beast, use the command `beast run -v` (or `beast run -nv` for local development without JWT authorization)

`make up NAME=myctf` starts the PostgreSQL and Redis dependencies. Start the Beast API separately with `beast run`. When running Beast in a container, provision the Hydra network and firewall rules on the Docker host before deploying instanced Compose challenges. See the [instanced Compose example](./_examples/instanced-compose).

### Hot reloading support

- run the following command to install `air` (hot reload support)

```bash
curl -sSfL https://raw.githubusercontent.com/cosmtrek/air/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
```
- modify `full_bin` in `.air.toml` for changing the arguments to run beast, default being `beast run -nv`

- run `make dev` to start `beast` in development mode

### Testing

Run `make test` for the Go suite and `make test-examples` to validate all ten bundled configurations. The example check requires a built and configured Beast. The PostgreSQL, Redis, and Docker integration tests use disposable services.

| Test dependency | Configuration |
| --- | --- |
| PostgreSQL | Set `BEAST_TEST_PG_DSN` to the test database connection string. |
| Redis | Set `BEAST_TEST_REDIS_ADDR`. For the API tests, set `REDIS_HOST` and `REDIS_PORT` to a disposable Redis server that accepts unauthenticated test connections. Set `BEAST_TEST_REDIS_FLUSH=1` only on an isolated test database to enable tests that clear it. |
| Docker | Set `BEAST_TEST_DOCKER=1` and, optionally, `BEAST_TEST_DOCKER_IMAGE`. |

Run `go test -race -count=1 ./...` with the fixture variables set to include integration checks. The submission concurrency script under `scripts/test/` uses its own PostgreSQL and Redis connection settings and recreates the selected test database.

For a smoke test, follow the [static challenge walkthrough](./docs/GettingStarted.md). Swagger is available at `/api/docs/index.html` on a running Beast server.

### Building documentation

```sh
python3 -m venv .venv
. .venv/bin/activate
python -m pip install -r requirements.txt
make docs
```

The documentation site and its Swagger page are generated into `site/`. Run `make cmdref` to refresh the CLI pages after changing commands. See the [API contributor notes](./api/README.md) for Swagger regeneration.

## Contributing

We are always open for contributions. If you find any feature missing, or just want to report a bug, feel free to open an issue and/or submit a pull request regarding the same.

For more information on contribution, check out our
[docs](./docs/Contribution.md).

## Contact

If you have a query regarding the product or just want to say hello then feel
free to visit [chat.sdslabs.co](https://chat.sdslabs.co) or drop a mail at
[contact@sdslabs.co.in](mailto:contact@sdslabs.co.in)

---

Made with :heart: by [SDSLabs](https://sdslabs.co)
