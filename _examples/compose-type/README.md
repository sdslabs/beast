# Shared Docker Compose example

The `docker_compose` field in `beast.toml` points to the Compose project for this PHP and MySQL challenge. Beast supplies the host port to the app through the `APP_PORT` environment variable.

From the repository root, deploy the definition using an author or administrator token.

```sh
curl -fsS -H "Authorization: Bearer $TOKEN" \
  --data-urlencode "challenge_dir=$PWD/_examples/compose-type" \
  http://localhost:5005/api/manage/deploy/local/

curl -fsS -H "Authorization: Bearer $TOKEN" \
  http://localhost:5005/api/status/challenge/compose-type
```

Once the status is `Deployed`, read `GET /api/info/challenge/compose-type` and open the host and allocated port returned in `deployedLink` and `ports`.
