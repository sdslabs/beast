# Static Content

This docker setup serves static content for beast. Mount the staging directory for beast at `/beast` of the docker container while running. It serves staged public assets at `/static/<challenge>/<asset>`. Beast also exposes public assets through `GET /api/info/download`.

Each challenge when staged will pull out the static content directory out of the challenge and put it inside staging area, this static content is then served using nginx service running in a container which mount the staging area as volume.

Build the docker image using

```bash
$ docker build . --tag beast-static:latest
```

To run the nginx powered static content serving container on port 8034 (by standard) for beast, run

```bash
$ export BEAST_STATIC_PORT=8034 
$ docker run -d -p $BEAST_STATIC_PORT:80 -v <beast-staging-directory>:/beast -v <beast-htpasswd-file>:/.static.beast.htpasswd beast-static
```

The staging virtual host uses HTTP Basic Authentication. Use `htpasswd` from `apache2-utils` to create its credentials file.

```bash
$ sudo apt-get install -y apache2-utils

$ htpasswd -c .static.beast.htpasswd <username>
<password>

$ mv .static.beast.htpasswd ~/.beast/
```
