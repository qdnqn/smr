package container

import (
	"context"
	"errors"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/simplecontainer/smr/pkg/logger"
)

func (container *Container) ConnectToTheSameNetwork(containerId string, networkId string) error {
	if c, _ := container.Get(); c != nil && c.State == "running" {
		ctx := context.Background()
		cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			panic(err)
		}
		defer cli.Close()

		// TODO: Don't connect if the network is same

		EndpointSettings := &network.EndpointSettings{
			NetworkID: networkId,
		}

		err = cli.NetworkConnect(ctx, networkId, containerId, EndpointSettings)

		if err != nil {
			logger.Log.Error(err.Error())
			return errors.New("failed to connect to the network")
		}

		return nil
	} else {
		return errors.New("container is not running")
	}
}
