# Getting started

[Set up Beast](Setup.md), then use a static challenge to verify the API and deployment pipeline.

## Deploy a static challenge

From the repository root, validate the bundled example and deploy it.

```sh
beast verify "$PWD/_examples/static-chall"
beast challenge deploy --local-directory "$PWD/_examples/static-chall"
beast chall-details --status deployed
```

With the API running, fetch its public asset.

```sh
curl -fsS 'http://localhost:5005/api/info/download?challenge=static-chall&asset=index.html'
```

`beast challenge undeploy static-chall` marks the challenge undeployed and keeps its staged files. Use `beast challenge purge static-chall` when you intend to remove the challenge and its competition records.

## Create a challenge

From a new challenge directory, run `beast new` to create a starting `beast.toml` and `public/` directory. Edit the author, replace the placeholder public key and flag, and select the challenge type. Keep the metadata name equal to the directory name. Run `beast verify "$PWD"` from that directory, or call `beast verify /path/to/challenge` from elsewhere.

For a containerized challenge, `[challenge.env].ports` lists container ports. Beast allocates host ports when it deploys. The [challenge configuration reference](ChallConfig.md) covers static assets, build scripts, custom Dockerfiles, and Compose.
