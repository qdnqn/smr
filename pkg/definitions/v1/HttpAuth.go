package v1

import (
	"encoding/json"
	"errors"
	"github.com/go-playground/validator/v10"
)

type HttpAuth struct {
	Meta HttpAuthMeta `json:"meta"`
	Spec HttpAuthSpec `json:"spec"`
}

type HttpAuthMeta struct {
	Group string `json:"group"`
	Name  string `json:"name"`
}

type HttpAuthSpec struct {
	Username string
	Password string
}

func (httpauth *HttpAuth) ToJsonString() (string, error) {
	bytes, err := json.Marshal(httpauth)
	return string(bytes), err
}

func (httpauth *HttpAuth) Validate() (bool, error) {
	validate := validator.New(validator.WithRequiredStructEnabled())

	err := validate.Struct(httpauth)
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
