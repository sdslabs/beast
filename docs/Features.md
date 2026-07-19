# Features

## Challenge lifecycle

- Local directories or one/more SSH-authenticated Git remotes as challenge sources.
- Static, generated service/web/bare images, custom Dockerfiles, and constrained Compose projects.
- Deploy, undeploy, redeploy, purge, logs, status, health probes, and administrator-controlled container execution.
- Bounded asynchronous worker queue with task de-duplication and surfaced failures.
- Local and SSH Docker workers with per-worker port ranges and round-robin placement.

## Competition services

- Password login with role-scoped JWT authorization.
- Contestant registration/password reset, submissions, hints, prerequisites, attempt limits, and cheating records.
- Optional dynamic scoring and freeze/unfreeze leaderboard behavior.
- Per-user instances with expiration, extension limits, reconciliation, and administrator cleanup.
- Slack/Discord notifications and SSE updates.

## Safety controls

- Mandatory HTTPS API with timeouts, request-size bounds, strict CORS, and controlled panic responses.
- Strict global/challenge TOML and constrained Compose parsing.
- Contained archive extraction and staging that reject traversal, links, special files, duplicates, and expansion abuse.
- Default/hard-ceiling CPU, memory, and PID limits.
- Verified SSH host keys and mandatory verified transport for remote PostgreSQL/Redis.
- Unprivileged, digest-pinned static service with per-challenge read-only mounts.

Docker-backed builds still execute trusted organizer code with root-equivalent daemon authority. These controls reduce mistakes and runtime exposure; they do not make hostile Dockerfiles or kernel exploits safe.
