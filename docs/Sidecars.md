# Sidecars

Sidecar container in beast are the container that provide additional functionality to existing challenge containers. For a challenge you can specify one or more sidecar containers, which your container can then access with the variables injected in container Environment Variables. As a example let's take MySQL sidecar for web challenges as an instance.

## Usage

### Challenge Configuration

To use a sidecar container with your challenge container you will need to specify the sidecar in the challenge configuration file(beast.toml) you will register a service as a sidecar in this file, for example

```toml
[challenge.metadata]
sidecar = "mysql"
```

Speciying a sidecar will inject some sidecar related configuration as environment variable inside your challenge container, which you can then use to interact with the sidecar. In case of mysql these configuration variable will include, `MYSQL_HOST`, `MYSQL_PASSWORD`, `MYSQL_DATABASE`, `MYSQL_PORT`. Then inside your challenge you can use these details to connect to the mysql server running as a sidecar.


## Architecture

Internally each sidecar is implemented using a `beast-agent` which runs as a process inside the container, this provides an interface to interact with the sidecar for configuration purposes. This agent exposes a RPC interface for other application(in this case beast) to trigger action and manage states.

In case of MySQL sidecar, the running agent exposes RPC to perform the following functions:

* Create a new database for a new user with a new password.
* Delete an existing database along with the user.

## Flow

The deployment of a challenge with a sidecar goes through the following flow:

* Stage the challenge for deployment.
* Commit the challenge image
* Identify the sidecar required and initialize a new instance for the challenge in sidecar. This action is done by making a RPC to beast-agent running inside the sidecar container.
	* For Example: In case of MySQL sidecar for the challenge a new database with a new user is created.
	* The details for this instance is also stored in the database for teardown purposes later on.
* Once the RPC returns with the parameters, we parse the parameters as Environment variables and store them in a file inside staging area.
* During the deploy stage of challenge deployment pipeline we run the container injecting the required environment variables inside the container.

Inside the container the user can then use the crednentials to interact the mysql server running in the sidecar container.

## Implementation

For implementation, since each sidecar will have different functions to perform we need different agents for different sidecars. For example a MySQL sidecar should expose RPC to create and delete database while a redis sidecar exposes function for creation and deletion of users. So for each sidecars we have an agenet implemented. These agents lies in `/extras/agents/` of the repository.

Each of these agenets are a GRPC servers implemented in a compiled language(Go for instance). Before we begin using these agents we should have the binary ready which can be run inside the containers, so compile your servers using `Makefile` inside extras, it will automatically place agents binarires in required directories.

Once we have agents as binaries we can start deploying sidecars. To deploy a sidecar make an API call to beast sidecar deployment endpoint(this will trigger the deployment). During deployment the agents are copied inside the sidecar and are run along with the sidecar as container entrypoint exposing the specified RPCs.

Communication of the challenge containers with the sidecar container is handled using docker networks. Whenever a sidecar is deployed a new network is created for it, for example mysql have `beast-mysql` network associated with it. Each challenge which then species this sidcar to be used is also associated with this network. Doing so provide complete observability between the two containers.

Using sidecar you have to pick your configuration variables from environment variables. For mysql sidecar example with php:

```php
<?php 

$dsn = "mysql:host=mysql;dbname=" . getenv("MYSQL_database") . ";charset=utf8mb4";
$options = [
  PDO::ATTR_EMULATE_PREPARES   => false, // turn off emulation mode for "real" prepared statements
  PDO::ATTR_ERRMODE            => PDO::ERRMODE_EXCEPTION, //turn on errors in the form of exceptions
  PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC, //make the default fetch be an associative array
];

try {
	$pdo = new PDO($dsn, getenv("MYSQL_username"), getenv("MYSQL_password"), $options);
} catch (Exception $e) {
	echo $e->getMessage();
}

echo "Success: A proper connection to MySQL was made! The my_db database is great." . PHP_EOL;

?>
```
