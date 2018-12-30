# MySQL Sidecar

This is the Dockerfile to build mysql 8.0 to be used as a sidecar by the challenge container. The base image is mysql 8.0 and the docker-entrypoint.sh file is taken from the official docker mysql docker library.

We first run the `entrypoint.sh` which runs the agent on the container as a background process and then we run the mysql docker entrypoint script.

Building the image

```bash
$ docker build . --tag beast-mysql:latest
```

Create a new network for your sidecar

```bash
$ docker network create beast-mysql
```

Running the container

```bash
$ docker run -d -p 127.0.0.1:9500:9500 --name mysql --network beast-mysql --env MYSQL_ROOT_PASSWORD=$(openssl rand -hex 20) beast-mysql
```
