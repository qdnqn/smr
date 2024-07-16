package v1

import (
	"encoding/json"
	"github.com/go-playground/validator/v10"
)

type Configuration struct {
	Meta ConfigurationMeta `json:"meta"`
	Spec ConfigurationSpec `json:"spec"`
}

type ConfigurationMeta struct {
	Group string `json:"group"`
	Name  string `json:"name"`
}

type ConfigurationSpec struct {
	Data map[string]string `json:"data"`
}

func (configuration *Configuration) ToJsonString() (string, error) {
	bytes, err := json.Marshal(configuration)
	return string(bytes), err
}

func (configuration *Configuration) Validate() (bool, error) {
	validate := validator.New(validator.WithRequiredStructEnabled())

	err := validate.Struct(configuration)
	if err != nil {
		if _, ok := err.(*validator.InvalidValidationError); ok {
			return false, err
		}
		// from here you can create your own error messages in whatever language you wish
		return false, err
	}

	return true, nil
}
