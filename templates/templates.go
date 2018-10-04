package templates

var CHALLENGE_CONFIG_FILE_TEMPLATE string = `
# This a sample challenge configuration file.
[author]
name      = {{.AuthorName}}                      # Name of the challenge creator
email     = {{.AuthorMail}}                      # Email for contact
ssh_key   = {{.AuthorPubKey}}                    # Public SSH key for the challenge author

[challenge]

	[challenge.metadata]
	id              = {{.ChallengeId}}           # ID of the challenge
	name            = {{.ChallengeName}}         # Name of the challenge
	type            = {{.ChallengeType}}         # Type of challenge -> [web service ssh]
	flag            = {{.ChallengeFlag}}		 # Challenge Flag

	[challenge.build]
	apt_deps         = {{.AptDeps}}              # Custom apt-dependencies for challenge
	ports	    	 = {{.Ports}}				 # Port to expose for the challenge
	setup_script     = {{.SetupScript}}          # Setup script to run additional steps for challenge deployment
	static_dir       = {{.StaticContentDir}}     # Static directory to be served for the challenge
	base			 = {{.ChallengeBase}}	  	 # Base image-type for the challenge[bare("web", "service"), static("web")]
	entrypoint 		 = {{.Entrypoint}} 			 # Entrypoint command for the challenge container(for bare base specify compelete command)
	entrypoint_args  = {{.EntrypointArgs}}		 # Arguments to entrypoint command
	sidecar 		 = {{.SidecarHelper}} 	 	 # Specify helper sidecar container for example mysql
`

var BEAST_BARE_DOCKERFILE_TEMPLATE string = `# Beast Dockerfile
FROM debian:jessie

LABEL version="0.1"
LABEL author="fristonio"

COPY . /challenge

RUN apt-get -y update && apt-get -y upgrade
RUN apt-get -y install {{.AptDeps}}

{{if .Ports}} EXPOSE {{.Ports}} {{end}}

RUN cd /challenge && \
	chmod +x {{.SetupScript}} && \
	./{{.SetupScript}}

RUN apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
RUN useradd -ms /bin/bash beast

USER beast
WORKDIR /challenge

CMD {{.RunCmd}}
`

var AUTHORIZED_KEY_TEMPLATE string = `
# Challenge Name : {{.Name}}
command="docker-enter {{.ContainerId}}",no-agent-forwarding,no-port-forwarding,no-X11-forwarding` +
	` ssh-rsa {{.PubKey}} {{.Mail}}`
