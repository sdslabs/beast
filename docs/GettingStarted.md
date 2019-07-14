# Getting Started

Beast is a tool for automatic deployment of CTF type challenges, the intial aim of the project
was to asist the deployment for challenges on backdoor.sdslabs.co, but since its inception beast has grown a lot beyond that scope.
It is a general tool for deployment of Jeopardy style CTF challenges and is not coupled with backdoor anymore.

The main hurdle for deployment of such challenges is the requirement of stong isolation
of the environment in which the challenges are being deployed, docker provides us with all the
sandboxing we need with minimum overhead. There are more secure runtime like `runsc`(gVisor) which can 
be used to improve the sandboxing capabilities of the containers.

There are a lot of features that comes embedded with beast, take a look [here](Features)

Beast comes in with an embedded web server which can be used as an interface for interacting with beast.
The web server is built with gin framework and is very performant, to run the server with debugging mode on
use the following command 

```bash
beast run -v -p 3333
```

To interact with beast you need to first authenticate yourself, currently this process is a little bit tedious since we
don't currently have a strict database storing the details of our users. To check out how the authentication
flow works for beast take a look [here](APIAuth)

Once you have the authentication token with you all you need to do is Embed the token in Headers of your request as
`Authorization: Bearer <token>`.

There is swagger generated API documentation with the details of endpoints exposed by beast which can be used for interacting
with the web server.

## Deploying your first challenge

In this section we will try to create a new challenge and deploy it using beast. For the simplicitiy of this tutorial we are
deploying a simple buffer overflow challenge.

The source code for our challenge file is:

```c
#include <stdio.h>
#include <unistd.h>

int sample()
{	FILE *ptr_file;
	char buf[100];

	ptr_file = fopen("flag.txt","r");
	if (!ptr_file)
		return 1;

	while (fgets(buf,100, ptr_file)!=NULL)
		fprintf(stderr, "%s",buf);
	fclose(ptr_file);
	return 0;
}

void test()
{	char input[50];
	gets(input);
	sleep(1);
	fprintf(stderr, "ECHO: %s\n",input); 
}

int main()
{	test();
	return 0;
}
```

Create a new directory for the challenge and create the source code file(`pwn_me.c`) with the above contents.

Create a file beast.toml with the following contents which defines the configuration of the challenge

```toml
[author]
name = "fristonio"                      # Name of the challenge creator
email = "deepeshpathak09@gmail.com"     # Email for contact
ssh_key = "ssh-rsa AAAAB3NzaC1y..."	    # Public SSH key for the challenge author

[challenge.metadata]
name = "PWN-TEST"                    # Name of the challenge, must be same as the directory name.
flag = "FLAG{TEST_CHALLENGE_PWNED}"  # Flag for the challenge
type = "service"                     # Type of challenge

# This section defines the environment for the challenge
[challenge.env]

# Define the dependencies we might need for the challenge.
# For example in this case we need gcc for compilation of the source file
apt_deps = ["gcc", "socat"]

# The relative path of the binary or executable which we should
# run for each connection to the challenge.
service_path = "pwn"

# Port to run the challenge on.
ports = [10003]

# Since we still haven't defined how we are going to compile the source 
# code, these scripts are for setting up the environment.
setup_scripts = ["setup.sh"]
```

The above configuration is simple and straightforward, from the perspective of the challenge
creator it simply asks how will he run the challenge locally. So for example he needs to install
some dependencies first like gcc, then he compiles the source and generates a binary, then he serves
the binary by exposing it as a service at some port. So from a challenge creator perspective
this is not a big hurdle.

Let's see how our setup scripts look like.

```bash
set -e

gcc -o pwn pwn_me.c
```

* All the commands that you execute are executed from the within the root of the challenge
directory.

It's simple we just compile the source code.

The final step is important and needed to be taken care of. We have setup the challenge but now we also 
need to provide the binary we obtained to the participant as part of the challenge.

Beast provides a way to do this using a special file named `post-build.sh`. This script is run once finally
after all our environment setup is done, in this file you can read/write/modify the final environment once more
before finally coming to the challenge. Up until this point we have the binary with us, we can write this script
to copy the final binary to the publically available directory within the challenge named `public/`.

This directory(`public/`) is exposed by beast using the static content provider. To make any file from within your challenge
publically accessible as part of the challenge you need to put that file in this directory rest is handled by beast itself.

This is how our `post-bulid.sh` script looks like for this particular challenge

```bash
#!/bin/bash

set -euxo pipefail

cp pwn public/
```

And that's it, we have our challenge ready to be deployed by beast.

## Deployment using beast

We have our challenge ready with the required configuration. To deploy the challenge check out the Deployment flow [here](Deployment).

## Note

* To know more about the environment configuration possiblities with beast head out to [this section](ChallConfig).
