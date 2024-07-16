package v1

import (
	"encoding/json"
	"errors"
	"github.com/go-playground/validator/v10"
)

type Resource struct {
	Meta ResourceMeta `json:"meta"`
	Spec ResourceSpec `json:"spec"`
}

type ResourceMeta struct {
	Group string `json:"group"`
	Name  string `json:"name"`
}

type ResourceSpec struct {
	Data map[string]string `json:"data"`
}

func (resource *Resource) ToJsonString() (string, error) {
	bytes, err := json.Marshal(resource)
	return string(bytes), err
}

func (resource *Resource) Validate() (bool, error) {
	validate := validator.New(validator.WithRequiredStructEnabled())

	err := validate.Struct(resource)
	if err != nil {
		var invalidValidationError *validator.InvalidValidationError
		if errors.As(err, &invalidValidationError) {
			return false, err
		}
		// from here you can create your own error messages in whatever language you wish
		return false, err
	}

	return true, nil
}
