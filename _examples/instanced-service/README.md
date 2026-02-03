# Instanced Service Challenge Example

This is an example of an **instanced challenge** - a challenge where each user gets their own dedicated container instance.

## Key Features

- **Per-user isolation**: Each user spawns their own container
- **Automatic expiration**: Instances expire after a configurable time (default: 5 minutes)
- **Dynamic port allocation**: Ports are assigned from a configured range (not from the challenge config)

## Configuration

In `beast.toml`, the key settings for instanced challenges are:

```toml
[challenge.metadata]
instanced = true              # Enable instancing
instance_expiration = 300     # Optional: override default expiration (in seconds)

[challenge.env]
# DO NOT specify ports for instanced challenges!
# Instead, use default_port to indicate which container port to expose
default_port = 9999
```

## Global Configuration

In your Beast `config.toml`, configure the instance settings:

```toml
[instance_config]
port_range_start = 30000      # Start of port range for instances
port_range_end = 40000        # End of port range for instances
default_expiration = 300      # Default TTL in seconds (5 minutes)
max_extension = 600           # Maximum extension time (10 minutes)
max_instances_per_user = 3    # Max concurrent instances per user
```

## API Usage

### User Endpoints

1. **Spawn an instance**:
   ```bash
   curl -X POST -H "Authorization: Bearer $TOKEN" \
     http://localhost:8080/api/instances/instanced-service/spawn
   ```

2. **Get your instance**:
   ```bash
   curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:8080/api/instances/instanced-service
   ```

3. **Get all your instances**:
   ```bash
   curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:8080/api/instances
   ```

4. **Extend instance lifetime**:
   ```bash
   curl -X POST -H "Authorization: Bearer $TOKEN" \
     -d "seconds=300" \
     http://localhost:8080/api/instances/instanced-service/extend
   ```

5. **Kill your instance**:
   ```bash
   curl -X DELETE -H "Authorization: Bearer $TOKEN" \
     http://localhost:8080/api/instances/instanced-service
   ```

### Admin Endpoints

1. **List all instances**:
   ```bash
   curl -H "Authorization: Bearer $ADMIN_TOKEN" \
     http://localhost:8080/api/admin/instances
   ```

2. **Kill any instance**:
   ```bash
   curl -X DELETE -H "Authorization: Bearer $ADMIN_TOKEN" \
     http://localhost:8080/api/admin/instances/{instance_id}
   ```

3. **Kill all instances for a challenge**:
   ```bash
   curl -X DELETE -H "Authorization: Bearer $ADMIN_TOKEN" \
     http://localhost:8080/api/admin/instances/challenge/instanced-service
   ```

## Response Example

When spawning an instance, you'll receive:

```json
{
  "instance_id": "a1b2c3d4e5f6",
  "challenge_name": "instanced-service",
  "hosted_address": "localhost",
  "port": 31234,
  "created_at": "2024-01-15T10:30:00Z",
  "expires_at": "2024-01-15T10:35:00Z",
  "ttl_seconds": 300
}
```

Connect to your instance:
```bash
nc localhost 31234
```

## Challenge Details

This example is a simple buffer overflow challenge:
- The `vulnerable()` function uses `gets()` which doesn't check bounds
- Overflow the 64-byte buffer to overwrite the return address
- Redirect execution to the `win()` function to get the flag
