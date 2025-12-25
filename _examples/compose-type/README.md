# Docker Compose example (PHP + MySQL)

This example demonstrates a multi-container challenge deployed via Docker Compose.

- `beast.toml` enables Compose mode using `docker_compose = "docker-compose.yml"`.
- `docker-compose.yml` defines an `app` container and a `mysql` container.

Deploy locally:

```bash
curl -X POST localhost:5005/api/manage/deploy/local/ \
  --data "challenge_dir=$PWD/_examples/compose-type"
```

After deployment, open:

- http://localhost:10020
