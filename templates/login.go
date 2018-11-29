package templates

// This template is to be inserted in the authorized_keys file which contains
// authorized ssh keys to login to the challenge container
var AUTHORIZED_KEY_TEMPLATE string = `
command="{{.Command}}",environment="SSH_USER={{.AuthorID}}",no-agent-forwarding,no-port-forwarding,no-X11-forwarding` +
	` {{.PubKey}} `

// This script will be run for each login, and command will be forced
// to enter the container on the basis of the command used by the
// user while logging in through ssh.
var SSH_LOGIN_SCRIPT_TEMPLATE string = `
#!/bin/bash
#
# This is an automatically generated shell script, don't edit this
# this wraps the forced command for a ssh session.
# $SSH_ORIGINAL_COMMAND recieves the original command sent by the user.
# This script is generate for user : {{.Author}}

case "$SSH_ORIGINAL_COMMAND" in

	{{range $name, $containerId := .Challenges}}
    "{{ $name }}")
        exec docker-enter {{ $containerId }}
        ;;
    {{end}}

    *)
        echo "Access denied...."
        exit 1
        ;;

esac
`
