# Setting up Beast

## Build and start

Run Beast on a Linux host with Docker, the Docker Compose plugin, and access to PostgreSQL and Redis. Building from source requires Go 1.23 or newer. Install the PostgreSQL client tools for database maintenance commands.

From the repository root, build Beast and start the development dependencies.

```sh
make build
test -e .env || cp sample.env .env
# Edit .env to set credentials and connection details before starting the services.
make up NAME=myctf
```

The build installs `beast` in the directory reported by `go env GOBIN`, or in `bin` under the first entry reported by `go env GOPATH`. Add that directory to your `PATH`. The build also checks the host tools used for SSH to challenge containers.

```sh
beast config --non-interactive
# Review ~/.beast/config.toml. Set jwt_secret and replace any $HOME path placeholders.
beast init --non-interactive
beast run
```

`config` prepares the global config and the Beast authorized-keys file. `init` also prepares the data directories and checks Docker, PostgreSQL, and Redis. Non-interactive setup skips the administrative-user prompt. Use `beast create-admin` to create an administrator.

`make up` starts PostgreSQL and Redis. Use `make logs NAME=myctf` to see their logs and `make down NAME=myctf` to stop the Compose project. The API defaults to port 5005. Pass `beast run --port 5505` to change it.

## Global configuration

Start with the example config in `_examples/example.config.toml` or generate one with `beast config`. Beast reads `~/.beast/config.toml` and accepts environment overrides. It loads `.env` from the working directory and `~/.beast/.env`, in that order, for variables absent from the process environment.

| TOML setting | What it controls |
| --- | --- |
| `[psql_config]` | PostgreSQL host, port, user, password, database name, and SSL mode. Overrides use `POSTGRES_HOST`, `POSTGRES_PORT`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`, and `POSTGRES_SSLMODE`. |
| `[redis_config]` | Redis connection and ACL credentials. Overrides use `REDIS_HOST`, `REDIS_PORT`, `REDIS_USER`, `REDIS_PASSWORD`, `REDIS_DEFAULT_PASSWORD`, and `REDIS_DB`. |
| `jwt_secret` | Secret used to sign API tokens. Set a unique value for each deployment. |
| `[available_servers.<name>]` | Deployment host, SSH connection details for remote hosts, and a host `port_range` such as `10000:20000`. Ranges are inclusive and must stay within 10000–20000. |
| `[[remote]]` | Git repository URL, name, branch, and SSH key path when needed. Set `active = true` to enable it. |
| `[instance_config]` | Default expiration and maximum extension in seconds, plus the concurrent instance limit per user. |
| `[competition_info]` | Competition name, schedule, presentation, and scoring mode. Use the date and timezone format shown in the example config. |
| `[[notification_webhooks]]` | Active Slack or Discord webhooks for deployment events. |
| `[mail_config]` | SMTP host, port, sender, and optional password for email verification and password reset. Overrides use `SMTP_HOST`, `SMTP_PORT`, `SMTP_FROM`, and `SMTP_PASSWORD`. SMTP TLS verifies the server certificate. |
| `health_prober`, `ticker_frequency` | Enables periodic health checks and sets their interval in seconds. |
| `remote_sync_period` | Duration between Git syncs when `beast run --periodic-sync` is enabled. |
| `authorized_keys_file`, `scripts_dir` | Host authorized-keys file and directory for generated SSH scripts. |
| `beast_static_url` | Base URL for the optional standalone static server. |

For a host-installed Beast using the Compose dependencies, use `localhost` or `127.0.0.1` with their published ports. For Beast running on the same Compose network, use the service names `postgres` and `redis` with their container ports.

`LOG_HTTP_PAYLOADS=true` enables bounded text payload previews on eligible routes. Sensitive routes omit payloads. HTTP error metadata is logged with payload logging disabled.

## Docker host and static server

Local deployments use the Docker daemon selected by the environment or Docker context. When running Beast in a container, give it access to the daemon and set `BEAST_HOST_DIR` to the host path backing the mounted Beast data directory if the host and container paths differ.

Instanced Compose challenges require `hydra-net`. Run `scripts/provision/hydra-net-setup.sh` on the Docker host to provision the network and firewall rules. See the [challenge configuration reference](ChallConfig.md#instanced-compose-ssh).

Beast can serve staged public assets through its API. You can also run the separate Nginx image built by `make requirements`. The standalone server serves assets at `/static/<challenge>/<asset>`.

## Data and maintenance

PostgreSQL stores persistent competition and deployment data. Redis stores cache data, live instance state, and port reservations. Files under `~/.beast` include configuration, cloned remotes, staged challenges, public assets, and backups. Use the CLI to back up and restore PostgreSQL and Redis. Restore the matching staged files separately when you need to recover deployment artifacts.
