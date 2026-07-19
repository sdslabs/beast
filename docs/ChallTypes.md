# Challenge types

## Static

`type = "static"` publishes files from `static_dir` (default `public`) through the shared unprivileged static service. Static challenges do not receive a runtime container or port.

## Service

`type = "service"` hosts a program through xinetd. Set `service_path` to a regular executable/script inside the challenge and provide one or more `ports`. An optional `xinetd_conf` replaces the generated service configuration.

## Web

Web types begin with `web`, such as the supported PHP/Python/Node variants returned by configuration helpers. Set `web_root` and ports, or provide a validated `docker_context`/`docker_compose` for a custom web stack.

## Bare/custom

`type = "bare"` uses a generated base image and requires `run_cmd` or `entrypoint`. Generated `run_cmd` execution uses the unprivileged challenge user. An explicit entrypoint controls the whole container and may execute with its image user/root privileges, so it should be treated like a custom Dockerfile.

## Custom Dockerfile

Set `docker_context` to the Dockerfile path inside the challenge. Beast validates containment and applies build/run resource controls, but Dockerfile instructions are trusted build code with Docker-daemon impact.

## Docker Compose

Set `docker_compose` to a Compose file inside the challenge. Beast accepts a constrained service schema, validates resource ceilings and build contexts, allocates port variables, and rejects host-level privilege controls. See [Challenge configuration](ChallConfig.md#compose-environments).

## Instanced challenges

Any supported runtime type may set `instanced = true`. Beast then creates isolated per-user instances with Redis-backed expiry, bounded extension, and administrator cleanup endpoints. Static-only challenges are not meaningful as instanced runtimes.
