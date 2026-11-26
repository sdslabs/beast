## beast

Deploy and manage CTF challenges with Beast

### Synopsis

Beast deploys and manages Jeopardy-style CTF challenges and competitions.

```
beast [flags]
```

### Options

```
  -h, --help      help for beast
  -v, --verbose   Enable verbose logging
```

### SEE ALSO

* [beast backup-cache](beast_backup-cache.md)	 - Back up the Redis cache
* [beast backup-database](beast_backup-database.md)	 - Back up the PostgreSQL database
* [beast chall-details](beast_chall-details.md)	 - Lists all challenge details
* [beast challenge](beast_challenge.md)	 - Performs action to the challs
* [beast cmdref](beast_cmdref.md)	 - Generate beast command reference
* [beast config](beast_config.md)	 - Run interactive beast configuration setup
* [beast create-admin](beast_create-admin.md)	 - Creates new admin
* [beast create-author](beast_create-author.md)	 - Creates new author
* [beast disable-author-ssh](beast_disable-author-ssh.md)	 - Disables current authors to ssh into the containers
* [beast getauth](beast_getauth.md)	 - Gets an auth token from the Beast server
* [beast health-probe](beast_health-probe.md)	 - Run Health Probe
* [beast init](beast_init.md)	 - Set up Beast and check its dependencies
* [beast logs](beast_logs.md)	 - Provides live logs of a container
* [beast new](beast_new.md)	 - generate config file
* [beast reset-cache](beast_reset-cache.md)	 - Backups the existing cache and cleans up old cache and remote/staging directories
* [beast reset-database](beast_reset-database.md)	 - Backups the existing database and cleans up old db and remote/staging directories
* [beast restore-cache](beast_restore-cache.md)	 - Restores the cache, with the backed-up file
* [beast restore-database](beast_restore-database.md)	 - Restores the database, with the backed-up file
* [beast run](beast_run.md)	 - Run Beast API server
* [beast verify](beast_verify.md)	 - Verifies challenge config
* [beast version](beast_version.md)	 - Displays the version of the current build of beast
