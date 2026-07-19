## beast challenge

Performs action to the challs

### Synopsis

Performs actions like : deploy, undeploy, redeploy, purge to the challs

```
beast challenge action [challname] [-atld] [flags]
```

### Options

```
  -a, --all                      Performs action to all challs
  -d, --delete-entry             Deletes db entry related to this challenge
  -h, --help                     help for challenge
  -l, --local-directory string   Deploys challenge from local directory
  -c, --no-cache                 Build image of challenge without using cache
  -t, --tag string               Performs action to the tag provided
```

### Options inherited from parent commands

```
  -v, --verbose   Print extra information in stdout1
```

### SEE ALSO

* [beast](beast.md)	 - Beast is an deployment and management tool for CTF challenges.
