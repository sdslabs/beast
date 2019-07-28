# Usage

## Installation

Before provisioning beast on your host, make sure you have the following dependencies installed.

* [Docker for Linux](https://docs.docker.com/install/linux/docker-ce/ubuntu/)
* [GoLang for Linux](https://golang.org/doc/install#tarball)
* [git](https://git-scm.com/)

You can either build beast from source or download a latest realease binary from Github Relaeases.

```bash
$ export GO111MODULES=on

$ git clone git@github.com:sdslabs/beastv4.git

$ cd beastv4 && make build
>>> Building Beast
```

This should build beast from source in `$GOPATH/bin/beast`, you can then use this binary to run beast API server. The `-n`
flag tells beast to not use the authorization middleware.

```bash
$ beast run -v -n
```

To interact with beast API server you can look at the swagger API documentation hosted on beast itself. Navigate to http://localhost:5005/api/docs/index.html to get a detail of available endpoints.

To be able to interact with the REST API interface you should be authorized by beast, go to [Authentication Section](/APIAuth) to know about the flow of authentication.

## Examples

Some examples of API action triggers are given below

```bash
# Reloading a configuration change in beast global config
$ curl -X PATCH localhost:5005/api/config/reload
{"message":"CONFIG RELOAD SUCCESSFUL"}

# Deploying a challenge named my-challenge using API.
$  curl -X POST --data "action=deploy&name=my-challenge" localhost:5005/api/manage/challenge/
{"message":"Deploy for challenge simple-web has been triggered, check stats"}

# Purging the deployed challenge completely.
$ curl -X POST --data "action=purge&name=my-challenge" localhost:5005/api/manage/challenge/
{"message":"Your action purge on challenge simple-web was successful"}
```

For more examples and available API routes go to Swagger API documentation.

## Note

* You can even run beast on your local environment and still be able to configure and manage deployments, for this you will need a secure tunnel to reach your docker daemon, which will be used for container lifecycle management on the actual host.
