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
    flag            = {{.ChallengeFlag}}         # Challenge Flag

    [challenge.env]
    apt_deps         = {{.AptDeps}}              # Custom apt-dependencies for challenge
    ports            = {{.Ports}}                # Port to expose for the challenge
    setup_script     = {{.SetupScript}}          # Setup script to run additional steps for challenge deployment
    static_dir       = {{.StaticContentDir}}     # Static directory to be served for the challenge
    base             = {{.ChallengeBase}}        # Base image-type for the challenge[bare("web", "service"), php(web), node(web)]
    run_cmd          = {{.RunCmd}}               # Entrypoint command for the challenge container(for bare base specify compelete command)
    sidecar          = {{.SidecarHelper}}        # Specify helper sidecar container for example mysql
`

var BEAST_BARE_DOCKERFILE_TEMPLATE string = `# Beast Dockerfile
FROM {{.DockerBaseImage}}

LABEL version="0.2"
LABEL author="SDSLabs"

RUN useradd -ms /bin/bash beast

RUN apt-get -y update && apt-get -y upgrade
RUN apt-get -y install {{.AptDeps}}
RUN apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

{{if .Ports}}EXPOSE {{.Ports}} {{end}}
VOLUME ["{{.MountVolume}}"]

COPY . /challenge

RUN cd /challenge {{ range $index, $elem := .SetupScripts}} && \
	chmod u+x {{$elem}} {{end}} {{ range $index, $elem := .SetupScripts}} && \
    ./{{$elem}} {{end}}

RUN touch /entrypoint.sh && \
    echo "#!/bin/bash" > /entrypoint.sh && \
    echo "set -euxo pipefail" >> /entrypoint.sh && \
    echo "if [ -f /challenge/post-build.sh ]; then" >> /entrypoint.sh && \
    echo "    chmod u+x /challenge/post-build.sh && /challenge/post-build.sh" >> /entrypoint.sh && \
    echo "fi" >> /entrypoint.sh && \
    echo "exec su beast /bin/bash -c \"{{.RunCmd}}\"" >> /entrypoint.sh && \
    chmod u+x /entrypoint.sh

WORKDIR /challenge

ENTRYPOINT ["/entrypoint.sh"]
`
