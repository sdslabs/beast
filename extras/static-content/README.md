# Static Content

This docker setup serves static content for beast. Mount the staging directory for beast at `/beast` of the docker container while running. It will automatically serve the static content for challenges which are staged.

Each challenge when staged will pull out the static content directory out of the challenge and put it inside staging area, this static content is then served using nginx service running in a container which mount the staging area as volume.

Build the docker image using

```bash
$ docker build . --tag beast-static:latest
```

To run the nginx powered static content serving container for beast run

```bash
$ docker run -d -p 80:8080 -v <beast-staging-directory>:/beast [IMAGE ID]
```
