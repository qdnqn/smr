package events

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
	"github.com/simplecontainer/smr/implementations/container/container"
	"github.com/simplecontainer/smr/implementations/container/shared"
	"github.com/simplecontainer/smr/pkg/logger"
	"go.uber.org/zap"
)

func ListenDockerEvents(shared *shared.Shared) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	cEvents, cErr := cli.Events(ctx, types.EventsOptions{})

	for {
		select {
		case err := <-cErr:
			fmt.Println(err)
		case msg := <-cEvents:
			// TODO: Do I want to do blocking here? Or go with gouroutine?
			HandleDockerEvent(shared, msg)
		}
	}
}

func HandleDockerEvent(shared *shared.Shared, event events.Message) {
	var container *container.Container

	c := container.GetFromId(event.Actor.Attributes["container"])

	if c == nil {
		logger.Log.Debug("container is not found in the docker daemon",
			zap.String("container", event.Actor.Attributes["container"]),
		)

		return
	}

	container = shared.Registry.Find(c.Labels["group"], c.Labels["name"])

	if container == nil {
		logger.Log.Debug("container is not found in the registry, ignore it",
			zap.String("container", c.Names[0]),
		)

		return
	}

	managed := false

	// only manage smr created containers, others are left alone to live and die in peace
	if c.Labels["managed"] == "smr" {
		managed = true
	}

	switch event.Action {
	case "connect":
		if managed {
			HandleConnect(shared, container, event)
		}
		break
	case "disconnect":
		if managed {
			HandleDisconnect(shared, container, event)
		}
		break
	case "start":
		if managed {
			HandleStart(shared, container, event)
		}
		break
	case "kill":
		if managed {
			HandleKill(shared, container, event)
		}
		break
	case "stop":
		if managed {
			HandleStop(shared, container, event)
		}
		break
	case "die":
		if managed {
			HandleDie(shared, container, event)
		}
		break
	default:
	}
}

func HandleConnect(shared *shared.Shared, container *container.Container, event events.Message) {
	// Handle network connect here
}

func HandleDisconnect(shared *shared.Shared, containerObj *container.Container, event events.Message) {
	for _, n := range containerObj.Runtime.Networks {
		for _, ip := range shared.DnsCache.FindDeleteQueue(containerObj.GetDomain(n.NetworkName)) {
			shared.DnsCache.RemoveARecord(containerObj.GetDomain(n.NetworkName), ip)
			shared.DnsCache.RemoveARecord(containerObj.GetHeadlessDomain(n.NetworkName), ip)
		}

		shared.DnsCache.ResetDeleteQueue(containerObj.GetDomain(n.NetworkName))
	}
}

func HandleStart(shared *shared.Shared, containerObj *container.Container, event events.Message) {
	// Container started it is running so update status accordingly
	// containerObj.Status.TransitionState(status.STATUS_RUNNING)
}

func HandleKill(shared *shared.Shared, containerObj *container.Container, event events.Message) {
	// It can happen that kill signal occurs in the container even if it is not dying; eg killing thread, goroutine etc.
	//containerObj.Status.TransitionState(containerObj.Static.GeneratedName, status.STATUS_KILLED)

	for _, n := range containerObj.Runtime.Networks {
		shared.DnsCache.RemoveARecordQueue(containerObj.GetDomain(n.NetworkName), n.IP)
	}
}

func HandleStop(shared *shared.Shared, containerObj *container.Container, event events.Message) {
	// Stop will stop the container so update the status accordingly
	// containerObj.Status.TransitionState(status.STATUS_DEAD)
}

func HandleDie(shared *shared.Shared, containerObj *container.Container, event events.Message) {
	reconcile := true

	// labels for ignoring events for specific container
	val, exists := containerObj.Static.Labels["reconcile"]
	if exists {
		if val == "false" {
			logger.Log.Info("reconcile label set to false for the container, skipping reconcile", zap.String("container", containerObj.Static.GeneratedName))
			reconcile = false
		}
	}

	if reconcile {
		logger.Log.Info(fmt.Sprintf("container is dead - reconcile %s", containerObj.Static.GeneratedName))
		shared.Watcher.Find(fmt.Sprintf("%s.%s", containerObj.Static.Group, containerObj.Static.GeneratedName)).ContainerQueue <- containerObj
	}
}
