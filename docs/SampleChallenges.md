# Sample challenges

The maintained examples live in `_examples/`:

- `static-chall`: shared static service.
- `service` and `xinetd-service`: generated/custom xinetd services.
- `web-php` and `web-php-mysql`: generated web environments.
- `bare-docker`: custom Dockerfile challenge.
- `compose-type`: constrained multi-service Compose challenge.
- `instanced-service` and `instanced-compose`: per-user runtime instances.

Validate an example before deployment:

```bash
beast verify --local-directory "$PWD/_examples/service"
```

Deploying examples executes their setup scripts or Docker build instructions. Review them as code and use a disposable development worker. Example flags and passwords are intentionally public and must never be reused in a competition.

Compose examples use `${VARIABLE}:container-port` mappings. Beast supplies host ports from the selected worker's configured range; hard-coded host ports and arbitrary interpolation are not supported.
