# Contribution Guidelines

## Contents

- [Coding Style Guide](#coding-style-guide)
- [Pre-Commit](#pre-commit)
- [Git Commit Message Format](#git-commit-message-format)
- [Maintenance](#maintenance)

## Coding Style Guide

Beast uses `logrus` for logging purposes and follows standard effective go guidelines. You can refer to [this guide](https://github.com/golang/go/wiki/CodeReviewComments)
for more information.

## Pre-Commit

Anytime you are writing a code, keep in mind to add necessary logs and documentation. Also, format the code before committing using `gofmt`. Or simply run the make command `make test`

For any API routes, you add to the beast API, do write Swagger API documentation.

## Git Commit Message Format

Taken from https://github.com/angular/angular.js/blob/master/CONTRIBUTING.md and modified as required.
Each commit message consists of a **header**, a **body** and a **footer**. The header has a special
format that includes a **type**, a **scope** and a **subject**:

```
<type>: <subject>
<BLANK LINE>
<summary>
```

Any line of the commit message cannot be longer 100 characters! This allows the message to be easier
to read on github as well as in various git tools.

### Type

Must be one of the following:

- **feat**: A new feature
- **fix**: A bug fix
- **style**: CSS Changes
- **cleanup**: Changes that do not affect the meaning of the code (white-space, formatting, missing
  semi-colons, dead code removal etc.)
- **refactor**: A code change that neither fixes a bug or adds a feature
- **perf**: A code change that improves performance
- **test**: Adding missing tests or fixing them
- **chore**: Changes to the build process or auxiliary tools and libraries such as documentation
  generation
- **tracking**: Any kind of tracking which includes Bug Tracking, User Tracking, Anyalytics, AB-Testing etc
- **docs**: Documentation only changes

### Subject

The subject contains succinct description of the change:

- use the imperative, present tense: "change" not "changed" nor "changes"
- don't capitalize first letter
- no dot (.) at the end

### Summary

Just as in the **subject**, use the imperative, present tense: "change" not "changed" nor "changes"
The body should include the motivation for the change, contrast this with previous behavior and testing steps.

## Maintenance

### Opening an Issue

- Bug - Link the exact line where you found the bug or a way to reproduce the bug
- Crash Bug - Provide the stack trace at the point where the crash happens
- Convention Violation - Nomenclature inconsistency, class design inconsistency, unnecessary includes in headers
- Maintenance - Issues related to maintenance practices
- Proposal - State at least these 3 exact things in your proposal: Usage of feature, frequency of usage, and which part of codebase it goes in (helps in reviewing)

### Opening a PR

- 1 PR solves 1 issue (Not true in initial steps of any system but this will be valid eventually)
- Github will tag the exact directories where you have made a change. Verify those
- PRs should be reviewed in PR meetings only
