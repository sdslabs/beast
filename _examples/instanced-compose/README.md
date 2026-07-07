# Instanced Docker Compose Challenge Example

This is an example of an **instanced challenge using Docker Compose** - a multi-container challenge where each user gets their own isolated environment with a web server and database.

## Architecture

```
┌──────────────────────────────────────────┐
│         User's Instanced Environment      │
│  ┌─────────────┐      ┌─────────────┐    │
│  │  PHP/Apache │ ───▶ │   MySQL     │    │
│  │    (web)    │      │    (db)     │    │
│  └─────────────┘      └─────────────┘    │
│        │                                  │
│        ▼                                  │
│   Port: 31234 (dynamically assigned)     │
└──────────────────────────────────────────┘
```

## Key Configuration

In `beast.toml`:

```toml
[challenge.metadata]
instanced = true
instance_expiration = 600  # 10 minutes

[challenge.env]
docker_compose = "docker-compose.yml"
default_port = 8080
```

In `docker-compose.yml`, use the `INSTANCE_PORT` environment variable:

```yaml
services:
  web:
    ports:
      - "${INSTANCE_PORT:-8080}:80"
```

## Challenge Details

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

## Testing Locally

```bash
# Build and run locally (for testing)
cd _examples/instanced-compose
docker-compose up -d

# Access at http://localhost:8080
```

## Usage via Beast API

```bash
# Spawn your instance
curl -X POST -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/instances/instanced-compose/spawn

# Response:
# {
#   "instance_id": "abc123def456",
#   "challenge_name": "instanced-compose",
#   "hosted_address": "localhost",
#   "port": 31234,
#   "expires_at": "2024-01-15T10:40:00Z",
#   "ttl_seconds": 600
# }

# Access your instance
open http://localhost:31234
```
