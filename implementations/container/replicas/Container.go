package replicas

import (
	"github.com/r3labs/diff/v3"
	"github.com/simplecontainer/smr/implementations/container/container"
	"github.com/simplecontainer/smr/implementations/container/shared"
	"github.com/simplecontainer/smr/implementations/container/status"
	"github.com/simplecontainer/smr/pkg/definitions/v1"
	"github.com/simplecontainer/smr/pkg/logger"
	"github.com/simplecontainer/smr/pkg/manager"
	"go.uber.org/zap"
	"strings"
)

func (replicas *Replicas) HandleContainer(shared *shared.Shared, mgr *manager.Manager, containerDefinition v1.Container, changelog diff.Changelog) ([]string, []string, error) {
	groups := make([]string, 0)
	names := make([]string, 0)

	numberOfReplicasToCreate, numberOfReplicasToDestroy, existingNumberOfReplicas := replicas.GetReplicaNumbers(replicas.Replicas, replicas.GeneratedIndex)

	if numberOfReplicasToDestroy > 0 {
		for i := existingNumberOfReplicas; i > (existingNumberOfReplicas - numberOfReplicasToDestroy); i -= 1 {
			name := containerDefinition.Meta.Name
			containerObj, err := container.NewContainerFromDefinition(mgr.Config.Environment, name, containerDefinition)

			if err != nil {
				return []string{}, []string{}, err
			}

			existingContainer := shared.Registry.Find(containerObj.Static.Group, name)

			if existingContainer != nil {
				existingContainer.Status.TransitionState(existingContainer.Static.GeneratedName, status.STATUS_PENDING_DELETE)
			}

			groups = append(groups, replicas.Group)
			names = append(names, name)
		}
	}

	for i := numberOfReplicasToCreate; i > 0; i -= 1 {
		name := containerDefinition.Meta.Name
		containerObj, err := container.NewContainerFromDefinition(mgr.Config.Environment, name, containerDefinition)

		if err != nil {
			return []string{}, []string{}, err
		}

		// Do pre-checks here before rewriting container in the registry

		logger.Log.Info("checking if pre-check conditions ready before add/update container in registry", zap.String("container", name))
		existingContainer := shared.Registry.Find(containerObj.Static.Group, name)

		if existingContainer != nil {
			logger.Log.Info("container already existing on the server", zap.String("container", name))

			var onlyReplicaChange = false

			if len(changelog) == 1 {
				for _, change := range changelog {
					if strings.Join(change.Path, ":") == "Spec:Container:Replicas" {
						if change.Type == "update" {
							onlyReplicaChange = true
						}
					}
				}
			}

			if onlyReplicaChange {
				continue
			}
		}

		shared.Registry.AddOrUpdate(replicas.Group, name, mgr.Config.Environment.PROJECT, containerObj)
		logger.Log.Info("added container to registry", zap.String("container", name), zap.String("group", replicas.Group))

		groups = append(groups, replicas.Group)
		names = append(names, name)
	}

	return groups, names, nil
}
