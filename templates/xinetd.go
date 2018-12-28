package templates

var XINETD_CONFIGURATION_TEMPLATE string = `# Xinetd configuration.
service {{.ServiceName}} 
{
    disable 	= no
    type        = UNLISTED
    wait        = no
    server      = /bin/sh
    server_args = -c cd${IFS}/challenge;exec${IFS}{{.ServicePath}}
    socket_type = stream
    protocol    = tcp
    user        = beast 
    port        = {{.Port}}
    bind        = 0.0.0.0

    instances   = UNLIMITED
    flags       = REUSE
    per_source	= 5     # Maximum instances of this service per source IP address
    rlimit_cpu	= 20    # Maximum number of CPU seconds that the service may use.
    rlimit_as   = 512M  # Address Space resource limit for the service
}
`
