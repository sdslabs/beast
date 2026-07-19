# Examples

These directories demonstrate Beast challenge formats. They contain public test flags and intentionally weak sample services; run them only on disposable development workers.

- `static-chall`: static-only content.
- `service`, `xinetd-service`: service challenges.
- `web-php`, `web-php-mysql`: generated web challenges.
- `bare-docker`: custom Dockerfile.
- `compose-type`: constrained Compose deployment.
- `instanced-service`, `instanced-compose`: per-user instances.
- `simple`: generated bare environment.

Validate before deployment:

```bash
beast verify --local-directory "$PWD/_examples/service"
```

Then deploy with the local CLI path:

```bash
beast challenge deploy --local-directory "$PWD/_examples/service"
```

Both commands load the operator's validated `$HOME/.beast/config.toml`. Local deployment is an administrator/controller-host workflow and still uses the configured Docker worker, resource ceilings, PostgreSQL, and Redis.
