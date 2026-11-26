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
  -d, --delete-entry             Compatibility flag for purge, which already removes the challenge record
  -h, --help                     help for challenge
  -l, --local-directory string   Deploys challenge from local directory
  -c, --no-cache                 Build image of challenge without using cache
  -t, --tag string               Performs action to the tag provided
```

### Options inherited from parent commands

```
  -v, --verbose   Enable verbose logging
```

### SEE ALSO

* [beast](beast.md)	 - Deploy and manage CTF challenges with Beast
