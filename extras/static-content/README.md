# Static Content

This docker setup serves static content for beast. Mount the staging directory for beast at `/beast` of the docker container while running. It will automatically serve the static content for challenges which are staged.

Each challenge when staged will pull out the static content directory out of the challenge and put it inside staging area, this static content is then served using nginx service running in a container which mount the staging area as volume.

Build the docker image using

```bash
$ docker build . --tag beast-static:latest
```

To run the nginx powered static content serving container for beast run

```bash
$ docker run -d -p 80:80 -v <beast-staging-directory>:/beast -v <beast-htpasswd-file>:/.static.beast.htpasswd beast-static
```

For authentication purposes you should create a htpasswd file using apache2-utils. First install apache2-utils and then create a htpasswd file

```bash
$ sudo apt-get install -y apache2-utils

$ htpasswd -c .static.beast.htpasswd <username>
<password>

$ mv .static.beast.htpasswd ~/.beast/
```
