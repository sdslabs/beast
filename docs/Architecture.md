# Architecture

## Controller

The Beast process owns the HTTPS API, scheduler, bounded worker queue, health/instance reconcilers, PostgreSQL pool, Redis client, and remote-worker registry. A filesystem lock prevents two controllers from using the same local state directory. Shutdown drains controller-owned goroutines and connections but intentionally leaves deployed challenges running.

## Durable and coordination state

PostgreSQL is the durable source for users, challenge metadata, ownership, submissions, scores, runtime identifiers, and deployed ports. Critical submission and dynamic-flag transitions use transactions, row locks, and unique indexes. Ports are unique per deployment server.

Redis stores namespaced coordination data, port allocation, cached metadata, and expiring instance markers. Atomic Lua/pipeline operations protect shared updates. Expiry notifications accelerate cleanup, while periodic reconciliation is the fallback.

## Deployment workers

Challenges run on the local Docker daemon or active SSH workers. Worker selection is round-robin. Remote host keys are verified from `known_hosts`; unavailable active workers fail startup rather than silently reducing capacity.

Challenge builds apply CPU, memory, and PID limits. Generated service images run challenge commands as an unprivileged user unless an explicit entrypoint/custom image requires otherwise. Docker/Compose authors remain trusted because build instructions execute with Docker authority.

## Request and task flow

1. TLS and request-size limits are applied.
2. Authentication establishes JWT claims and role middleware.
3. Ownership checks authorize challenge-specific management operations.
4. Strict configuration/path validation runs before staging.
5. Work enters a bounded, de-duplicating queue.
6. The selected local/remote runtime performs build or lifecycle work.
7. Completion and failures are recorded and returned/logged; database state changes are transactional where multiple relations must move together.

Scheduled tasks never overlap with themselves. Health probes use timeouts, verify HTTPS certificates, reject redirects, and bound response bodies.

## Static service

Static challenge content is copied to normalized public directories. A digest-pinned `nginx-unprivileged` image runs as UID 101 and receives only per-challenge read-only mounts, never the whole staging tree. The service listens on container port 8080 and is published on host port 8034; production deployments should terminate public TLS in front of it.

## Trust boundaries

- API users are untrusted; inputs, body sizes, filenames, archive entries, and authorization are validated.
- Challenge authors are trusted to supply build code, but paths and dangerous Compose runtime controls are constrained.
- Docker and remote SSH credentials are highly privileged and must be protected as root-equivalent.
- PostgreSQL, Redis, Git, SMTP, and webhook networks are external trust boundaries and require authenticated, verified transport when remote.
