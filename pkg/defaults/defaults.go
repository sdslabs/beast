package defaults

import "time"

var (
	// DefaultDockerStopTimeout is the default timeout to pass when trying to stop
	// the provided docker container using docker golang client.
	DefaultDockerStopTimeout time.Duration = time.Second * 3
)
