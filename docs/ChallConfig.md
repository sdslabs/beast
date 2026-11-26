# Challenge configuration

Each challenge has a `beast.toml` at its root. Its directory name must match `challenge.metadata.name`. This example uses the files in a `public/` directory.

```toml
[author]
name = "Challenge Author"
email = "author@example.test"
ssh_key = "<author public key>"

[challenge.metadata]
name = "example-static"
type = "static"
flag = "beast{replace_this}"
description = "An example static challenge"
points = 100

[challenge.env]
static_dir = "public"
```

## Metadata

The author requires `email` and `ssh_key`. Add `[[maintainer]]` entries with the same fields when needed.

Challenge metadata includes `name`, `type`, `flag`, `description`, `difficulty`, `points`, and `tags`. Set `dynamicFlag = true` for a dynamic flag challenge. `maxAttemptLimit` limits failed attempts. Omitting it or setting it to zero leaves attempts unlimited. Use `maxPoints` and `minPoints` to configure the scoring range for dynamic scoring.

Use `preReqs` for prerequisite challenge names, `assets` for asset references, and `additionalLinks` for related URLs. Add hints through `[[challenge.metadata.hints]]` entries with `text` and `points`.

Set `instanced = true` for a per-user challenge and `instance_expiration` to override the server's default lifetime in seconds.

## Container deployment

| Field | Behavior |
| --- | --- |
| `ports` | Container ports for a non-Compose challenge, such as `[8080]`. Beast allocates the host ports from the selected server's `port_range`. The first container port is the primary port for configurations that need one. |
| `apt_deps`, `setup_scripts` | Packages and relative scripts used when Beast builds a generated image. The generated image runs `post-build.sh`, when present, at container startup. |
| `run_cmd`, `entrypoint` | Startup command or relative entrypoint for a generated Dockerfile. Configure one of them. |
| `service_path`, `xinetd_conf` | Relative executable and optional xinetd config for a service challenge. |
| `web_root` | Relative root of a web application. |
| `static_dir` | Relative public assets directory. Defaults to `public/` for containerized challenges. Static challenges use their root when this is absent. |
| `base_image` | Base image for a generated image, subject to the server's allowlist. |
| `docker_context` | Relative path to a custom Dockerfile. |
| `docker_compose` | Relative path to a Compose file. Publish host ports using environment variables such as `${APP_PORT}:8080`. Beast supplies the host values. |
| `traffic` | Port publishing protocol, such as `tcp` or `udp`. |

Add environment variables through `[[challenge.env.var]]` entries with `key` and `value`. Set container resource overrides in `[resource]` using `cpu_shares`, `memory_limit` in bytes, `pids_limit`, and `cpuslimit`.

## Instanced Compose SSH

An instanced Compose challenge requires a service named `ssh` with exactly one published mapping such as `${SSH_PORT}:22`. Connect that service to the external `hydra-net` and to an internal bridge network. Other services use the internal network and keep their ports within the project. The SSH service needs a project-managed named volume at `/challenge` and a checker script at `/challenge/check.sh`.

Set `docker_compose` and `instanced = true` in `beast.toml`. Beast reads the SSH port variable from the Compose mapping. Provision Hydra on the Docker host as described in [setup](Setup.md#docker-host-and-static-server). Use the first entry in the instance response's `port` array to reach the SSH service.
