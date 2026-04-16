# Instanced Docker Compose Challenge Example

This is an example of an **instanced challenge using Docker Compose** — a multi-container challenge where each user gets their own isolated environment with a web server and database.

## Architecture

Beast treats the compose **service** named `ssh` as the primary instance container (users SSH into it; it also serves the web app here). The project name prefixes actual container names (for example `beast-instance-…-ssh-1`), so authors must keep the service key as `ssh`, not the literal container name.

```
┌──────────────────────────────────────────────────┐
│         User's instanced environment               │
│  ┌─────────────────────────┐      ┌────────────┐   │
│  │  ssh (PHP/Apache + SSH) │ ───▶ │    db      │   │
│  └─────────────────────────┘      └────────────┘   │
│        │                                           │
│        ▼                                           │
│   HTTP: INSTANCE_PORT → 80 (dynamic on host)     │
│   SSH:  INSTANCE_SSH_PORT → 22 (dynamic on host)   │
└──────────────────────────────────────────────────┘
```

## Key configuration

In `beast.toml`:

```toml
[challenge.metadata]
instanced = true

[challenge.env]
docker_compose = "docker-compose.yml"
default_port_var = "INSTANCE_SSH_PORT"
```

See `beast.toml` in this directory for full metadata (e.g. `instance_expiration`).

`default_port_var` selects which env-backed **host** port Beast treats as the primary instance port in API responses (here, SSH on the `ssh` service). HTTP is still published via `INSTANCE_PORT` → 80; both variables must appear in `docker-compose.yml` ports so Beast allocates a host port for each.

In `docker-compose.yml`, every published port must use an environment placeholder (no `${VAR:-default}` syntax). Example:

```yaml
services:
  ssh:
    ports:
      - "${INSTANCE_PORT}:80"
      - "${INSTANCE_SSH_PORT}:22"
```

## Challenge details

This is a SQL injection challenge:

1. The login form is vulnerable to SQL injection
2. Bypass authentication to login as admin
3. The flag is stored in the `secrets` table

### Solution

```
Username: admin' OR '1'='1' --
Password: anything
```

Or use UNION-based injection to extract data directly.

## Testing locally

```bash
cd _examples/instanced-compose
export INSTANCE_PORT=8080
export INSTANCE_SSH_PORT=2222
docker compose up -d --build
```

The web app is at `http://localhost:8080`. SSH matches `default_port_var`: `ssh -p 2222 beast@localhost` (adjust user/host as in your setup).

## Usage via Beast API

```bash
curl -X POST -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/instances/instanced-compose/spawn
```

The returned `port` is the allocated host port for **`INSTANCE_SSH_PORT`** (SSH), matching `default_port_var`. HTTP is bound separately via `INSTANCE_PORT` (another host port in the same compose up).

```bash
ssh -p <port from response> beast@<hosted_address>
```
