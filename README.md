# Beast

![Beast Logo](./docs/res/beast-logo.png)

> An automatic deployment and management tool for CTF challenges

[![Netlify Status](https://api.netlify.com/api/v1/badges/bea0e0b4-30e1-4830-ba98-e484b51e4036/deploy-status)](https://app.netlify.com/sites/beast-docs-sdslabs/deploys) [![Build Status](https://dev.azure.com/deepshpathak/deepshpathak/_apis/build/status/sdslabs.beastv4?branchName=master)](https://dev.azure.com/deepshpathak/deepshpathak/_build/latest?definitionId=1&branchName=master) [![Apache License](https://img.shields.io/badge/license-Apache-blue.svg)](https://github.com/sdslabs/beastv4/blob/master/LICENSE.md)

## Overview

Beast is a service that runs on your host(may be a bare metal server or a cloud instance) and helps in the mangement of deployment, lifecycle and health check of CTF challenges. Beast is created to automate and ease the deployment procedure of challenges for a Jeopardy style CTF competition.

## Features

- Git based source of truth.
- Container based isolation
- Easy configuration
- Testing and SSH support for challenge instances
- Web and command line interface to perform actions and host competitions
- REST API interface for the entire ecosystem
- An optional automated health check service to periodically check the status of challenges and report if there is some sort of problem with one.
- Single source of truth for all the static content related to all the challenges making it easy to debug, monitor and manage
  static content through a single interface.
- Use of sidecar mechanism for stateful workloads which can be shared by multiple challenges at once, MySQL for example.
- Support for various notification channels like slack, discord.
- Compatibility with Linux, Windows, MacOS, FreeBSD and OpenBSD.
- Everything embedded to a single go binary which can be easily used anywhere.

For more details on the features, refer to [Features](./docs/Features.md)

## Supported Challenge's Types

As of now beast support the following type of challenges:

- Service - A service hosted on beast container instance
- Web - Web based challenges for various languages including PHP, Python, Node.js etc.
- Static - Challenges with static files, this may include forensics challenges.
- Bare - Highly customisable challenges.
- Docker - Challenges which are provided with their own docker file.

## Development

Beast go version is under development; follow the below instructions to get started.

- Install go 1.13 or above
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

### Documentation

The documentation for the project lies in [/docs](/docs). We use `mkdocs` to automatically generate documentation from markdown. The configuration file for the same can be found at [mkdocs.yml](/mkdocs.yml). To view the documentation locally, create a virtual environment locally and install [requirements](/requirements-dev.txt).

```bash
$ virtualenv venv && source venv/bin/activate

$ pip install -r requirements.txt

$ mkdocs serve

Serving on http://127.0.0.1:8000
```

### Development notes

Beast uses `logrus` for logging purposes and follows standard effective go guidelines, so anytime you are writing a code, keep in mind to add necessary logs and documentation. Also, format the code before committing using `gofmt`. Or simply run the make command `make test`

For any API routes, you add to the beast API, do write Swagger API documentation.

The design documentation for the new Beast can be found [here](https://docs.google.com/document/d/1BlRes900aFS2s8jicrSx2W7b1t1FnYZhx70jGQu__HE/edit)

## Contributing

We are always open for contributions. If you find any feature missing, or just want to report a bug, feel free to open an issue and/or submit a pull request regarding the same.

For more information on contribution, check out our
[docs](https://kiwi.sdslabs.co/docs/contribution-guide.html).

## Contact

If you have a query regarding the product or just want to say hello then feel
free to visit [chat.sdslabs.co](https://chat.sdslabs.co) or drop a mail at
[contact@sdslabs.co.in](mailto:contact@sdslabs.co.in)

---

Made by [SDSLabs](https://sdslabs.co)
