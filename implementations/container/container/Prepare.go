package container

import (
	"encoding/json"
	"fmt"
	v1 "github.com/simplecontainer/smr/pkg/definitions/v1"
	"github.com/simplecontainer/smr/pkg/f"
	"github.com/simplecontainer/smr/pkg/objects"
	"github.com/simplecontainer/smr/pkg/template"
	"net/http"
	"regexp"
	"strings"
)

// TODO: Refactor prepare it sucks
func (container *Container) Prepare(client *http.Client) error {
	var configuration map[string]string
	var resources = make([]Resource, 0)
	var dependencyMap []*f.Format
	var err error

	format := f.New("configuration", container.Static.Group, container.Static.GeneratedName, "")

	obj := objects.New(client)
	configuration, dependencyMap, err = template.ParseTemplate(obj, container.Runtime.Configuration, format)

	if err != nil {
		return err
	}

	for _, v := range container.Static.Resources {
		format = f.New("resource", v.Group, v.Name, "object")

		obj = objects.New(client)
		err = obj.Find(format)

		if err != nil {
			return err
		}

		resourceObject := v1.Resource{}

		err = json.Unmarshal(obj.GetDefinitionByte(), &resourceObject)

		if err != nil {
			return err
		}

		v.Data = resourceObject.Spec.Data
		resources = append(resources, v)

	}

	for k, _ := range resources {
		resources[k].Data, _, err = template.ParseTemplate(obj, resources[k].Data, nil)

		if err != nil {
			return err
		}
	}

	for k, _ := range resources {
		container.Runtime.ObjectDependencies = append(container.Runtime.ObjectDependencies, &f.Format{
			Kind:       "resource",
			Group:      container.Static.Resources[k].Group,
			Identifier: container.Static.Resources[k].Name,
			Key:        "",
		})
	}

	container.Runtime.Configuration = configuration

	for _, network := range container.Static.Networks {
		container.Runtime.Configuration[fmt.Sprintf("%s_hostname", network)] = container.GetDomain(network)
	}

	container.Runtime.ObjectDependencies = append(container.Runtime.ObjectDependencies, dependencyMap...)
	container.Static.Resources = resources

	// Replace placholders from the label keys in container obj with data from
	// Configuration.Runtime which has data retrieved from the KV store
	for index, _ := range container.Static.Labels {
		regexDetectBigBrackets := regexp.MustCompile(`{{([^{\n}]*)}}`)
		matches := regexDetectBigBrackets.FindAllStringSubmatch(index, -1)

		if len(matches) > 0 {
			trimmedMatch := strings.TrimSpace(matches[0][1])
			SplitByDot := strings.SplitN(trimmedMatch, ".", 2)

			if len(SplitByDot) > 1 && container.Runtime.Configuration[SplitByDot[1]] != "" {
				newIndex := strings.Replace(index, matches[0][0], container.Runtime.Configuration[SplitByDot[1]], 1)
				container.Static.Labels[newIndex] = container.Static.Labels[index]

				delete(container.Static.Labels, index)
			}
		}
	}

	// Replace placholders from the envs in container obj with data from
	// Configuration.Runtime which has data retrieved from the KV store
	for index, value := range container.Static.Env {
		regexDetectBigBrackets := regexp.MustCompile(`{{([^{\n}]*)}}`)
		matches := regexDetectBigBrackets.FindAllStringSubmatch(value, -1)

		if len(matches) > 0 {
			SplitByDot := strings.SplitN(matches[0][1], ".", 2)

			trimmedIndex := strings.TrimSpace(SplitByDot[1])

			if len(SplitByDot) > 1 && container.Runtime.Configuration[trimmedIndex] != "" {
				container.Static.Env[index] = strings.Replace(container.Static.Env[index], matches[0][0], container.Runtime.Configuration[trimmedIndex], 1)
			}
		}
	}

	// Replace placholders from the Readiness body in container obj with data from
	// Configuration.Runtime which has data retrieved from the KV store
	for indexReadiness, _ := range container.Static.Readiness {
		for index, value := range container.Static.Readiness[indexReadiness].Body {
			regexDetectBigBrackets := regexp.MustCompile(`{{([^{\n}]*)}}`)
			matches := regexDetectBigBrackets.FindAllStringSubmatch(value, -1)

			if len(matches) > 0 {
				format = f.NewFromString(matches[0][1])

				if format.IsValid() && format.Kind == "secret" {
					continue
				} else {
					container.Static.Readiness[indexReadiness].Body[index] = strings.Replace(container.Static.Readiness[indexReadiness].Body[index], matches[0][0], container.Runtime.Configuration[format.Group], 1)
				}
			}
		}
	}

	return nil
}
