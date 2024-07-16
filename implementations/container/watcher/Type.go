package watcher

import (
	"context"
	"github.com/simplecontainer/smr/implementations/container/container"
	"github.com/simplecontainer/smr/implementations/container/dependency"
	"github.com/simplecontainer/smr/implementations/hub/hub"
	"go.uber.org/zap"
	"time"
)

type ContainerWatcher struct {
	Container map[string]*Container
}

type Container struct {
	Container      *container.Container
	Syncing        bool
	ContainerQueue chan *container.Container      `json:"-"`
	ReadinessChan  chan *container.ReadinessState `json:"-"`
	DependencyChan chan *dependency.State         `json:"-"`
	Ctx            context.Context                `json:"-" `
	Cancel         context.CancelFunc             `json:"-"`
	Ticker         *time.Ticker                   `json:"-"`
	Logger         *zap.Logger
	EventChannel   chan *hub.Event
}
