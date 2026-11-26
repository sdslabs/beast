# Usage

Follow [setup](Setup.md) to build and configure Beast, then start the API with `beast run`. The server defaults to port 5005 and serves Swagger at `/api/docs/index.html`.

Use `--noauth` only for a local development server. On a server with authorization enabled, send a JWT in the `Authorization: Bearer <token>` header. See the [authentication guide](APIAuth.md).

## Manage a challenge

```sh
beast verify "$PWD/_examples/static-chall"
beast challenge deploy --local-directory "$PWD/_examples/static-chall"
beast chall-details --status deployed
beast challenge undeploy static-chall
```

For queued API actions, use the status endpoint to check when work has finished. The following examples use an author or administrator token.

```sh
curl -fsS -H "Authorization: Bearer $TOKEN" \
  --data-urlencode "challenge_dir=$PWD/_examples/static-chall" \
  http://localhost:5005/api/manage/deploy/local/

curl -fsS -H "Authorization: Bearer $TOKEN" \
  http://localhost:5005/api/status/challenge/static-chall
```

Use `GET /api/info/challenge/{name}` to look up allocated host ports for a deployed service. For per-user deployments, use `POST /api/instances/{challenge_name}/spawn` and use the host and port array returned for that instance.

To reload global configuration, send `PATCH /api/config/reload` with an administrator token. See the [CLI reference](cmdref/beast.md) for deployment, logs, backups, and restore commands.
