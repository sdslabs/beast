# Instanced Compose example

This challenge creates a per-user PHP/MySQL Compose project. Beast substitutes the allocated `${INSTANCE_PORT}` in `docker-compose.yml`; no default-value or arbitrary environment interpolation is accepted.

Validate and deploy the challenge definition:

```bash
beast verify --local-directory "$PWD/_examples/instanced-compose"
beast challenge deploy --local-directory "$PWD/_examples/instanced-compose"
```

Spawn an instance through the TLS API:

```bash
curl --cacert "$HOME/.beast/secrets/tls.crt" \
  --header "Authorization: Bearer $TOKEN" \
  --request POST \
  https://localhost:5005/api/instances/instanced-compose/spawn
```

Open `http://<hosted_address>:<port>` from the JSON response. Challenge traffic is plain HTTP in this example; the Beast management API remains HTTPS.

The application intentionally contains SQL injection and public sample credentials/flags. Do not deploy it on production infrastructure.
