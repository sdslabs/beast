# Challenge types

Set `type` in `[challenge.metadata]` and the deployment settings in `[challenge.env]`.

| Type | Deployment settings |
| --- | --- |
| `static` | Place public assets in the challenge directory or set `static_dir`. Beast stages the files and serves them through the asset API or a separate static server. |
| `service` | Set `service_path` to a relative executable path and provide a container port. Beast generates an xinetd configuration unless you provide `xinetd_conf`. |
| `bare` | Set `run_cmd` or `entrypoint` for a generated Dockerfile. Supply container ports and any build dependencies. |
| `web` and `web:<language>:<version>:<framework>` | Choose a web preset and provide `web_root` and container ports. Examples include `web:php:8.2:cli`, `web:node:20`, and `web:python:3.12:flask`. A plain `web` challenge without a custom build needs its own `run_cmd`. |

Set `docker_context` to use a custom Dockerfile or `docker_compose` to deploy a multi-service project. These are deployment settings under `[challenge.env]`. The metadata still uses a supported type such as `bare` or `web`.

Set `instanced = true` in the metadata to prepare a containerized challenge for per-user spawning. Instanced Compose has a separate SSH and networking contract. See the [configuration reference](ChallConfig.md).
