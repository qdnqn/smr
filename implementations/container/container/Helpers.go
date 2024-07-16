package container

import (
	"encoding/json"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	v1 "github.com/simplecontainer/smr/pkg/definitions/v1"
	"github.com/simplecontainer/smr/pkg/logger"
	"log"
	"net/http"
	"os"
)

func (container *Container) portMappings() map[nat.Port][]nat.PortBinding {
	var portBindings = make(map[nat.Port][]nat.PortBinding, len(container.Static.MappingPorts))

	for _, v := range container.Static.MappingPorts {
		if v.Host != "" {
			portBindings[nat.Port(v.Container)] = []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: v.Host,
				},
			}
		}
	}

	return portBindings
}

func (container *Container) exposedPorts() map[nat.Port]struct{} {
	var exposedPorts = make(map[nat.Port]struct{}, len(container.Static.ExposedPorts))

	for _, v := range container.Static.ExposedPorts {
		exposedPorts[nat.Port(v)] = struct{}{}
	}

	return exposedPorts
}

func convertResourcesDefinitionToResources(definition []map[string]string) []Resource {
	mapToJson, err := json.Marshal(definition)
	if err != nil {
		logger.Log.Error(err.Error())
		return []Resource{}
	}

	var resources []Resource
	if err := json.Unmarshal(mapToJson, &resources); err != nil {
		logger.Log.Error(err.Error())
		return []Resource{}
	}

	for i, _ := range resources {
		resources[i].Data = make(map[string]string)
	}

	return resources
}

func (container *Container) mappingToMounts(client *http.Client) []mount.Mount {
	var mounts []mount.Mount

	for _, v := range container.Static.Resources {
		tmpFile, err := os.CreateTemp("/tmp", container.Static.Name)
		if err != nil {
			log.Fatal(err)
		}

		if _, err = tmpFile.WriteString(UnpackSecretsResources(client, v.Data[v.Key])); err != nil {
			log.Fatal(err)
		}

		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: tmpFile.Name(),
			Target: v.MountPoint,
		})
	}

	for _, v := range container.Static.MappingFiles {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: v["host"],
			Target: v["target"],
		})
	}

	return mounts
}

func convertMapToPortMapping(ports []map[string]string) []PortMappings {
	portmappings := make([]PortMappings, 0)

	for _, pm := range ports {
		portmappings = append(portmappings, PortMappings{
			Container: pm["container"],
			Host:      pm["host"],
		})
	}

	return portmappings
}

func convertPortMappingsToExposedPorts(portMappings []PortMappings) []string {
	var exposedPorts []string

	for _, portMap := range portMappings {
		exposedPorts = append(exposedPorts, portMap.Container)
	}

	return exposedPorts
}

func convertReadinessDefinitionToReadiness(readinessDefinition []v1.Readiness) []Readiness {
	var readiness = make([]Readiness, 0)

	for _, val := range readinessDefinition {
		readinessTmp := Readiness{
			Name:     val.Name,
			Operator: val.Operator,
			Timeout:  val.Timeout,
			Body:     val.Body,
			Solved:   false,
			Ctx:      nil,
			Cancel:   nil,
		}

		readiness = append(readiness, readinessTmp)
	}

	return readiness
}
