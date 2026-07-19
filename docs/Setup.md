# Setup

## Host assumptions

Beast is supported on Linux and requires Go 1.23+, Docker Engine, Git, Make, PostgreSQL, and Redis. Install them from trusted packages. `scripts/installenv.sh` verifies Go, Git, Make, and a usable Docker daemon; it deliberately does not install packages or alter system services.

Docker control is root-equivalent. Use a dedicated Beast account and host/VM, keep the management API private, and do not share that account with untrusted users.

## Build

```bash
git clone https://github.com/sdslabs/beastv4.git
cd beastv4
./scripts/installenv.sh
make build
beast version
```

Set `BEAST_OUTPUT=/absolute/path/beast` when a location other than `$(go env GOPATH)/bin/beast` is desired.

## PostgreSQL and Redis

Create or select PostgreSQL and Redis services before initialization. Beast can provision its application database/user and Redis ACL user, but it needs administrator credentials during `beast init`.

For loopback-only development, PostgreSQL may use `sslmode = "disable"` and Redis may use `tls = false`. For any non-loopback address:

- PostgreSQL must use `sslmode = "verify-full"` and `sslrootcert` must identify a readable CA bundle.
- Redis must use `tls = true`; configure `ca_file` when the service CA is not in system roots and `server_name` when it differs from the host.

Beast stores only its own keys under `beast:*` and creates a command-scoped Redis ACL. Do not reuse the configured application credentials as datastore administrator credentials.

## Initialize

The recommended interactive path is:

```bash
beast init
```

Initialization:

1. Creates `$HOME/.beast` and private subdirectories with mode `0700`.
2. Creates a mode-`0600` configuration and prompts for datastore, worker, resource, and competition settings.
3. Generates a localhost-only development certificate when the default TLS files are both absent.
4. Checks Docker, provisions the Redis ACL and PostgreSQL database, runs schema migrations, and optionally creates an administrator.

For a pre-generated configuration:

```bash
./setup.sh
$EDITOR "$HOME/.beast/config.toml"
beast init
```

`setup.sh` creates unique JWT/PostgreSQL/Redis secrets and builds Beast. It does not install, start, or reconfigure external services. It never overwrites an existing configuration.

For production, replace the generated localhost certificate with a certificate issued for the deployed hostname. The certificate may be public; the key must be a non-symlink regular file with mode `0600`.

## Important configuration controls

Use `_examples/example.config.toml` as the annotated reference.

- `config.toml` must be a non-symlink regular file with mode `0600`.
- `jwt_secret` must be unique and at least 32 bytes.
- `allowed_origins` is an explicit CORS allowlist. Use HTTPS except for loopback development.
- Default CPU, memory, PID, and CPU-count values are also maximum per-challenge overrides.
- Every active worker needs a non-overlapping host `port_range` on that worker.
- Remote workers require a mode-`0600` SSH key and a trusted `known_hosts` file. Beast never accepts an unknown host key automatically.
- Active Git remotes accept SSH URLs only and require a mode-`0600` key.
- Active webhooks must be official HTTPS Slack or Discord webhook URLs.

Remote SSH users need access to Docker on their worker. That permission is root-equivalent; use a dedicated account and key. Beast runs argument-quoted commands and stores remote staging data under the SSH user's `~/.beast` directory.

## Run

```bash
beast run --health-probe
```

The default listener is `https://0.0.0.0:5005`. Restrict it with host firewall/security-group rules or a trusted reverse proxy. API documentation is available at `https://<host>:5005/api/docs/index.html`.

Optional run controls include `--auto-deploy`, `--periodic-sync`, `--no-cache`, `--port`, and `--default-author-password-file`. The password file must be a non-symlink regular file with mode `0600` and contain 12–128 bytes.

## Static assets

Build the pinned unprivileged Nginx image with:

```bash
make requirements
```

Deploy it through the authenticated management API. Beast mounts only each challenge's normalized static directory read-only and publishes the container on host port `8034`. Put a TLS reverse proxy in front of that port and configure `beast_static_url` to the public HTTPS origin. No htpasswd file is used.

## Stop and remove local state

```bash
./scripts/teardown.sh
./scripts/teardown.sh --purge-data
```

Teardown obtains or verifies the controller lock before signaling a same-user Beast process. Normal shutdown preserves deployed challenges. `--purge-data` removes local `$BEAST_HOME` (default `$HOME/.beast`) only; PostgreSQL and Redis data are not removed.

Back up the PostgreSQL database and Redis state before destructive reset/restore commands. Those CLI commands require `--yes`.
