package templates

var CHALLENGE_CONFIG_FILE_TEMPLATE string = `
# This a sample challenge configuration file.

[challenge]
id              = {{.ChallengeId}}               # ID of the challenge
name            = {{.ChallengeName}}             # Name of the challenge
challenge_type  = {{.ChallengeType}}             # Type of challenge -> web service ssh

    [challenge.details]
    flag                 = {{.ChallengeFlag}}    # Flag for the challenge -> Can be left blank
    apt_dependencies     = {{.AptDeps}}          # Custom apt-dependencies for challenge
    setup_script         = {{.SetupScript}}      # Setup script to run additional steps for challenge deployment
    static_content_dir   = {{.StaticContentDir}} # Static directory to be served for the challenge
    ports				 = {{.Ports}}
    run_cmd              = {{.RunCmd}}

[author]
name      = {{.AuthorName}}                      # Name of the challenge creator
email     = {{.AuthorMail}}                      # Email for contact
ssh_key   = {{.AuthorPubKey}}                    # Public SSH key for the challenge author
`

var BEAST_BARE_DOCKERFILE_TEMPLATE string = `# Beast Dockerfile
FROM debian:jessie

LABEL version="0.1"
LABEL author="fristonio"


WORKDIR /challenge
COPY . /challenge

RUN apt-get -y update && apt-get -y upgrade
RUN apt-get -y install {{.AptDeps}}

EXPOSE {{.Ports}}

RUN chmod +x {{.SetupScript}} && \
	./{{.SetupScript}}

USER beast
ENTRYPOINT ["{{.RunCmd}}"]
`

var AUTHORIZED_KEY_TEMPLATE string = `
# Challenge Name : {{.Name}}
command="docker-enter {{.ContainerId}}",no-agent-forwarding,no-port-forwarding,no-X11-forwarding` +
	` ssh-rsa {{.PubKey}} {{.Mail}}`
