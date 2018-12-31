package api

const (
	WELCOME_TEXT = `
                             Beast
                       -----------------
       Beast is an automatic challenge deployment and management
       tool for backdoor.
                     @SDSLabs IIT Roorkee
`

	HELP_TEXT = `
BEAST API
=========

* /welcome - Beast welcome text
* /help - Help related to Beast API

* action - Action to be taken on the selected challenge which may
	be one of up, down, restart, purge
* id - Corresponds to unique Identifier for each challenge.

Namespaces with routes:

* Deploy Namespace(/deploy)
	1. /deploy/all/{action}
	2. /deploy/challenge/{id}
	3. /deploy/local/

* Status Namespace
	1. /status/all
	2. /status/challenge/{id}

* Info Namespace
	1. /info/challenge/{id}
	2. /info/available

* Remotes Namespaces
	1. /remote/sync
	2. /remote/reset

`

	WIP_TEXT = "WORK IN PROGRESS"

	MANAGE_ACTION_UNDEPLOY = "undeploy"
	MANAGE_ACTION_DEPLOY   = "deploy"
	MANAGE_ACTION_PURGE    = "purge"
	MANAGE_ACTION_REDEPLOY = "redeploy"
)
