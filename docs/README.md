# Beast Documentation

This directory contains documentation related to beast and will guide you through flow, architecture usage and gotchas of beast.

## Index

* [Usage](/Usage)
* [Architecture](Architecture)
* [Authentication Flow](Auth)

## Intro

Beast is a service that runs on your host(may be a bare metal server or a cloud instance) and helps in the mangement of deployment, lifecycle and health check of CTF challenges. Beast is created to automate and ease the deployment procedure of challenges for a Jeopardy style CTF competition. As of now beast support the following type of challenges:

* Service - A service hosted on beast container instance
* Web - Web based challenges for various languages including PHP, Python, Node.js etc.
* Static - Challenges with static files, this may include forensics challenges.

## Tech Stack

Beast is written completely in Golang and comes with a clean REST API interface to trigger actions or interact with underlying functionalities.
The REST API server is implemented using `gin` go library and uses JWT as an authentication mechanism. Being written in go, Beast is compiled into
a single binary which can run on any linux distribution.

Beast uses Docker as a container runtimes to run challenges in a sandboxed environment. Note that container does not provide a very strong isolation, but our host is safe as long as there is no 0-day in linux kernel itself. Even though container provide a security layer for the challenges, we follow some practices to harden those security measures.

We use Swagger for automatic generation of API documentation and you can find the docs at `/api/docs/index.html` from beast server root.

To save the state of the deployments and challenges beast uses SQLite as a database, all the information ranging from challenge deployment state to allocated ports and author information is stored in this database. This database is created automatically in the root of your beast configuration directory.

### Note

* To run challenges in a highly secure mode you can change the runtime for docker(by default it is runc) to gVisor(runsc) which provides a much better security layer in the containerized sandboxed environment.
