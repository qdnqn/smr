package v1

import (
	"encoding/json"
	"github.com/go-playground/validator/v10"
)

type CertKey struct {
	Meta CertKeyMeta `json:"meta"`
	Spec CertKeySpec `json:"spec"`
}

type CertKeyMeta struct {
	Group string `json:"group"`
	Name  string `json:"name"`
}

type CertKeySpec struct {
	Certificate        string `json:"certificate"`
	PublicKey          string `json:"publicKey"`
	PrivateKey         string `json:"privateKey"`
	PrivateKeyPassword string `json:"privateKeyPassword"`
	KeyStore           string `json:"keystore"`
	KeyStorePassword   string `json:"keyStorePassword"`
	CertStore          string `json:"certstore"`
	CertStorePassword  string `json:"certstorePassword"`
}

func (certkey *CertKey) ToJsonString() (string, error) {
	bytes, err := json.Marshal(certkey)
	return string(bytes), err
}

func (certkey *CertKey) Validate() (bool, error) {
	validate := validator.New(validator.WithRequiredStructEnabled())

	err := validate.Struct(certkey)
	if err != nil {
		if _, ok := err.(*validator.InvalidValidationError); ok {
			return false, err
		}
		// from here you can create your own error messages in whatever language you wish
		return false, err
	}

	return true, nil
}
