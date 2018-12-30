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
* [Xinted Service challenge](./xinetd-service)

To test any of the above challenges, cd to \_example directory and use the below command:

```bash
$ curl -X POST localhost:5005/api/manage/deploy/local/ \
	--data "challenge_dir=$PWD/<challenge_name>"
```
