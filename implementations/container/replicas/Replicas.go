package replicas

import (
	"github.com/r3labs/diff/v3"
	"github.com/simplecontainer/smr/implementations/container/container"
	"github.com/simplecontainer/smr/implementations/container/shared"
	"github.com/simplecontainer/smr/pkg/definitions/v1"
	"strings"
)

func (replicas *Replicas) HandleReplica(shared *shared.Shared, containerDefinition v1.Container, changelog diff.Changelog) (map[string][]string, map[string][]string, error) {
	create := map[string][]string{
		"groups": make([]string, 0),
		"names":  make([]string, 0),
	}

	remove := map[string][]string{
		"groups": make([]string, 0),
		"names":  make([]string, 0),
	}

	numberOfReplicasToCreate, numberOfReplicasToDestroy, existingNumberOfReplicas := replicas.GetReplicaNumbers(replicas.Replicas, replicas.GeneratedIndex)

	// Destroy from the end to start
	if numberOfReplicasToDestroy > 0 {
		for i := existingNumberOfReplicas; i > (existingNumberOfReplicas - numberOfReplicasToDestroy); i -= 1 {
			name, _ := shared.Registry.NameReplicas(containerDefinition.Meta.Group, containerDefinition.Meta.Name, shared.Manager.Config.Environment.PROJECT, i)
			existingContainer := shared.Registry.Find(containerDefinition.Meta.Group, name)

			if existingContainer != nil {
				remove["groups"] = append(remove["groups"], existingContainer.Static.Group)
				remove["names"] = append(remove["names"], existingContainer.Static.GeneratedName)
			}
		}
	}

	// Create from the start to the end
	for i := numberOfReplicasToCreate; i > 0; i -= 1 {
		name, _ := shared.Registry.NameReplicas(containerDefinition.Meta.Group, containerDefinition.Meta.Name, shared.Manager.Config.Environment.PROJECT, i)
		existingContainer := shared.Registry.Find(containerDefinition.Meta.Group, name)

		if existingContainer != nil {
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

		containerObj, err := container.NewContainerFromDefinition(shared.Manager.Config.Environment, name, containerDefinition)

		if err != nil {
			return nil, nil, err
		}

		if existingContainer == nil {
			shared.Registry.AddOrUpdate(replicas.Group, name, shared.Manager.Config.Environment.PROJECT, containerObj)
		} else {
			if replicas.Changed {
				shared.Registry.AddOrUpdate(replicas.Group, name, shared.Manager.Config.Environment.PROJECT, containerObj)
			}
		}

		create["groups"] = append(create["groups"], replicas.Group)
		create["names"] = append(create["names"], name)
	}

	return create, remove, nil
}

func (replicas *Replicas) GetReplica(shared *shared.Shared, containerDefinition v1.Container) ([]string, []string, error) {
	groups := make([]string, 0)
	names := make([]string, 0)

	_, _, existingNumberOfReplicas := replicas.GetReplicaNumbers(replicas.Replicas, replicas.GeneratedIndex)

	for i := existingNumberOfReplicas; i > 0; i -= 1 {
		name, _ := shared.Registry.NameReplicas(containerDefinition.Meta.Group, containerDefinition.Meta.Name, shared.Manager.Config.Environment.PROJECT, i)
		containerObj := shared.Registry.Find(containerDefinition.Meta.Group, name)

		if containerObj != nil {
			groups = append(groups, replicas.Group)
			names = append(names, name)
		}
	}

	return groups, names, nil
}

func (replicas *Replicas) GetReplicaNumbers(replicasNumber int, generatedNumber int) (int, int, int) {
	if replicasNumber > generatedNumber {
		return replicasNumber, 0, generatedNumber
	} else if replicasNumber == generatedNumber {
		return generatedNumber, 0, generatedNumber
	} else {
		return 0, generatedNumber - replicasNumber, generatedNumber
	}
}
