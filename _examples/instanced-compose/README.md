# Instanced Docker Compose Challenge Example

This is an example of an **instanced challenge using Docker Compose** — a multi-container challenge where each user gets their own isolated environment with a web server and database.

## Architecture

Beast treats the compose **service** named `ssh` as the primary instance container. The project name prefixes actual container names (for example `beast-instance-...-ssh-1`), so authors must keep the service key as `ssh`, not the literal container name.

```
┌──────────────────────────────────────────────────┐
│         User's instanced environment               │
│  ┌─────────────────────────┐      ┌────────────┐   │
│  │  ssh (PHP/Apache + SSH) │ ───▶ │    db      │   │
│  └─────────────────────────┘      └────────────┘   │
│        │                                           │
│        ▼                                           │
│   SSH: SSH_PORT -> 22 through hydra-net            │
└──────────────────────────────────────────────────┘
```

## Key configuration

In `beast.toml`:

```toml
[challenge.metadata]
instanced = true

[challenge.env]
docker_compose = "docker-compose.yml"
default_port_var = "SSH_PORT"
```

See `beast.toml` in this directory for full metadata (e.g. `instance_expiration`).

`default_port_var` selects which env-backed **host** port Beast treats as the primary instance port in API responses. For hydra-net challenges this must be the SSH mapping on the `ssh` service.

In `docker-compose.yml`, every published port must use an environment placeholder (no `${VAR:-default}` syntax). Example:

```yaml
services:
  ssh:
    ports:
      - "${SSH_PORT}:22"
    networks:
      - exposed
      - internal

  db:
    networks:
      - internal

networks:
  exposed:
    external: true
    name: hydra-net
  internal:
    driver: bridge
    internal: true
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
export SSH_PORT=2222
docker compose up -d --build
```

SSH matches `default_port_var`: `ssh -p 2222 beast@localhost` (adjust user/host as in your setup). The database is reachable only through the internal compose network.

## Usage via Beast API

```bash
curl -X POST -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/instances/instanced-compose/spawn
```

The returned `port` is the allocated host port for **`SSH_PORT`** (SSH), matching `default_port_var`.

```bash
ssh -p <port from response> beast@<hosted_address>
```
