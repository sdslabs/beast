# Features

### Git based source of truth

* Single or multiple git repositories as reliable source of truth for all the challenges.

* Easy to collaborate similar to an open source application where everyone can give their reviews on your contribution.

* Automatic deployment of challenges by using a triggering pipeline in conjunction with beast web server, this can
be configured in many ways:
    * Execute a dry run on the challenge using beast.
    * Using github webhook trigger deploy on beast when a challenge in pushed.

### Container based Isolation

We use containers as the atomic source of handle for each of our challenge. They provides us with an isolated and secure
environment for the challenges.

* Currently only docker based container runtime support is available but we are extending to create a generalized
container runtime interface implementation to support multiple providers similar to what kubernetes does.

* Optionally security or sandboxing capabilities can be further enhanced by using more secure runtime like `runsc` in place
of runc.

* We are also looking to support VM based implmentation in place of these containers such as firecracker, kata containers,
intel clear containers etc.

### Easy Configuration

Beast provides an easy configuration interface which allows the challenge creator to focus on only one problem which 
is creating the challenge rather than thinking about the deployment scenarios for the same. There is a minimal overhead
due to simplicity of configuration that the author goes through during challenge creation.

Configuration is even less of a pain due to great sensible defaults provided beast which works for most of the cases but are of
course configurable.

To know more about configuration parameters provided by beast move to [this section](ChallConfig)

### Testing Support

Beast provides challenge author access to each instance of all the challenges so that these challenges can be
debugged on the fly in case there is a need for. This also means that challenge author can test/debug the challenges before publishing 
them.

For challenges which needs high degree of customization the author can create the environment by sshing to the container
and then export the running container image getting a tarball which can be used later to reproduce the
environments.

### Miscellaneous

* An optional automated health check service to periodically check the status of challenges and report if there is
some sort of problem with one.

* Web and Command line interface to perform actions.

* Single source of truth for all the static content related to all the challenges making it easy to debug, monitor and manage
static content through a single interface.

* Extensible and flexible structure for easier development and feature introduction, some features we are exploring to support 
with beast are:
    * Kubernetes(k8s or k3s) based control plane to manage the lifecycle of challenges.
    * Multi server support.
    * Dry run support to help locally support development of challenges.

* Use of sidecar mechanism for stateful workloads which can be shared by multiple challenges at once, MySQL for example.

* Everything embedded to a single go binary which can be easily used anywhere.
