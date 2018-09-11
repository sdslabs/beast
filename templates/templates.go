package templates

var CHALLENGE_CONFIG_FILE_TEMPLATE string = `
# This a sample challenge configuration file.

[challenge]
id              = {{.ChallengeId}}               # ID of the challenge
name            = {{.ChallengeName}}             # Name of the challenge
challenge_type  = {{.ChallengeType}}             # Type of challenge -> web service ssh
run_cmd         = {{.RunCmd}}

    [challenge.details]
    flag                 = {{.ChallengeFlag}}    # Flag for the challenge -> Can be left blank
    apt_dependencies     = {{.AptDeps}}          # Custom apt-dependencies for challenge
    custom_setup_script  = {{.SetupScript}}      # Setup script to run additional steps for challenge deployment
    static_content_dir   = {{.StaticContentDir}} # Static directory to be served for the challenge

[author]
name      = {{.AuthorName}}                      # Name of the challenge creator
email     = {{.AuthorMail}}                      # Email for contact
ssh_key   = {{.AuthorPubKey}}                    # Public SSH key for the challenge author
`

var BEAST_DOCKERFILE_TEMPLATE string = `# Beast Dockerfile
FROM debian:jessie

WORKDIR /challenge
COPY . /challenge

RUN apt-get update && apt-get upgrade
RUN apt-get install {{.AptDeps}}

RUN chmod +x {{.SetupFile}} && \
	./{{.SetupFile}}

USER beast
ENTRYPOINT ["{{.RunCmd}}"]
`

var AUTHORIZED_KEY_TEMPLATE string = `
# Challenge Name : {{.Name}}
command="docker-enter {{.ContainerId}}",no-agent-forwarding,no-port-forwarding,no-X11-forwarding` +
	` ssh-rsa {{.PubKey}} {{.Mail}}`
