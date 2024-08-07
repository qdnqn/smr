package registry

import (
	"github.com/simplecontainer/smr/implementations/container/container"
	"github.com/simplecontainer/smr/pkg/objects"
	"sync"
)

type Registry struct {
	Containers     map[string]map[string]*container.Container
	ContainersLock sync.RWMutex
	Indexes        map[string][]int
	BackOffTracker map[string]map[string]int
	Object         map[string]objects.Object
}
