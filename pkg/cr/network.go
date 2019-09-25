package cr

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"

	"golang.org/x/net/context"
)

type CreateNetworkConfig struct {
	NetworkName string
}

func SearchNetworkByFilter(filterMap map[string]string) ([]types.NetworkResource, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return []types.NetworkResource{}, err
	}

	filterArgs := filters.NewArgs()
	for key, val := range filterMap {
		filterArgs.Add(key, val)
	}

	networkDetails, err := cli.NetworkList(context.Background(), types.NetworkListOptions{
		Filters: filterArgs,
	})
	return networkDetails, err
}

func CreateNetwork(networkConfig *CreateNetworkConfig) (string, error) {
	networkName := networkConfig.NetworkName
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return "", err
	}

	network, err := cli.NetworkCreate(ctx, networkName, types.NetworkCreate{})
	if err != nil {
		log.Error("Error while creating network")
		return "", err
	}
	return network.ID, nil
}
