# Usage

All API traffic uses HTTPS. Examples below trust the development certificate explicitly:

```bash
export BEAST_URL=https://localhost:5005
export BEAST_CA="$HOME/.beast/secrets/tls.crt"
```

## Authenticate

```bash
beast getauth --host "$BEAST_URL" --ca-file "$BEAST_CA" --username <username>
```

Or call the login endpoint and extract the returned token:

```bash
curl --cacert "$BEAST_CA" --request POST "$BEAST_URL/auth/login" \
  --data-urlencode 'username=<username>' \
  --data-urlencode 'password=<password>'
```

Send `Authorization: Bearer <token>` on protected routes. Never place passwords or tokens in URLs, shell history, source files, or logs.

## CLI challenge operations

```bash
beast challenge show --all
beast challenge deploy <challenge-name>
beast challenge undeploy <challenge-name>
beast challenge redeploy <challenge-name>
beast challenge purge <challenge-name>
```

Bulk operations accept `--all` or `--tag`. Controller-local deployment accepts `beast challenge deploy --local-directory /absolute/path`; validate it first with `beast verify --local-directory /absolute/path`. The HTTP equivalents for bulk, scheduling, static-service, and controller-local path operations are administrator-only. Authors/maintainers may operate only challenges they own.

Worker failures are returned by the CLI after queued work completes. A successful enqueue is not reported as a successful deployment.

## API examples

```bash
curl --cacert "$BEAST_CA" \
  --header "Authorization: Bearer $BEAST_TOKEN" \
  "$BEAST_URL/api/status/all"

curl --cacert "$BEAST_CA" \
  --header "Authorization: Bearer $BEAST_TOKEN" \
  --request POST "$BEAST_URL/api/manage/challenge/" \
  --data-urlencode 'action=deploy' \
  --data-urlencode 'name=my-challenge'
```

Use Swagger at `$BEAST_URL/api/docs/index.html` for the current route and payload contract. Management endpoints require author/maintainer or administrator roles; remote, configuration, user-control, and static-service operations require an administrator.

## Maintenance

Database and cache backup/reset/restore commands initialize the same validated runtime configuration as the server. Reset and restore are destructive and require `--yes`. Backups are explicit operations; health checks and shutdown do not perform synchronous backups.

Controller shutdown stops its workers, scheduler, subscriptions, and datastore connections while preserving deployed workloads. On restart, Beast reconciles runtime state and expired instances.
