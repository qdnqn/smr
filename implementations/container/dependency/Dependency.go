package dependency

import (
	"context"
	"errors"
	"github.com/cenkalti/backoff/v4"
	"github.com/simplecontainer/smr/implementations/container/registry"
	"github.com/simplecontainer/smr/implementations/container/status"
	"github.com/simplecontainer/smr/pkg/definitions/v1"
	"github.com/simplecontainer/smr/pkg/f"
	"time"
)

func NewDependencyFromDefinition(depend v1.DependsOn) *Dependency {
	if depend.Timeout == "" {
		depend.Timeout = "30s"
	}

	timeout, err := time.ParseDuration(depend.Timeout)

	if err != nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	return &Dependency{
		Name:    depend.Name,
		Timeout: depend.Timeout,
		Ctx:     ctx,
		Cancel:  cancel,
	}
}

func Ready(registry *registry.Registry, group string, name string, dependsOn []v1.DependsOn, channel chan *State) (bool, error) {
	for _, depend := range dependsOn {
		dependency := NewDependencyFromDefinition(depend)
		dependency.Function = func() error {
			return SolveDepends(registry, group, name, dependency, channel)
		}

		backOff := backoff.WithContext(backoff.NewExponentialBackOff(), dependency.Ctx)

		err := backoff.Retry(dependency.Function, backOff)
		if err != nil {
			dependency.Cancel()

			channel <- &State{
				State: FAILED,
			}

			return false, nil
		}
	}

	channel <- &State{
		State: SUCCESS,
	}

	return true, nil
}

func SolveDepends(registry *registry.Registry, myGroup string, myName string, depend *Dependency, channel chan *State) error {
	format := f.NewFromString(depend.Name)

	myContainer := registry.Find(myGroup, myName)

	if myContainer == nil || myContainer.Status.IfStateIs(status.STATUS_DEPENDS_CHECKING) {
		depend.Cancel()
		return errors.New("container not found")
	}

	otherGroup := format.Kind
	otherName := format.Group

	channel <- &State{
		State: CHECKING,
	}

	container := registry.Find(otherGroup, otherName)

	if container == nil {
		return errors.New("container not found")
	} else {
		if container.Status.LastReadiness {
			depend.Cancel()
			return nil
		} else {
			return errors.New("container not ready")
		}
	}
}
