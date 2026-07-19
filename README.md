# Beast

Beast is a Linux service for building, deploying, and operating jeopardy-style CTF challenges. It exposes an HTTPS API and CLI, stores durable state in PostgreSQL, uses Redis for coordination and instance expiry, and deploys challenges through local or remote Docker daemons.

## Security model

Beast executes organizer-supplied challenge build contexts and controls Docker. Docker socket access and membership in the Docker group are effectively root-equivalent. Run Beast on a dedicated host or VM, use a dedicated unprivileged account, restrict management API access, and treat challenge authors as trusted build-code contributors. Containers reduce risk but are not a security boundary against a hostile kernel exploit.

The controller requires TLS. Non-loopback PostgreSQL connections require `sslmode = "verify-full"` and a CA file; non-loopback Redis connections require TLS. SSH workers verify `known_hosts`, and private keys/configuration files must be regular files with mode `0600`.

## Requirements

- Linux
- Go 1.23 or newer
- Docker Engine with a reachable daemon
- Git and Make
- PostgreSQL
- Redis with ACL support

Use trusted operating-system packages. The setup scripts do not install system packages or pipe remote scripts into a shell.

## Install and initialize

```bash
git clone https://github.com/sdslabs/beastv4.git
cd beastv4
./scripts/installenv.sh
make build
beast init
```

`beast init` creates private state under `$HOME/.beast`, generates or validates the TLS certificate and configuration, provisions the configured PostgreSQL database and Redis ACL user, and can create the first administrator. It prompts before privileged datastore operations.

For a non-interactive filesystem/bootstrap starting point, run `./setup.sh`, review the generated `$HOME/.beast/config.toml`, then run `beast init`. The setup script generates unique JWT, PostgreSQL, and Redis secrets but does not install or start those services.

Start the controller:

```bash
beast run --health-probe
```

The default endpoint is `https://localhost:5005`. For a locally generated certificate, pass its CA/certificate explicitly to clients. Do not disable TLS verification.

```bash
beast getauth --host https://localhost:5005 \
  --ca-file "$HOME/.beast/secrets/tls.crt" \
  --username <admin-username>
```

## Configuration

The complete annotated example is [`_examples/example.config.toml`](_examples/example.config.toml). Important rules:

- `$HOME/.beast/config.toml` must be a non-symlink regular file with mode `0600`.
- `jwt_secret` must contain at least 32 bytes.
- TLS certificate/key paths are mandatory; the private key must be mode `0600`.
- Active remote workers need a mode-`0600` SSH key and a populated `known_hosts` file.
- Resource defaults are hard ceilings for per-challenge overrides.
- CORS origins must be explicit HTTPS origins (loopback HTTP is accepted for development only).

## Challenge workflow

Create a strict, parseable static-challenge scaffold in an empty directory:

```bash
mkdir my-challenge && cd my-challenge
beast new
```

Edit `beast.toml`, place downloadable files in `public/`, and validate before deployment:

```bash
beast verify --local-directory "$PWD"
```

Controller-local path deployment (`beast challenge deploy --local-directory …` and its API equivalent) is administrator-only. Authors can upload a bounded ZIP whose `author`/`maintainer` email matches their account, or manage synchronized challenges they own.

Challenge names, referenced files, Compose build contexts, setup scripts, assets, and environment-value files are validated and must remain inside the challenge directory. Compose files accept a constrained schema and only port-variable interpolation; privileged, host-network, host-PID/IPC, device, socket, and unsafe mount controls are rejected.

See the [challenge configuration guide](docs/ChallConfig.md) and [examples](_examples/README.md).

## Development

```bash
make check_format
go vet ./...
go test ./...
go test -race ./...
make build
```

PostgreSQL concurrency tests run when `BEAST_TEST_PG_DSN` is set. Redis integration tests run when `BEAST_TEST_REDIS_ADDR` and related credentials are set. The example deployment harness is destructive and opt-in:

```bash
BEAST_RUN_INTEGRATION=1 make integration-test
```

Build documentation with pinned Python dependencies from `requirements.txt`:

```bash
python3 -m venv .venv
. .venv/bin/activate
pip install --requirement requirements.txt
make cmdref
make docs
```

## Teardown

`scripts/teardown.sh` verifies the controller lock and process ownership before sending `SIGTERM`. It does not undeploy challenges or delete external PostgreSQL/Redis data.

```bash
./scripts/teardown.sh              # stop only
./scripts/teardown.sh --purge-data # also remove $BEAST_HOME or $HOME/.beast
```

The purge option is intentionally destructive for local Beast state and refuses unsafe target paths.

## License

[Apache License 2.0](LICENSE.md)
