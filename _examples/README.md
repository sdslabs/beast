# Examples

Run `make test-examples` from the repository root to validate these configurations. Runtime tests require the images, host networking, and services used by each challenge.

### Configuration file samples:

* [Beast global configuration sample](./example.config.toml)
* [Beast static container authentication file](./.static.beast.htpasswd)

### Sample Challenges 

* [Simple Challenge - Bare](./simple)
* [PHP Web challenge](./web-php)
* [PHP Web challenge with MySQL](./web-php-mysql)
* [Challenge with Static files only](./static-chall)
* [xinetd Service challenge with custom xinetd config](./xinetd-service)
* [Service challenge with auto-generated xinetd config](./service)
* [A bare challenge using a custom Dockerfile](./bare-docker)
* [Docker compose challenge](./compose-type)
* [Instanced service challenge](./instanced-service)
* [Instanced Docker compose challenge](./instanced-compose)

From the repository root, use an author or administrator token to deploy a challenge definition. Replace `<challenge_name>` with its directory name.

```bash
curl -fsS -H "Authorization: Bearer $TOKEN" \
	--data-urlencode "challenge_dir=$PWD/_examples/<challenge_name>" \
  http://localhost:5005/api/manage/deploy/local/
```

Poll `GET /api/status/challenge/{name}` for the result. For a shared service, find the allocated host ports in `GET /api/info/challenge/{name}`. For an instanced challenge, spawn an instance after deploying its definition.
