# Challenge configuration

Every challenge directory contains a strict TOML file named `beast.toml`. Unknown keys are rejected. Generate a parseable static scaffold with `beast new`, then validate changes with:

```bash
beast verify --local-directory /absolute/path/to/challenge
```

Challenge names must match `^[a-z0-9][a-z0-9._-]{0,63}$`. Referenced files/directories must be relative, regular entries that resolve inside the challenge root; symlinks and path traversal are rejected during validation/staging.

## Author and maintainers

```toml
[author]
name = "Author Name"
email = "author@example.com"

[[maintainer]]
name = "Maintainer Name"
email = "maintainer@example.com"
```

Email is required and must be canonical. Existing users identified by email become challenge managers; ownership is enforced by management, logs, and execution endpoints.

## Metadata

```toml
[challenge.metadata]
name = "example-web"
type = "web:php"
flag = "flag{replace-me}"
dynamicFlag = false
difficulty = "medium"
description = "Example challenge"
tags = ["web", "php"]
points = 500
minPoints = 100
maxPoints = 500
maxAttemptLimit = 0
preReqs = []
assets = ["download.zip"]
additionalLinks = ["https://example.invalid/rules"]
instanced = false
instance_expiration = 300

[[challenge.metadata.hints]]
text = "A bounded hint"
points = 50
```

`flag` may be empty only when `dynamicFlag = true`. `maxAttemptLimit = 0` means unlimited attempts. Point ranges must be internally consistent. Prerequisites must be valid challenge names, links must be HTTP(S), and each asset must exist beneath `static_dir` (default `public`).

Instanced challenges create per-user runtime instances. Expiration falls back to the global instance default when omitted; extensions and per-user counts are capped globally.

## Generated environments

Non-static generated challenge images use `[challenge.env]`:

```toml
[challenge.env]
ports = [8080]
default_port = 8080
apt_deps = []
setup_scripts = ["setup.sh"]
static_dir = "public"
base_image = "ubuntu:24.04"
run_cmd = "./server"
service_path = ""
web_root = "challenge"
entrypoint = ""
docker_context = ""
xinetd_conf = ""
traffic = "tcp"

[[challenge.env.var]]
key = "FLAG_FILE"
value = "secrets/flag"
```

Rules:

- One to three unique container ports in `1..65535` are allowed; `default_port` must be one of them.
- `traffic` is `tcp` or `udp`.
- `base_image` must be in the administrator allowlist.
- Setup scripts run at image-build time and therefore are trusted code.
- Environment `value` is a path to a file inside the challenge, not a literal secret. Beast reads the file and injects its content.
- `run_cmd` and `entrypoint` are mutually exclusive. An explicit entrypoint may run as the image user/root; use it only when required.
- `docker_context` names a Dockerfile inside the challenge. Custom Dockerfiles are trusted build code.

Static challenges need only `static_dir`; no port or runtime command is required.

## Compose environments

```toml
[challenge.env]
docker_compose = "docker-compose.yml"
default_port_var = "APP_PORT"
static_dir = "public"
```

Compose port bindings use variables assigned by Beast, for example `${APP_PORT}:8080`. Only port interpolation is accepted. The parser rejects unknown fields and security-sensitive runtime controls including privileged mode, host network/PID/IPC, devices, Docker socket access, unsafe mounts, namespace sharing, added capabilities, and arbitrary restart ownership. Build contexts and Dockerfiles must remain inside the challenge directory, and every service must comply with global resource ceilings.

When `docker_compose` is present, generated-image fields such as `run_cmd`, `entrypoint`, `base_image`, `apt_deps`, and `setup_scripts` are ignored.

## Resources

```toml
[resource]
cpu_shares = 256
cpuslimit = 0.20
memory_limit = 268435456
pids_limit = 64
```

Omitted or zero values inherit global defaults. A challenge may request less, never more, than the configured global ceilings.

## Archives and uploads

Uploaded ZIP/TAR content is bounded by entry count and expanded size. Absolute paths, `..`, duplicate paths, symlinks, devices, FIFOs, sockets, and overwrite attempts are rejected. Staging copies regular files only and never follows links.
