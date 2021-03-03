# Beast

![Beast Logo](./docs/res/beast-logo.png)

> Jeopardy-style CTF challenge deployment and management tool.

[![Netlify Status](https://api.netlify.com/api/v1/badges/bea0e0b4-30e1-4830-ba98-e484b51e4036/deploy-status)](https://app.netlify.com/sites/beast-docs-sdslabs/deploys) [![Build Status](https://dev.azure.com/deepshpathak/deepshpathak/_apis/build/status/sdslabs.beastv4?branchName=master)](https://dev.azure.com/deepshpathak/deepshpathak/_build/latest?definitionId=1&branchName=master) [![Apache License](https://img.shields.io/badge/license-Apache-blue.svg)](https://github.com/sdslabs/beastv4/blob/master/LICENSE.md)

## Contents

- [Overview](#overview)
- [Features](#features)
- [Supported Challenge](#supported-challenges)
- [Download](#download)
- [Tech Stack](#tech-stack)
- [Development](#development)
- [Contributing](#contributing)
- [Meet the A-Team](#meet-the-a-team)
- [Contact](#contact)

## Overview

Beast is a service that runs on your host(maybe a bare metal server or a cloud instance) and helps manage deployment, lifecycle, and health check of CTF challenges. It can also be used to host Jeopardy-style CTF competition.

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
- Use of sidecar mechanism for stateful workloads which can be shared by multiple challenges at once, MySQL for example.
- Support for various notification channels like slack, discord.
- Everything embedded to a single go binary which can be easily used anywhere.

For more details on the features, refer to [Features](./docs/Features.md)

## Supported Challenges

As of now beast support the following type of challenges:

- Service - A service hosted on beast container instance
- Web - Web based challenges for various languages including PHP, Python, Node.js etc.
- Static - Challenges with static files, this may include forensics challenges.
- Bare - Highly customisable challenges.
- Docker - Challenges which are provided with their own docker file.

## Download

Assuming you have the [docker](https://www.docker.com/) installed, head over to Beast's [releases](https://github.com/sdslabs/beast/releases) page and grab the latest binary and `setup.sh` script.

Run the `setup.sh` script once. It will setup the required folders and configuration files for you.

Run the downloaded binary with

```bash
$ ./beast run -v
```

## Tech Stack

Beast is written completely in Golang and comes with a clean REST API interface to trigger actions or interact with underlying functionalities.
The REST API server is implemented using `gin` go library and uses JWT as an authentication mechanism. Being written in go, Beast is compiled into
a single binary which can run on any linux distribution.

Beast uses Docker as a container runtimes to run challenges in a sandboxed environment. Note that container does not provide a very strong isolation, but our host is safe as long as there is no 0-day in linux kernel itself. Even though container provide a security layer for the challenges, we follow some practices to harden those security measures.

We use Swagger for automatic generation of API documentation and you can find the docs at `/api/docs/index.html` from beast server root.

To save the state of the deployments and challenges beast uses SQLite as a database, all the information ranging from challenge deployment state to allocated ports and author information is stored in this database. This database is created automatically in the root of your beast configuration directory.

## Development

Beast go version is under development; follow the below instructions to get started.

- Make sure you have docker up and running.
- Install go [1.13.X](https://golang.org/dl/) or above
- Make sure that `GO111MODULES` environment variable should be set to `on`, or do `export GO111MODULES=on`
- Clone the repository.
- Jump to `$GOPATH/src/github.com/sdslabs/beastv4/` and start hacking.

```bash
$ go version
go version go1.13 linux/amd64

$ export GO111MODULES=on

$ git clone git@github.com:sdslabs/beastv4.git

$ cd beastv4 && make help
BEAST: An automated challenge deployment tool for backdoor

* build: Build Beast and copy binary to PATH set for go build binaries.
* check_format: Check for formatting errors using gofmt
* format: format the go files using go_fmt in the project directory.
* test: Run tests for beast
* tools: Set up required tools for Beast which includes - docker-enter, importenv
```

**All the dependencies are already vendored with the project, so no need to install any dependencies**. The project uses go modules from go 1.13.X of dependency management. Make sure you vendor any library used using `go mod vendor`

### Building

To build Beast from Source use the Makefile provided.

- `make build`

This will build Beast and place the binary in `$GOPATH/bin/` and copy the necessery tools to the desired place. To build this in production make sure you also have built the static-content docker image in `/extras/static-content`

To run the API server for Beast, use the command `beast run -v`

### Testing

To test use the sample challenges in the `_examples` directory. Use the challenge simple and try to deploy it using
Beast. Follow the below instructions.

You can find swagger API documentation here: http://localhost:5005/api/docs/index.html

```bash
# Build beast
$ make build

# Run beast server
# Beast server will start running on port 5005 port by default
$ beast run -v

# In another terminal Start the local deployment of the challenge, using the directory
$ curl -X POST localhost:5005/api/manage/deploy/local/ --data "challenge_dir=<absolute_path_to_challenge_simple>"

# Or you can directly deploy the challenge using name in the remote
$ curl -X POST --data "action=deploy&name=<challenge_name>" localhost:5005/api/manage/challenge/

# Wait for Beast to finish the image build and deployment of the challenge
# This might take some time. Have some snacks ready!
# Try connecting to the deployed service
$ nc localhost 10001

--- Menu ---
1.New note
2.Delete note
3.Help
4.Exit
choice > 4
```

### Building documentation

The documentation for the project lies in [/docs](/docs). We use `mkdocs` to automatically generate documentation from markdown. The configuration file for the same can be found at [mkdocs.yml](/mkdocs.yml). To view the documentation locally, create a virtual environment locally and install [requirements](/requirements-dev.txt).

```bash
$ virtualenv venv && source venv/bin/activate

$ pip install -r requirements.txt

$ mkdocs serve

Serving on http://127.0.0.1:8000
```

## Contributing

We are always open for contributions. If you find any feature missing, or just want to report a bug, feel free to open an issue and/or submit a pull request regarding the same.

For more information on contribution, check out our
[docs](./docs/Contribution.md).

## Meet the A-Team

- Deepesh Pathak [@fristonio](https://github.com/fristonio)
- Piyush Sethia [@kokil87](https://github.com/kokil87)
- Shubham Gupta [@shubhamgupta2956](https://github.com/shubhamgupta2956)

You can find the entire list of contributors [here](https://github.com/sdslabs/beastv4/graphs/contributors)

## Contact

If you have a query regarding the product or just want to say hello then feel
free to visit [chat.sdslabs.co](https://chat.sdslabs.co) or drop a mail at
[contact@sdslabs.co.in](mailto:contact@sdslabs.co.in)

---

Made with :heart: by [SDSLabs](https://sdslabs.co)
