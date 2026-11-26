# Examples

> This directory contains a few challenges example for beast, which are properly tested and should work out of the box.

### Configuration file samples:

* [Beast global configuration sample](./example.config.toml)
* [Beast static container authentication file](./.static.beast.htpasswd)

### Sample Challenges 

* [Simple Challenge - Bare](./simple)
* [PHP Web challenge](./web-php)
* [PHP Web challenge with MySQL](./web-php-mysql)
* [Challenge with Static files only](./static-chall)
* [Xinted Service challenge with custom xinetd config](./xinetd-service)
* [Service challenge with auto-generated xinetd config](./service)
* [A bare challenge using a custom Dockerfile](./bare-docker)
* [Docker compose challenge](./compose-type)
* [Instanced service challenge](./instanced-service)
* [Instanced Docker compose challenge](./instanced-compose)

To test any of the above challenges, cd to `_examples` directory and use the below command (pass `Authorization: Bearer <token>` if auth is enabled):

```bash
$ curl -X POST localhost:5005/api/manage/deploy/local/ \
	--data "challenge_dir=$PWD/<challenge_name>"
```
