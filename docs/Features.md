# Features

- Deploy a challenge from a local directory or configure Git remotes as the source of challenge definitions. Beast can sync them on demand and periodically.
- Build images and manage Docker or Compose workloads, including shared services and per-user instances with expiring leases.
- Publish static assets through the API or the optional Nginx static server.
- Use the CLI or HTTP API to deploy, check status, view logs, and administer a competition.
- Support competition registration, JWT login, optional email verification and recovery, flag submissions, hints, attempt limits, dynamic scoring, and leaderboard administration.
- Run periodic health checks and send deployment notifications through configured Slack and Discord webhooks. The API also streams competition announcements through server-sent events.
- Back up and restore PostgreSQL and Redis through the CLI.

See [setup](Setup.md) for host requirements and [challenge types](ChallTypes.md) for the supported deployment configurations.
