package mtls

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/simplecontainer/smr/pkg/configuration"
	"github.com/simplecontainer/smr/pkg/keys"
	"github.com/simplecontainer/smr/pkg/static"
	"log"
	"math/big"
	"net"
	"time"
)

/*
Taken and modified from the: https://gist.github.com/shaneutt/5e1995295cff6721c89a71d13a71c251
*/

func GenerateKeys(keys *keys.Keys, config *configuration.Configuration) error {
	// set up our CA certificate
	ca := &x509.Certificate{
		SerialNumber: generateSerialNumber(keys),
		Subject: pkix.Name{
			Organization:  []string{"Simple container manager."},
			Country:       []string{"BA"},
			Province:      []string{""},
			Locality:      []string{"Zivinice"},
			StreetAddress: []string{"BB"},
			PostalCode:    []string{"75270"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// create our private and public key
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
	}

	// create the CA
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return err
	}

	// pem encode
	caPEM := new(bytes.Buffer)
	pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})

	keys.CAPem = caPEM

	caPrivKeyPEMPKCS8, err := x509.MarshalPKCS8PrivateKey(caPrivKey)

	caPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: caPrivKeyPEMPKCS8,
	})

	keys.CAPrivateKey = caPrivKeyPEM

	keys.ServerPrivateKey, keys.ServerCertPem, err = generateCertPrivKeyPair(keys, config, ca, caPrivKey)

	if err != nil {
		return err
	}

	keys.ClientPrivateKey, keys.ClientCertPem, err = generateCertPrivKeyPair(keys, config, ca, caPrivKey)

	if err != nil {
		return err
	}

	return nil
}

func generateCertPrivKeyPair(keys *keys.Keys, config *configuration.Configuration, ca *x509.Certificate, caPrivKey *rsa.PrivateKey) (*bytes.Buffer, *bytes.Buffer, error) {
	certPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}

	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&certPrivKey.PublicKey)
	if err != nil {
		log.Fatalf("failed to marshal public key: %s", err)
	}

	SubjectKeyIdentifier := sha1.Sum(pubKeyBytes)

	ip := net.ParseIP(config.ExternalIP)

	if ip == nil {
		return nil, nil, errors.New("invalid external IP provided")
	}

	cert := &x509.Certificate{
		SerialNumber: generateSerialNumber(keys),
		Subject: pkix.Name{
			Organization:  []string{"Simple container manager."},
			Country:       []string{"BA"},
			Province:      []string{""},
			Locality:      []string{"Zivinice"},
			StreetAddress: []string{"BB"},
			PostalCode:    []string{"75270"},
		},
		DNSNames:     []string{config.Domain, fmt.Sprintf("smr-agent.%s", static.SMR_LOCAL_DOMAIN)},
		IPAddresses:  []net.IP{ip, net.IPv6loopback},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: SubjectKeyIdentifier[:],
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &certPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, err
	}

	certPEM := new(bytes.Buffer)
	pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	certPrivKeyPKCS8, err := x509.MarshalPKCS8PrivateKey(certPrivKey)

	certPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: certPrivKeyPKCS8,
	})

	return certPrivKeyPEM, certPEM, nil
}

func generateSerialNumber(keys *keys.Keys) *big.Int {
	keys.SerialNumber += 1
	return big.NewInt(keys.SerialNumber)
}
