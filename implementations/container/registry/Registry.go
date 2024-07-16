package registry

import (
	"fmt"
	"github.com/simplecontainer/smr/implementations/container/container"
)

func (registry *Registry) AddOrUpdate(group string, name string, project string, containerAddr *container.Container) {
	registry.ContainersLock.Lock()
	if registry.Containers[group] == nil {
		tmp := make(map[string]*container.Container)
		tmp[name] = containerAddr

		registry.Containers[group] = tmp
	} else {
		registry.Containers[group][name] = containerAddr
	}

	registry.ContainersLock.Unlock()
}

func (registry *Registry) Remove(group string, name string) bool {
	registry.ContainersLock.Lock()
	if registry.Containers[group] == nil {
		registry.ContainersLock.Unlock()
		return true
	} else {
		delete(registry.Containers[group], name)

		if len(registry.Containers[group]) == 0 {
			delete(registry.Containers, group)
		}

		registry.ContainersLock.Unlock()
		return true
	}
}

func (registry *Registry) Find(group string, name string) *container.Container {
	registry.ContainersLock.RLock()
	if registry.Containers[group] != nil {
		if registry.Containers[group][name] != nil {
			registry.ContainersLock.RUnlock()
			return registry.Containers[group][name]
		} else {
			registry.ContainersLock.RUnlock()
			return nil
		}
	} else {
		registry.ContainersLock.RUnlock()
		return nil
	}
}

func (registry *Registry) Name(group string, name string, project string) (string, int) {
	index := registry.GenerateIndex(group, project)
	return fmt.Sprintf("%s-%s-%d", group, name, index), index
}

func (registry *Registry) NameReplicas(group string, name string, project string, index int) (string, int) {
	return fmt.Sprintf("%s-%s-%d", group, name, index), index
}

func (registry *Registry) BackOffTracking(group string, name string) {
	registry.ContainersLock.Lock()
	if registry.BackOffTracker[group] == nil {
		tmp := make(map[string]int)
		tmp[name] = 0

		registry.BackOffTracker[group] = tmp
	}

	registry.BackOffTracker[group][name] += 1
	registry.ContainersLock.Unlock()
}

func (registry *Registry) BackOffReset(group string, name string) {
	registry.ContainersLock.Lock()
	if registry.BackOffTracker[group] == nil {
		tmp := make(map[string]int)
		tmp[name] = 0

		registry.BackOffTracker[group] = tmp
	}

	registry.BackOffTracker[group][name] = 0
	registry.ContainersLock.Unlock()
}
