package templates

var CHALLENGE_CONFIG_FILE_TEMPLATE string = `# This a sample challenge configuration file.
[author]
name      = {{.Author.Name}}                      # Required: Name of the challenge creator
email     = {{.Author.Email}}                      # Required: Email for contact
ssh_key   = {{.Author.SSHKey}}                    # Required: Public SSH key for the challenge author

[challenge.metadata]
name            = {{.Challenge.Metadata.Name}}         # Required: Name of the challenge, should be same as directory.
type            = {{.Challenge.Metadata.Type}}         # Required: Type of challenge -> [web:<language>:<version>:<framework> static service]
dynamic_flag    = {{.Challenge.Metadata.DynamicFlag}} # Required: Dynamic flag or not -> [true/false]
flag            = {{.Challenge.Metadata.Flag}}         # Challenge Flag if dynamic_flag is false
sidecar          = {{.Challenge.Metadata.Sidecar}}        # Specify helper sidecar container for example mysql

[challenge.env]
apt_deps         = {{.Challenge.Env.AptDeps}}              # Custom apt-dependencies for challenge
ports            = {{.Challenge.Env.Ports}}                # Required: Port to expose for the challenge
setup_script     = {{.Challenge.Env.SetupScripts}}          # Setup script to run additional steps for challenge deployment
static_dir       = {{.Challenge.Env.StaticContentDir}}     # Static directory to be served for the challenge
base             = {{.Challenge.Env.BaseImage}}        # Base image-type for the challenge[bare("web", "service"), php(web), node(web)]
run_cmd          = {{.Challenge.Env.RunCmd}}               # Required(not for web): Entrypoint command for the challenge container(for bare base specify compelete command)
`

var BEAST_DOCKERFILE_TEMPLATE string = `# Beast Dockerfile
FROM {{.DockerBaseImage}}

LABEL version="0.2"
LABEL author="SDSLabs"

RUN groupadd -g 1337 beast-grp
RUN useradd -u 1337 -g 1337 -ms /bin/bash beast

RUN apt-get -y update && apt-get -y upgrade
RUN apt-get -y install {{.AptDeps}}

{{if .Ports}}EXPOSE {{.Ports}} {{end}}
VOLUME ["{{.MountVolume}}"]

COPY . /challenge

WORKDIR /challenge

{{ range $key, $elem := .EnvironmentVariables}}
ENV {{$key}} "{{$elem}}" 
{{end}}

RUN cd /challenge {{ range $index, $elem := .SetupScripts}} && \
    chmod u+x {{$elem}} {{end}} {{ range $index, $elem := .SetupScripts}} && \
    ./{{$elem}} {{end}}

{{if not .Entrypoint}}
RUN touch /entrypoint.sh && \
    echo "#!/bin/bash" > /entrypoint.sh && \
    echo "set -euxo pipefail" >> /entrypoint.sh && \
    echo "if [ -f /challenge/post-build.sh ]; then" >> /entrypoint.sh && \
    echo "    chmod u+x /challenge/post-build.sh && /challenge/post-build.sh" >> /entrypoint.sh && \
    echo "fi" >> /entrypoint.sh && \
    echo "cd /challenge" >> /entrypoint.sh && \
    echo "if [ -d /challenge/public ]; then" >> /entrypoint.sh && \
    echo "    chgrp beast-grp -R /challenge/public" >> /entrypoint.sh && \
    echo "    chmod -R 755 /challenge/public" >> /entrypoint.sh && \
    echo "fi" >> /entrypoint.sh && \
{{if .SetupCommand}}    echo "{{.SetupCommand}}" >> /entrypoint.sh && {{end}}\
{{if .XinetdService}}   echo "mv xinetd.conf /etc/xinetd.d/pwn_service" >> /entrypoint.sh && {{end}}\
    echo {{if .RunRoot}}"exec /bin/bash -c \"{{.RunCmd}}\""{{else}} "exec su beast /bin/bash -c \"{{.RunCmd}}\"" {{end}} >> /entrypoint.sh && \
    chmod u+x /entrypoint.sh
{{else}}
RUN chmod u+x {{.Entrypoint}}
{{end}}
WORKDIR /challenge
RUN chmod 600 /challenge/beast.toml {{ range $index, $elem := .Executables}} && \
    chmod +x {{$elem}} {{end}}
ENTRYPOINT ["{{if .Entrypoint}}{{.Entrypoint}}{{else}}/entrypoint.sh{{end}}"]
`
