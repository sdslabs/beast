# Beast documentation

Beast builds and operates CTF challenges across local or remote Docker workers through an authenticated HTTPS API.

Start here:

- [Setup](Setup.md): host, datastore, TLS, worker, and teardown requirements.
- [Usage](Usage.md): CLI and API operation.
- [Authentication](APIAuth.md): login, JWT, roles, and CORS.
- [Challenge configuration](ChallConfig.md): strict `beast.toml` schema and safety rules.
- [Challenge types](ChallTypes.md): static, service, web, bare/custom, and Compose models.
- [Architecture](Architecture.md): state, queues, workers, reconciliation, and trust boundaries.
- [Command reference](cmdref/beast.md): generated from the current binary.

The live server exposes generated Swagger documentation at `/api/docs/index.html` over HTTPS.

## Operator warning

Access to Docker is root-equivalent, and challenge build contexts contain executable organizer code. Run Beast on dedicated infrastructure, protect its configuration/keys, restrict management API reachability, and use verified TLS for every non-loopback dependency.
