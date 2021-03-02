# Challenge Config

You can think of beast as a wrapper around the underlying container runtime with a lot of addtional functionalities
including lifecycle management, health checks etc. On a very high level you can say that **beast is to
CTF challenges what Docker is to container.**

Now that we know what is beast actually is(just a wrapper around challenge containers) the question is Why Beast?
The answer to which lies in Why docker when you have runc?

Similar to what docker provides, a high level abstraction to manage the lifecycle, network, state etc among other things
for containers, beast provides a nice abstraction to create, manage and deploy challenges. This allows a challenge creator to 
focus on creating the challenge rather than thinking of everything else. It exposes a very little overhead to the 
side of Challenge Creator apart from creating a challenge and handles everything itself.

Challenge configuration is the heart of challenge deployment using beast. You can think of it as the blueprint for the 
challenge which requires some metadata holding instructions to beast on how to handle the challenge. Think of it as what Dockerfile is 
to Docker image.

Internally since everything we do in beast revolves around containers, this configuration is also used to generate a _Dockerfile_ which is
then used to build the images for the underlying atomic elements to a challenge a container. Think of this challenge coniguration as
a nice wrapper around the Dockerfile itself which is more understandable from a Security Researcher perspective than all the Jargon 
in Dockerfile.

## Structure

The configuration corresponding to a challenge is writtern to a file named `beast.toml` in the root of the challenge directory.
The configuration itself is very minimilistic and is provided in TOML format(mostly because of it's highly readable syntax).

There are three main sections to the configuration the structure of which is as below.

```toml
# Section containing the details corresponding the the author of challenge
[author]

# Stores details corresponding to metadata of challenge
[challenge.metadata]

# Contains the environment or deployment details of the challenge
[challenge.env]
```

All the keys accepted by these sections are mentioned below:

### Author

This section contains the metadata about the author, it is used for various purposes among which the most important 
one is giving the challenge environment access to Author for testing and debugging purposes.

This section accepts the following fields

```toml
# Optional fields
name = ""

# Required Fields
email = ""
ssh_key = "" # Public ssh Key of the author.
```

### Challenge Metadata

This section contains metadata information about the challenge and is consumed by beast to be provided to 
the user.

Structure of the sections with the acceptable fields are:

```toml
# Required Fields
flag = "" # Flag for the challenge
name = "" # Name of the challenge
type = "" # Type of the challenge, one of - Get available types from /api/info/types/available
description = "" # Descritption for the challenge.

# Optional fields.
tags = ["", ""] # Tags that the challenge might belong to, used to do bulk query and handling eg. binary, misc etc.
hints = ["", ""]
sidecar = "" # Name of the sidecar if any used by the challenge.
points = 0 # Points given to the player for correct flag submission. Default value is 0, if not mentioned in the challenge config file 
```

### Challenge Environment

This is the core of deployment configuraiton for the challenge which is consumed by beast.
It contains all the information required by beast to manage the lifecycle of the challenge.

Acceptable fields for this section are:

```toml
# Ports to reserve for the challenge, we bind only one of these to host other are for internal communictaions only.
# Should be within a particular permissible range.
ports = [0, 0]
default_port = 0 # Default port to use for any port specific action by beast.

# Port mapping is the array of port mapping from host to container.
# The first port mentioned in the mapping is the host port and the second is the container port.
# Port Mapping is given preference as compared to ports, so if you have a port and the same port in mapping
# then the host port corresponding to container port in the port mapping.
port_mappings = ["10005:80"]


# Dependencies required by challenge, installed using default package manager of base image apt for most cases.
apt_deps = ["", ""] 


# A list of setup scripts to run for building challenge enviroment.
# Keep in mind that these are only for building the challenge environment and are executed
# in the iamge building step of the deployment pipeline.
setup_scripts = ["", ""]


# A directory containing any of the static assets for the challenge, exposed by beast static endpoint.
static_dir = ""


# Command to execute inside the container, if a predefined type is being used try to
# use an existing field to let beast automatically calculate what command to run.
# If you want to host a binary using xinetd use type service and specify absolute path
# of the service using service_path field.
run_cmd = ""


# Similar to run_cmd but in this case you have the entire container to yourself
# and everything you are doing is done using root permissions inside the container
# When using this keep in mind you are root inside the container.
entrypoint = ""


# Relative path to binary which needs to be executed when the specified
# Type for the challenge is service.
# This can be anything which can be exeucted, a python file, a binary etc.
service_path = ""


# Relative directory corresponding to root of the challenge where the root
# of the web application lies.
web_root = ""


# Any custom base image you might want to use for your particular challenge.
# Exists for flexibility reasons try to use existing base iamges wherever possible.
base_image = ""


# Docker file name for specific type challenge - `docker`.
# Helps to build flexible images for specific user-custom challenges
docket_context = ""


# Environment variables that can be used in the application code.
[[var]]
    key = ""
    value = ""

[[var]]
    key = ""
    value = ""

# Protocol supported by the challenge, currently supported are tcp and udp.
traffic = "tcp"/"udp"
```

If you want to checkout some example challenge configuration, checkout `_example` directory in the 
root of the repository. It has a bunch of challenge templates example to get started with. Pick one from 
there and start building your own challenge.

## Note

We currently don't do automatic port management for challenge, it is mostly due to historic
reasons. Beast still handles challenge deployment for [Backdoor](https://backdoor.sdslabs.co/) which has a different database
as that of beast and to have the port synced among these two database is not easy so for the initial
milestone of beast we targatted static ports.
