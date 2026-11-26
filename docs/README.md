# Beast documentation

Beast manages Jeopardy-style CTF challenges and competitions. It uses Docker for challenge workloads, PostgreSQL for persistent state, and Redis for live instance state, port reservations, and caching. The Go binary includes the CLI and the REST API. You can use a separate frontend for the competition.

- [Setup](Setup.md)
- [Getting started](GettingStarted.md)
- [CLI and API usage](Usage.md)
- [Authentication](APIAuth.md)
- [Challenge types](ChallTypes.md)
- [Challenge configuration](ChallConfig.md)
- [Bundled examples](SampleChallenges.md)
- [Deployment](Deployment.md)
- [Architecture](Architecture.md)
- [CLI reference](cmdref/beast.md)
