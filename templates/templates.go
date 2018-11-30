package templates

var CHALLENGE_CONFIG_FILE_TEMPLATE string = `
# This a sample challenge configuration file.
[author]
name      = {{.AuthorName}}                      # Name of the challenge creator
email     = {{.AuthorMail}}                      # Email for contact
ssh_key   = {{.AuthorPubKey}}                    # Public SSH key for the challenge author

[challenge]

	[challenge.metadata]
	name            = {{.ChallengeName}}         # Name of the challenge
	type            = {{.ChallengeType}}         # Type of challenge -> [web service ssh]
	flag            = {{.ChallengeFlag}}		 # Challenge Flag

	[challenge.env]
	apt_deps         = {{.AptDeps}}              # Custom apt-dependencies for challenge
	base_image       = {{.BaseImage}}            # Base Image
	ports	    	 = {{.Ports}}				 # Port to expose for the challenge
	setup_script     = {{.SetupScript}}          # Setup script to run additional steps for challenge deployment
	static_dir       = {{.StaticContentDir}}     # Static directory to be served for the challenge
	base			 = {{.ChallengeBase}}	  	 # Base image-type for the challenge[bare("web", "service"), php(web), node(web)]
	run_cmd 		 = {{.RunCmd}} 				 # Entrypoint command for the challenge container(for bare base specify compelete command)
	sidecar 		 = {{.SidecarHelper}} 	 	 # Specify helper sidecar container for example mysql
`

var BEAST_BARE_DOCKERFILE_TEMPLATE string = `# Beast Dockerfile
FROM {{.DockerBaseImage}}

LABEL version="0.2"
LABEL author="{{.Author}}"

RUN useradd -ms /bin/bash beast

RUN apt-get -y update && apt-get -y upgrade
RUN apt-get -y install {{.AptDeps}}
RUN apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

{{if .Ports}} EXPOSE {{.Ports}} {{end}}

COPY . /challenge
RUN cd /challenge && \
	chmod +x {{ range $index, $elem := .SetupScripts}} {{$elem}} {{end}} \
	{{ range $index, $elem := .SetupScripts}} && ./{{$elem}} \ {{end}}

USER beast
WORKDIR /challenge

CMD {{.RunCmd}}
`
