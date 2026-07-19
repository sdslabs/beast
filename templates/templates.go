package templates

var CHALLENGE_CONFIG_FILE_TEMPLATE string = `# Beast challenge configuration. Replace the example metadata before deployment.
[author]
name = {{printf "%q" .Author.Name}}
email = {{printf "%q" .Author.Email}}

[challenge.metadata]
name = {{printf "%q" .Challenge.Metadata.Name}}
type = {{printf "%q" .Challenge.Metadata.Type}}
dynamicFlag = {{.Challenge.Metadata.DynamicFlag}}
flag = {{printf "%q" .Challenge.Metadata.Flag}}
difficulty = {{printf "%q" .Challenge.Metadata.Difficulty}}
points = {{.Challenge.Metadata.Points}}
maxAttemptLimit = 0
tags = []
assets = []
additionalLinks = []

[challenge.env]
static_dir = {{printf "%q" .Challenge.Env.StaticContentDir}}

[resource]
# Omitted limits inherit the administrator-defined global defaults.
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
{{if .XinetdService}}   echo "mv {{.XinetdConf}} /etc/xinetd.d/pwn_service" >> /entrypoint.sh && {{end}}\
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
