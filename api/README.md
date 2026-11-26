# Beast API reference

On a running Beast server, open `/api/docs/index.html`. The generated spec is at `/api/docs/doc.json`. In Swagger's authorization dialog, enter the complete value `Bearer <token>` for protected operations. The [authentication guide](../docs/APIAuth.md) describes how to get a token.

## Regenerate Swagger

Update the handler annotations when changing routes, form fields, response schemas, or authorization. Install the version of Swag used by this checkout.

```sh
go install github.com/swaggo/swag/cmd/swag@v1.16.4
make swagger
make docs
```

Run the Make targets from the repository root. `make swagger` updates the embedded Go spec and the JSON and YAML copies used by the documentation site. Set `SWAG=/path/to/swag` if the generator is installed outside Go's binary directory. `make docs` builds the standalone Swagger page as `site/apidocs.html`. Use Swagger on a running Beast server to send live requests.
