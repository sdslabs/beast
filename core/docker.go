package core

import (
	"fmt"

	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

var DockerClient client.Client

func init() {
	DockerClient, err := client.NewEnvClient()
	if err != nil {
		fmt.Println("Error while creating a docker client for beast")
		panic(err)
	}
}
