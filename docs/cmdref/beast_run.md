## beast run

Run Beast API server

### Synopsis

Run beast API server using beast/api/server, optionally an argument can be provided to specify the port to run the server on.

```
beast run [flags]
```

### Options

```
  -a, --auto-deploy                           Auto deploy all challenges from remote on server start.
      --default-author-password-file string   0600 file containing the password used to create missing authors
  -k, --health-probe                          Run health check service for beast deployed challenges
  -h, --help                                  help for run
  -c, --no-cache                              Build image of challenge without using cache
  -s, --periodic-sync                         Periodically sync remote with beast and auto update challenges.
  -p, --port string                           Port to run the beast server on.
```

### Options inherited from parent commands

```
  -v, --verbose   Print extra information in stdout1
```

### SEE ALSO

* [beast](beast.md)	 - Beast is an deployment and management tool for CTF challenges.
