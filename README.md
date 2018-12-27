# Beast

> Beast is an automatic deployment and management tool for CTF challenges hosted on backdoor.sdslabs.co

## Development

Beast go version is under development, follow the below instructions to get started.

* Install go 1.11.x
* Make sure that `GO111MODULES` environment variable should be set to `on`, or do `export GO111MODULES=on`
* Clone the repository.
* Jump to `$GOPATH/src/github.com/sdslabs/beastv4/` and start hacking.

```bash
$ go version
go version go1.11 linux/amd64

$ export GO111MODULES=on

$ git clone git@github.com:sdslabs/beastv4.git

$ cd beastv4 && make help
BEAST: An automated challenge deployment tool for backdoor

* build: Build beast and copy binary to PATH set for go build binaries.
* check_format: Check for formatting errors using gofmt
* format: format the go files using go_fmt in the project directory.
* test: Run tests for beast
* tools: Set up required tools for beast which includes - docker-enter, importenv
```

**All the dependencies are already vendored with the project so no need to install any dependencies**. The projcet uses go modules from 
go 1.11.X fo dependency management. Make sure you vendor any library used using `go mod vendor`

### Building

To build beast from Source use the Makefile provided.

* `make build`

This will build beast and will place the binary in `$GOPATH/bin/` will also copy the necessery tools to desired place. To build this in production make sure you also have built the static-content docker image in `/extras/static-content`

To run the API server for beast use command `beast run -v`

### Directory Structure

* **api**
	* API exposed by beast
	* This uses `gin` as rest API framework and routes are grouped under `/api`
	* API Docs are served using swagger API specs.

* **scripts**
	* Build scripts for beast.
	* Other relevent scripts for beast including docker-enter.

* **cmd**
	* Package containing command line functionality of beast.
	* `commands.go` is the main entrypoint for the package
	* This makes use of spf13/cobra for command line flag parsing.

* **core**
	* Core functionalities of beast
	* It includes package managing challenges.
	* Inside package `manager` lives the code relating to all the core functionality that beast provides.

* **database**
	* Database wrapper using gorm for beast.

* **docker**
	* Docker wrapper for beast container API provider

* **git**
	* Git functions wrapper provider for beast functions.

* **notify**
	* Package implementing notification functionality for beast, this includes slack notifications.

* **templates**
	* Tempaltes used by beast.
	* For example - Beast dockerfile template, beast challenge config template etc.

* **utils**
	* Beast utility functions package.

* **version**
	* Version package for beast.
	* Use `beast version`

* **_examples**
	* This directory contains example challenges for beast.
	* An example beast global root config to be placed in `$HOME/.beast/config.toml`

### Testing

To test use the sample challenges in the `_examples` directory. Use the challenge simple and try to deploy it using
beast. Follow the below instructions.

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

# Wait for beast to finish the image build and deployment of the challenge
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

The documentation for the project lies in [/docs](/docs). We use `mkdocs` to automatically generate documentation from markdown. The configuration file for the same can be found at [mkdocs.yml](/mkdocs.yml). To view the documentation locally create a virtual environment locally and install [requirements](/requirements-dev.txt).

```bash
$ virtualenv venv && source venv/bin/activate

$ pip install -r requirements-dev.txt

$ mkdocs serve

Serving on http://127.0.0.1:8000
```

### Development notes

Beast uses `logrus` for logging purposes and follow standard effective go guidelines, so anytime you are writing a code keep in mind to 
add necessery logs and documentation. Also format the code before commiting using `gofmt`. Or simply run the make command `make test`

For any API routes you add to the beast API do write Swagger API documentation.

The design documentation for the new Beast can be found [here](https://docs.google.com/document/d/1BlRes900aFS2s8jicrSx2W7b1t1FnYZhx70jGQu__HE/edit)

