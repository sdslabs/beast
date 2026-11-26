# Deployment

Beast can deploy a challenge from a local directory, an uploaded archive, or a configured Git remote. Each challenge needs a matching directory name and `beast.toml`. See [challenge configuration](ChallConfig.md).

- **Local** — use `beast challenge deploy --local-directory /path/to/challenge` or `POST /api/manage/deploy/local/` with `challenge_dir`.
- **Git** — configure active `[[remote]]` entries. Store challenges under `challenges/<name>` in each repository. An administrator can request `POST /api/remote/sync`, then an author or administrator can request deployment by name. The `--periodic-sync` and `--auto-deploy` server flags enable the corresponding background workflows.
- **Upload** — use `POST /api/manage/challenge/upload` with the challenge archive. See the Swagger reference for the upload fields.

The API acknowledges queued work. Poll `GET /api/status/challenge/{name}` for the deployment state. For a shared container or Compose project, check `GET /api/info/challenge/{name}` for host and port details. For an instanced challenge, deploy its definition first and then call the instance API.

Undeploy stops the workload and keeps staged build artifacts. Redeploy uses a source directory or staged artifacts. Purge removes staged artifacts and the challenge's competition records. See the [CLI reference](cmdref/beast_challenge.md) for flags and syntax.
