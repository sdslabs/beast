# Challenge Types

## Service Challenge

Any service whether it is a binary file, or a shell script, which needs to be instantiated on every connection can be easily hosted using `service` type challenge. **Xinetd** is for hosting these type of challenges inside a docker container.

###Primary Requirements

```toml
# Relative path to binary or script which needs to be executed when the specified
# Type for the challenge is service.
# This can be anything which can be exeucted, a python file, a binary etc.
service_path = ""
```

## Web Challenge

Web challenges are hosted using the corresponding images from Dockerhub. Currently only these types are supported:

* Node
* Python : Django and Flask
* Php

###Primary Requirements

```toml
# Relative directory corresponding to root of the challenge where the root
# of the web application lies.
web_root = ""
```

## Static Challenge

All the challenges which requires the hackers to only have static files comes under `static` challenges. All the files are mounted on a single container which serves all the static files to the hackers.

## Bare Challenge

A challenge which requires high level of customization can be hosted using `bare` challenge. In these case, a bare base image is provided with access to mentioned sidecars, and exposed ports. 

###Primary Requirements

```toml
# Command to execute inside the container, if a predefined type is being used try to
# use an existing field to let beast automatically calculate what command to run.
run_cmd = ""

# OR

# Provide a script to run on startup of container
# Similar to run_cmd but in this case you have the entire container to yourself
# and everything you are doing is done using root permissions inside the container
# When using this keep in mind you are root inside the container.
entrypoint = ""
```

## Docker Challenge

Authors might have tested the challenges in a isolated docker environment and might not want to port the challenge to one of these types. So they can use `docker` type challenge in which you can provide your own docker context file and ports.

###Primary Requirements

```toml
# Docker file name for specific type challenge - `docker`.
# Helps to build flexible images for specific user-custom challenges
docket_context = ""
```
