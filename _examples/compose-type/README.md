# Compose example (PHP + MySQL)

This example uses Beast's constrained Compose schema. `${APP_PORT}` is allocated by Beast and mapped to the web container. Both services declare CPU, memory, PID, capability, and `no-new-privileges` controls.

```bash
beast verify --local-directory "$PWD/_examples/compose-type"
beast challenge deploy --local-directory "$PWD/_examples/compose-type"
```

Use `beast challenge show compose-type` to obtain the selected worker and allocated port. The challenge application itself serves plain HTTP; do not confuse it with the HTTPS Beast management API.
