package cr

import (
	"time"

	"github.com/docker/docker/client"
)

const (
	dockerAPIRequestTimeout = 30 * time.Second
	dockerAPILongTimeout    = 2 * time.Minute
)

func newDockerClient() (*client.Client, error) {
	return client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
}
