# Static Content

This docker setup serves static content for beast. Mount the staging directory for beast at `/beast` of the docker container while running. It will automatically serve the static content for challenges which are staged.

Each challenge when staged will pull out the static content directory out of the challenge and put it inside staging area, this static content is then served using nginx service running in a container which mount the staging area as volume.

Build the docker image using

```bash
$ docker build . --tag beast-static:latest
```

Beast deploys the image on host port 8034 through the authenticated management API. It mounts only each challenge's public static directory read-only; do not mount the complete Beast staging directory.

```bash
$ make requirements
$ beast run
$ curl --cacert <ca-file> -H 'Authorization: Bearer <admin-token>' \
    -X POST https://localhost:5005/api/manage/static/deploy
```
