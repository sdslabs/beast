## beast chall-details

Lists all challenge details

### Synopsis

Lists all challenge details | Flags available : --status , --tags. Status flag can take arguments : deployed / undeployed / queued. Tags flag can take multiple arguments seperated with ',' : (Ex : --tags=pwn,image,docker). Details are shown for challenges that have specified status and one of the specified tags.

```
beast chall-details [flags]
```

### Options

```
  -h, --help            help for chall-details
  -s, --status string   Filter by status : deployed / undeployed / queued (default "all")
  -t, --tags string     Filter by tagname : pwn / web / image / docker
```

### Options inherited from parent commands

```
  -v, --verbose   Print extra information in stdout1
```

### SEE ALSO

* [beast](beast.md)	 - Beast is an deployment and management tool for CTF challenges.
