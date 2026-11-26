# Authentication

Beast authenticates API users with a username and password and issues a signed JWT. Set a unique `jwt_secret` in `~/.beast/config.toml`. SSH public keys belong to the account or challenge author and support SSH access to challenge environments.

## Sign in

Send form fields `username` and `password` to `POST /auth/login`. Successful responses contain `token`, `role`, and `message`. Login tokens expire after six hours.

```sh
curl -fsS --data-urlencode "username=$BEAST_USERNAME" \
  --data-urlencode "password=$BEAST_PASSWORD" \
  http://localhost:5005/auth/login
```

The CLI can request the same token.

```sh
beast getauth --username "$BEAST_USERNAME" --password "$BEAST_PASSWORD" \
  --host http://localhost:5005/
```

Pass the returned token as `Authorization: Bearer <token>`. Deployment and other management routes require an author or administrator. Configuration changes, competition administration, Git remote management, and notification writes require an administrator.

## Register or recover an account

- Registration uses `POST /auth/register` with `username`, `password`, `email`, and `ssh-key`. The `name` field is optional.
- With SMTP configured and authorization enabled, first send a code with `POST /auth/send-otp` and verify it with `POST /auth/verify-otp`. Send `email` to the first route and `email` and `otp` to the second.
- Password recovery uses `POST /auth/send-otp-forget` and `POST /auth/verify-otp-forget` with the same fields. Send the verification token as a bearer token to `POST /auth/reset-password` with the `new_pass` field.

Email codes expire after five minutes. The recovery endpoint issues a token with a five-minute lifetime. Keep `beast run --noauth` for local development.
