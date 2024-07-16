package objects

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/r3labs/diff/v3"
	"github.com/simplecontainer/smr/pkg/f"
	"github.com/simplecontainer/smr/pkg/logger"
	"github.com/simplecontainer/smr/pkg/static"
	"go.uber.org/zap"
	"net/http"
	"reflect"
	"strings"
	"time"
)

//go:generate mockgen -source=Interface.go -destination=mock/Interface.go

func New(client *http.Client) *Object {
	return &Object{
		Changelog:        diff.Changelog{},
		client:           client,
		definition:       map[string]any{},
		definitionString: "",
		definitionByte:   make([]byte, 0),
		exists:           false,
		changed:          false,
		created:          time.Now(),
		updated:          time.Now(),
	}
}

func (obj *Object) GetDefinitionString() string {
	return obj.definitionString
}

func (obj *Object) GetDefinition() map[string]any {
	return obj.definition
}

func (obj *Object) GetDefinitionByte() []byte {
	return obj.definitionByte
}

func (obj *Object) Add(format *f.Format, data string) error {
	URL := fmt.Sprintf("https://%s/api/v1/database/create/%s", static.SMR_AGENT_URL, format.ToString())
	response := SendRequest(obj.client, URL, "POST", map[string]string{"value": data})

	logger.Log.Debug("object add", zap.String("URL", URL), zap.String("data", data))

	if response.Success {
		return nil
	} else {
		return errors.New(response.ErrorExplanation)
	}
}

func (obj *Object) Update(format *f.Format, data string) error {
	URL := fmt.Sprintf("https://%s/api/v1/database/update/%s", static.SMR_AGENT_URL, format.ToString())
	response := SendRequest(obj.client, URL, "PUT", map[string]string{"value": data})

	logger.Log.Debug("object update", zap.String("URL", URL), zap.String("data", data))

	if response.Success {
		return nil
	} else {
		return errors.New(response.ErrorExplanation)
	}
}

func (obj *Object) Find(format *f.Format) error {
	URL := fmt.Sprintf("https://%s/api/v1/database/get/%s", static.SMR_AGENT_URL, format.ToString())
	response := SendRequest(obj.client, URL, "GET", nil)

	logger.Log.Debug("object find", zap.String("URL", URL))

	if response.Success {
		for key, value := range response.Data {
			if value == nil {
				continue
			}

			if strings.Contains(key, "object") {
				b64decoded, err := base64.StdEncoding.DecodeString(value.(string))

				data := make(map[string]any)
				err = json.Unmarshal(b64decoded, &data)

				if err != nil {
					return err
				}

				obj.definition = data
				obj.definitionByte = b64decoded
			} else {
				b64decoded, _ := base64.StdEncoding.DecodeString(value.(string))
				obj.definitionString = string(b64decoded)
			}
		}

	} else {
		return errors.New(response.ErrorExplanation)
	}

	obj.changed = false
	obj.exists = true

	return nil
}

func (obj *Object) FindMany(format *f.Format) (map[string]*Object, error) {
	var objects = make(map[string]*Object)

	URL := fmt.Sprintf("https://%s/api/v1/database/keys/prefix/%s", static.SMR_AGENT_URL, format.ToString())
	response := SendRequest(obj.client, URL, "GET", nil)

	logger.Log.Debug("object find many", zap.String("URL", URL))

	if response.Success {
		for key, value := range response.Data {
			if strings.Contains(key, "object") {
				fmt.Println(value.(string))

				b64decoded, err := base64.StdEncoding.DecodeString(value.(string))

				if err != nil {
					return nil, err
				}

				fmt.Println(string(b64decoded))

				data := make(map[string]any)
				err = json.Unmarshal(b64decoded, &data)

				if err != nil {
					return nil, err
				}

				objMany := New(obj.client)
				objMany.definition = data
				objMany.definitionByte = b64decoded

				objects[key] = objMany
			} else {
				b64decoded, _ := base64.StdEncoding.DecodeString(value.(string))

				objMany := New(obj.client)
				objMany.definitionString = string(b64decoded)
				objMany.definitionByte = b64decoded

				objects[key] = objMany
			}
		}
	} else {
		return nil, errors.New(response.ErrorExplanation)
	}

	return objects, nil
}

func (obj *Object) Remove(format *f.Format) (bool, error) {
	prefix := format.ToString()

	if !format.Full() {
		// Append dot to the end of the format so that we delimit what we deleting from the kv-store
		prefix += "."
	}

	URL := fmt.Sprintf("https://%s/api/v1/database/keys/%s", static.SMR_AGENT_URL, prefix)
	response := SendRequest(obj.client, URL, "DELETE", nil)

	logger.Log.Debug("object remove", zap.String("URL", URL))

	if response.Success {
		return true, nil
	} else {
		return false, errors.New(response.ErrorExplanation)
	}
}

func (obj *Object) Diff(definition string) bool {
	data := make(map[string]any)
	err := json.Unmarshal([]byte(definition), &data)

	if err != nil {
		return true
	}

	var changelog diff.Changelog

	if reflect.DeepEqual(obj.definition, data) {
		obj.changed = false
	} else {
		changelog, _ = diff.Diff(obj.definition, data)
		obj.Changelog = changelog
		obj.changed = true
	}

	return obj.changed
}

func (obj *Object) Exists() bool {
	return obj.exists
}

func (obj *Object) ChangeDetected() bool {
	return obj.changed
}
