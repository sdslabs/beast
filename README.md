# Beast

> Beast is an automatic deployment and management tool for CTF challenges hosted on backdoor.sdslabs.co

## Development

Beast go version is under development, follow the below instructions to get started.

* Install go 1.11.x
* Clone the repository.
* Jump to `$GOPATH/src/github.com/sdslabs/beastv4/` and start hacking.

**All the dependencies are already vendored with the project so no need to install any dependencies**. The projcet uses go modules from 
go 1.11.X fo dependency management. Make sure you vendor any library used using `go mod vendor`

### Building

To build beast from Source use the Makefile provided.

* `make build`

### Structure

* **api**
	* API exposed by beast
	* This uses `gin` as rest API framework and routes are grouped under `/api`

* **build**
	* Build scripts for beast.

* **cmd**
	* Package containing command line functionality of beast.
	* `commands.go` is the main entrypoint for the package
	* This makes use of spf13/cobra for command line flag parsing.

* **core**
	* Core functionalities of beast
	* It includes package for beast database management and deploy pipeline.

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

### Testing

To test use the sample challenges in the `_examples` directory. Use the challenge simple and try to deploy it using
beast. Follow the below instructions.

```bash
# Build beast
$ make build

# Run beast server
# Beast server will start running on port 5005 port by default
$ beast run -v

# In another terminal Start the local deployment of the challenge, using the directory
$ curl -X POST localhost:5005/api/deploy/local/ --data "challenge_dir=<absolute_path_to_challenge_simple>"

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

### Development notes

Beast uses `logrus` for logging purposes and follow standard effective go guidelines, so anytime you are writing a code keep in mind to 
add necessery logs and documentation. Also format the code before commiting using `gofmt`. Or simply run the make command `make test`

The design documentation for the new Beast can be found [here](https://docs.google.com/document/d/1BlRes900aFS2s8jicrSx2W7b1t1FnYZhx70jGQu__HE/edit)

