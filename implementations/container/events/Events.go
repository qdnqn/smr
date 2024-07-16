package events

import (
	"fmt"
	"github.com/simplecontainer/smr/implementations/container/container"
	"github.com/simplecontainer/smr/implementations/container/shared"
	"github.com/simplecontainer/smr/implementations/container/status"
	"github.com/simplecontainer/smr/implementations/hub/hub"
	"github.com/simplecontainer/smr/pkg/helpers"
)

func ListenEvents(shared *shared.Shared, e chan *hub.Event) {
	for {
		select {
		case event := <-e:
			Event(shared, event)
		}
	}
}

func Event(shared *shared.Shared, event *hub.Event) {
	var container *container.Container

	fmt.Println(event)

	if event == nil {
		fmt.Println("nil event")
		return
	}

	// handle container events
	if helpers.Contains([]string{"Container"}, event.Kind) {
		container = shared.Registry.Find(event.Group, event.Identifier)
	}

	if container == nil {
		return
	}

	c, err := container.Get()

	if err != nil {
		return
	}

	managed := false

	// only manage smr created containers, others are left alone to live and die in peace
	if c.Labels["managed"] == "smr" {
		managed = true
	}

	switch event.Kind {
	case "change":
		if managed {
			HandleChange(shared, container)
		}
		break
	default:
	}
}

func HandleChange(shared *shared.Shared, containerObj *container.Container) {
	reconcile := true

	// labels for ignoring events for specific container
	val, exists := containerObj.Static.Labels["reconcile"]
	if exists {
		if val == "false" {
			reconcile = false
		}
	}

	if reconcile {
		containerObj.Status.TransitionState(containerObj.Static.GeneratedName, status.STATUS_PREPARE)
		shared.Watcher.Find(containerObj.GetGroupIdentifier()).ContainerQueue <- containerObj
	}
}
