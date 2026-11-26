# Sample challenges

From the repository root, run `make test-examples` against a configured Beast install. It validates the ten bundled `beast.toml` files. Use the [getting-started guide](GettingStarted.md) for a full local deployment.

| Directory under `_examples/` | Configuration it demonstrates |
| --- | --- |
| `static-chall` | Static files staged by Beast and served through the asset API. |
| `simple` | Bare challenge built from a generated image. |
| `service` | Service with generated xinetd configuration. |
| `xinetd-service` | Service with its own xinetd configuration. |
| `web-php` | PHP CLI web preset. |
| `web-php-mysql` | PHP and MySQL in a Compose project. |
| `bare-docker` | Bare challenge with a custom Dockerfile. |
| `compose-type` | Shared Compose deployment with an allocated host port. |
| `instanced-service` | A single container spawned per user. |
| `instanced-compose` | Per-user Compose project using SSH, Hydra, and a checker. |

`beast verify` checks a challenge's configuration and required files. Runtime checks require the images, services, and host setup relevant to each example. For a deployed shared service, use `beast chall-details` or `GET /api/info/challenge/{name}` to find the allocated host ports. For an instanced challenge, spawn an instance and use its returned `hosted_address` and `port` array.
