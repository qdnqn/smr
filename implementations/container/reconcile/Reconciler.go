package reconcile

import (
	"context"
	"errors"
	"fmt"
	"github.com/cenkalti/backoff/v4"
	"github.com/simplecontainer/smr/implementations/container/container"
	"github.com/simplecontainer/smr/implementations/container/dependency"
	"github.com/simplecontainer/smr/implementations/container/shared"
	"github.com/simplecontainer/smr/implementations/container/status"
	"github.com/simplecontainer/smr/implementations/container/watcher"
	hubShared "github.com/simplecontainer/smr/implementations/hub/shared"
	"github.com/simplecontainer/smr/pkg/manager"
	"github.com/simplecontainer/smr/pkg/plugins"
	"go.uber.org/zap"
	"time"
)

func NewWatcher(containerObj *container.Container, mgr *manager.Manager) *watcher.Container {
	interval := 5 * time.Second
	ctx, fn := context.WithCancel(context.Background())

	cfg := zap.NewProductionConfig()
	cfg.OutputPaths = []string{fmt.Sprintf("/tmp/container.%s.%s.log", containerObj.Static.Group, containerObj.Static.GeneratedName)}

	loggerObj, err := cfg.Build()
	if err != nil {
		panic(err)
	}

	pl := plugins.GetPlugin(mgr.Config.Root, "hub.so")
	sharedContainer := pl.GetShared().(*hubShared.Shared)

	return &watcher.Container{
		Container:      containerObj,
		Syncing:        false,
		ContainerQueue: make(chan *container.Container),
		ReadinessChan:  make(chan *container.ReadinessState),
		DependencyChan: make(chan *dependency.State),
		Ctx:            ctx,
		Cancel:         fn,
		Ticker:         time.NewTicker(interval),
		Logger:         loggerObj,
		EventChannel:   sharedContainer.Event,
	}
}

func HandleTickerAndEvents(shared *shared.Shared, containerWatcher *watcher.Container) {
	for {
		select {
		case <-containerWatcher.Ctx.Done():
			containerWatcher.Ticker.Stop()

			close(containerWatcher.ContainerQueue)
			close(containerWatcher.EventChannel)
			close(containerWatcher.ReadinessChan)
			close(containerWatcher.DependencyChan)

			shared.Watcher.Remove(containerWatcher.Container.GetGroupIdentifier())

			return
		case <-containerWatcher.ContainerQueue:
			containerWatcher.Ticker.Reset(5 * time.Second)
			go ReconcileContainer(shared, containerWatcher)
			break
		case <-containerWatcher.Ticker.C:
			if !containerWatcher.Container.Status.Reconciling && containerWatcher.Container.Status.GetCategory() != status.CATEGORY_END {
				go ReconcileContainer(shared, containerWatcher)
			} else {
				containerWatcher.Ticker.Stop()
			}
			break
		}
	}
}

func ReconcileContainer(shared *shared.Shared, containerWatcher *watcher.Container) {
	containerObj := containerWatcher.Container

	if containerObj.Status.Reconciling {
		containerWatcher.Logger.Info("container already reconciling, waiting for the free slot")
		return
	}

	containerObj.Status.Reconciling = true

	switch containerObj.Status.GetState() {
	case status.STATUS_CREATED:
		dockerState := GetState(containerWatcher)

		switch dockerState {
		case "running":
			containerWatcher.Logger.Info("container created but already running")
			containerObj.Status.TransitionState(containerObj.Static.GeneratedName, status.STATUS_KILL)
			break
		case "exited":
			containerWatcher.Logger.Info("container created but already exited")
			containerObj.Status.TransitionState(containerObj.Static.GeneratedName, status.STATUS_DEAD)
			break
		default:
			containerWatcher.Logger.Info("container created")
			containerObj.Status.TransitionState(containerObj.Static.GeneratedName, status.STATUS_PREPARE)
		}

		ReconcileLoop(containerWatcher)
		break
	case status.STATUS_PREPARE:
		err := containerObj.Prepare(shared.Client)

		if err == nil {
			go dependency.Ready(shared.Registry, containerObj.Static.Group, containerObj.Static.GeneratedName, containerObj.Static.Definition.Spec.Container.Dependencies, containerWatcher.DependencyChan)

			containerWatcher.Logger.Info("container prepared")
			containerObj.Status.TransitionState(containerObj.Static.GeneratedName, status.STATUS_DEPENDS_CHECKING)
		} else {
			containerWatcher.Logger.Info(err.Error())
			containerObj.Status.TransitionState(containerObj.Static.GeneratedName, status.STATUS_INVALID_CONFIGURATION)
		}

		ReconcileLoop(containerWatcher)
		break
	case status.STATUS_DEPENDS_CHECKING:
		ContinueReconciliation := false
		for !ContinueReconciliation {
			select {
			case dependencyResult := <-containerWatcher.DependencyChan:
				if dependencyResult == nil {
					ContinueReconciliation = true
					break
				}

				switch dependencyResult.State {
				case dependency.CHECKING:
					containerWatcher.Logger.Info("checking dependency")
					break
				case dependency.SUCCESS:
					containerWatcher.Logger.Info("dependency check success")
					containerObj.Status.TransitionState(containerObj.Static.GeneratedName, status.STATUS_DEPENDS_SOLVED)

					ContinueReconciliation = true
					break
				case dependency.FAILED:
					containerWatcher.Logger.Info("dependency check failed")
					containerObj.Status.TransitionState(containerObj.Static.GeneratedName, status.STATUS_DEPENDS_FAILED)

					ContinueReconciliation = true
					break
				}
			}
		}

		ReconcileLoop(containerWatcher)
		break
	case status.STATUS_DEPENDS_SOLVED:
		containerObj.Status.TransitionState(containerObj.Static.GeneratedName, status.STATUS_START)

		ReconcileLoop(containerWatcher)
		break
	case status.STATUS_START:
		containerWatcher.Logger.Info("container attempt to start")
		_, err := containerObj.Run(shared.Manager.Config.Environment, shared.Client, shared.DnsCache)

		if err == nil {
			containerWatcher.Logger.Info("container started")
			containerObj.Status.TransitionState(containerObj.Static.GeneratedName, status.STATUS_READINESS_CHECKING)
			go containerObj.Ready(shared.Client, containerWatcher.ReadinessChan, containerWatcher.Logger)
		} else {
			containerWatcher.Logger.Info("container start failed", zap.Error(err))
			containerObj.Status.TransitionState(containerObj.Static.GeneratedName, status.STATUS_DEAD)
		}

		ReconcileLoop(containerWatcher)
		break
	case status.STATUS_READINESS_CHECKING:
		ContinueReconciliation := false
		for !ContinueReconciliation {
			select {
			case readinessResult := <-containerWatcher.ReadinessChan:
				dockerState := GetState(containerWatcher)

				if dockerState != "running" {
					ContinueReconciliation = true
					containerObj.Status.TransitionState(containerObj.Static.GeneratedName, status.STATUS_DEAD)
					break
				}

				switch readinessResult.State {
				case dependency.CHECKING:
					containerWatcher.Logger.Info("checking readiness")
					break
				case dependency.SUCCESS:
					containerWatcher.Logger.Info("readiness check success")
					containerObj.Status.TransitionState(containerObj.Static.GeneratedName, status.STATUS_READY)

					ContinueReconciliation = true
					break
				case dependency.FAILED:
					containerWatcher.Logger.Info("readiness check failed")
					containerObj.Status.TransitionState(containerObj.Static.GeneratedName, status.STATUS_READINESS_FAILED)

					ContinueReconciliation = true
					break
				}
			}
		}

		ReconcileLoop(containerWatcher)
		break
	case status.STATUS_READY:
		containerObj.Status.TransitionState(containerObj.Static.GeneratedName, status.STATUS_RUNNING)

		ReconcileLoop(containerWatcher)
		break
	case status.STATUS_RUNNING:
		break
	case status.STATUS_DEAD:
		dockerState, err := containerObj.Get()

		if err != nil {
			containerWatcher.Logger.Info(err.Error())
			break
		}

		if dockerState.State == "exited" {
			containerWatcher.Logger.Info("container is dead")
			shared.Registry.BackOffTracking(containerObj.Static.Group, containerObj.Static.GeneratedName)

			if shared.Registry.BackOffTracker[containerObj.Static.Group][containerObj.Static.GeneratedName] > 5 {
				containerWatcher.Logger.Error(fmt.Sprintf("%s container is backoff restarting", containerObj.Static.GeneratedName))

				shared.Registry.BackOffReset(containerObj.Static.Group, containerObj.Static.GeneratedName)

				containerObj.Status.TransitionState(containerObj.Static.GeneratedName, status.STATUS_BACKOFF)
			} else {
				containerObj.Delete()
				containerObj.Status.TransitionState(containerObj.Static.GeneratedName, status.STATUS_PREPARE)
			}
		} else {
			containerWatcher.Logger.Info("container not dead retry again", zap.String("current-state", dockerState.State))
			containerObj.Status.TransitionState(containerObj.Static.GeneratedName, status.STATUS_KILL)
		}

		ReconcileLoop(containerWatcher)
		break
	case status.STATUS_BACKOFF:
		containerWatcher.Logger.Info("container is in backoff state")
		break
	case status.STATUS_DEPENDS_FAILED:
		containerWatcher.Logger.Info("container readiness failed")
		containerObj.Status.TransitionState(containerObj.GetGroupIdentifier(), status.STATUS_KILL)

		ReconcileLoop(containerWatcher)
		break
	case status.STATUS_READINESS_FAILED:
		containerWatcher.Logger.Info("container readiness failed")
		containerObj.Status.TransitionState(containerObj.GetGroupIdentifier(), status.STATUS_KILL)

		ReconcileLoop(containerWatcher)
		break
	case status.STATUS_INVALID_CONFIGURATION:
		containerWatcher.Logger.Info("container invalid configuration")
		break

	case status.STATUS_KILL:
		containerWatcher.Logger.Info("attempt to shutdown gracefully")

		go containerObj.Stop()
		Wait(func() error {
			c, err := containerObj.Get()

			if err != nil {
				return nil
			}

			if c != nil && c.State == "exited" {
				containerObj.Status.TransitionState(containerObj.GetGroupIdentifier(), status.STATUS_DEAD)
				return nil
			} else {
				return errors.New("container is not yet in exited state")
			}
		})

		ReconcileLoop(containerWatcher)
		break
	case status.STATUS_PENDING_DELETE:
		c, err := containerObj.Get()

		containerWatcher.Logger.Info("container is pending delete")

		if err != nil {
			shared.Registry.Remove(containerObj.Static.Group, containerObj.Static.GeneratedName)
			containerWatcher.Cancel()
		} else {
			if c.State == "running" {
				containerWatcher.Logger.Info("starting graceful termination, timeout 30s")

				containerObj.Stop()
				Wait(func() error {
					c, err := containerObj.Get()

					if err != nil {
						return nil
					}

					if c != nil && c.State == "exited" {
						return nil
					} else {
						return errors.New("container is not yet in exited state")
					}
				})
			}

			err = containerObj.Delete()

			if err != nil {
				containerWatcher.Logger.Info("failed to delete container from docker daemon")

			} else {
				containerWatcher.Logger.Info("container is deleted")

				shared.Registry.Remove(containerObj.Static.Group, containerObj.Static.GeneratedName)
				containerWatcher.Cancel()
			}
		}
		break
	}
}

func ReconcileLoop(containerWatcher *watcher.Container) {
	containerWatcher.Container.Status.Reconciling = false
	containerWatcher.ContainerQueue <- containerWatcher.Container
}

func GetState(containerWatcher *watcher.Container) string {
	c, err := containerWatcher.Container.Get()

	if err != nil {
		return ""
	}

	if c != nil {
		return c.State
	}

	return ""
}

func Wait(f func() error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	backOff := backoff.WithContext(backoff.NewExponentialBackOff(), ctx)

	err := backoff.Retry(f, backOff)
	if err != nil {
		return
	}

	return
}
