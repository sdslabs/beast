# Getting started

1. Follow [Setup](Setup.md) to build Beast, configure TLS/PostgreSQL/Redis, and create an administrator.
2. Start the controller with `beast run --health-probe`.
3. Authenticate with `beast getauth --host https://localhost:5005 --ca-file "$HOME/.beast/secrets/tls.crt" --username <admin>`.
4. Create an empty challenge directory and run `beast new` inside it.
5. Edit the generated `beast.toml` and place static files in `public/`.
6. Run `beast verify --local-directory "$PWD"`.
7. As an administrator on the controller, deploy locally with `beast challenge deploy --local-directory "$PWD"`; otherwise commit the directory beneath `challenges/` in a configured SSH Git remote or upload a bounded ZIP through the API.

Use lowercase challenge names containing only letters, digits, dots, underscores, and hyphens. Do not include symlinks, device files, sockets, or paths outside the challenge root.

The generated scaffold is a static challenge. For service, web, custom Dockerfile, Compose, and instanced variants, see [Challenge types](ChallTypes.md), [Challenge configuration](ChallConfig.md), and the repository's `_examples/` directory.
