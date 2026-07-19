# Authentication

Beast uses password authentication and HMAC-signed JWT bearer tokens. There is no SSH-key challenge-response login and authorization cannot be disabled from the command line.

## Credentials

- Passwords must contain 12–128 bytes and cannot be whitespace-only.
- Contestant usernames are 3–12 lowercase letters, digits, dots, underscores, or hyphens.
- `jwt_secret` must contain at least 32 bytes and must be unique per deployment.
- Tokens expire after six hours. Password-reset tokens have a separate, restricted claim type.

Create the first administrator during `beast init`. Additional authors/admins can be created with the CLI on the controller host.

## Login

```bash
curl --cacert "$HOME/.beast/secrets/tls.crt" \
  --request POST https://localhost:5005/auth/login \
  --data-urlencode 'username=admin' \
  --data-urlencode 'password=<password>'
```

The response contains `token`, `role`, and `message`. Supply the token exactly as:

```text
Authorization: Bearer <token>
```

The CLI performs the same flow and verifies TLS:

```bash
beast getauth --host https://localhost:5005 \
  --ca-file "$HOME/.beast/secrets/tls.crt" \
  --username admin
```

Login attempts are rate-limited, unknown users receive the same public failure as bad passwords, and banned users cannot obtain tokens. Keep the API behind network access controls as rate limiting is not a substitute for perimeter protection.

## Roles

- `contestant`: competition and instance APIs.
- `author` / `maintainer`: challenge-management routes only for challenges they own or maintain; uploaded configuration email ownership is checked.
- `admin`: bulk/scheduled operations, controller-local path deployment, remote synchronization, configuration mutation, static-service management, user control, and administrative instance operations.

Authorization is checked both at route level and against challenge ownership for sensitive operations such as logs and container execution.

## Browser access

CORS is disabled unless `server.allowed_origins` is populated. Origins are exact and credentialed cross-origin requests are disabled. Never use a wildcard origin for an administrative frontend.
