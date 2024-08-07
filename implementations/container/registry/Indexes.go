package registry

import (
	"fmt"
	"github.com/simplecontainer/smr/pkg/logger"
	"sort"
	"strconv"
	"strings"
)

func (registry *Registry) GenerateIndex(name string, project string) int {
	var indexes []int = registry.GetIndexes(name, project)
	var index int = 0

	if len(indexes) > 0 {
		sort.Ints(indexes)
		index = indexes[len(indexes)-1] + 1
	}

	if index < 0 {
		index = 0
	}

	return index
}

func (registry *Registry) GetIndexes(name string, project string) []int {
	containers := registry.Containers[name]

	var indexes = make([]int, 0)
	name = fmt.Sprintf("%s-%s", project, name)

	if len(containers) > 0 {
		for _, containerObj := range containers {
			split := strings.Split(containerObj.Static.GeneratedName, "-")
			index, err := strconv.Atoi(split[len(split)-1])

			if err != nil {
				logger.Log.Fatal("Failed to convert string to int for index calculation")
			}

			indexes = append(indexes, index)
		}
	}

	if len(indexes) == 0 {
		indexes = append(indexes, 0)
	}

	return indexes
}
