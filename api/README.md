# Beast API documentation

Regenerate the embedded Swagger contract with the pinned module version:

```bash
go run github.com/swaggo/swag/cmd/swag@v1.16.4 init \
  --generalInfo main.go --dir api --output api/docs --parseDependency
```

Commit `api/docs/docs.go`, `swagger.json`, and `swagger.yaml` together with annotation changes. The running HTTPS server publishes the result at `/api/docs/index.html`.
