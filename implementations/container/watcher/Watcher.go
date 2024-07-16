package watcher

func (ContainerWatcher *ContainerWatcher) AddOrUpdate(groupidentifier string, container *Container) {
	if ContainerWatcher.Container[groupidentifier] == nil {
		ContainerWatcher.Container[groupidentifier] = container
	} else {
		ContainerWatcher.Container[groupidentifier] = container
	}
}

func (ContainerWatcher *ContainerWatcher) Remove(groupidentifier string) {
	delete(ContainerWatcher.Container, groupidentifier)
}

func (ContainerWatcher *ContainerWatcher) Find(groupidentifier string) *Container {
	if ContainerWatcher.Container[groupidentifier] != nil {
		return ContainerWatcher.Container[groupidentifier]
	} else {
		return nil
	}
}
