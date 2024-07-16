package container

import (
	"fmt"
	"github.com/simplecontainer/smr/pkg/static"
)

func (container *Container) GetDomain(networkName string) string {
	return fmt.Sprintf("%s.%s.%s.%s", networkName, container.Static.Group, container.Static.GeneratedName, static.SMR_LOCAL_DOMAIN)
}

func (container *Container) GetHeadlessDomain(networkName string) string {
	return fmt.Sprintf("%s.%s.%s.%s", networkName, container.Static.Group, container.Static.Name, static.SMR_LOCAL_DOMAIN)
}

func (container *Container) GetGroupIdentifier() string {
	return fmt.Sprintf("%s.%s", container.Static.Group, container.Static.GeneratedName)
}
