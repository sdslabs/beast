# Instanced service example

Each authenticated user receives a dedicated service container and a host port allocated from the selected worker's `available_servers.<name>.port_range`. `default_port = 9999` identifies the container port; the global `[instance_config]` controls default expiration, maximum extension, and per-user counts.

```bash
export BEAST_URL=https://localhost:5005
export BEAST_CA="$HOME/.beast/secrets/tls.crt"

curl --cacert "$BEAST_CA" \
  --header "Authorization: Bearer $TOKEN" \
  --request POST "$BEAST_URL/api/instances/instanced-service/spawn"

curl --cacert "$BEAST_CA" \
  --header "Authorization: Bearer $TOKEN" \
  "$BEAST_URL/api/instances/instanced-service"

curl --cacert "$BEAST_CA" \
  --header "Authorization: Bearer $TOKEN" \
  --request POST --data-urlencode 'seconds=300' \
  "$BEAST_URL/api/instances/instanced-service/extend"

curl --cacert "$BEAST_CA" \
  --header "Authorization: Bearer $TOKEN" \
  --request DELETE "$BEAST_URL/api/instances/instanced-service"
```

The spawn response supplies `hosted_address`, `port`, and expiration data. Connect with `nc <hosted_address> <port>`.

Administrator instance routes are documented in the live Swagger UI. This example contains a deliberately vulnerable program and a public test flag; use a disposable worker.
