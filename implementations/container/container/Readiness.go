package container

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cenkalti/backoff/v4"
	"github.com/simplecontainer/smr/implementations/container/status"
	v1 "github.com/simplecontainer/smr/pkg/definitions/v1"
	"github.com/simplecontainer/smr/pkg/f"
	"github.com/simplecontainer/smr/pkg/static"
	"go.uber.org/zap"
	"net/http"
	"regexp"
	"strings"
	"time"
)

func NewReadinessFromDefinition(client *http.Client, container *Container, readiness v1.Readiness) (*Readiness, error) {
	if readiness.Timeout == "" {
		readiness.Timeout = "30s"
	}

	timeout, err := time.ParseDuration(readiness.Timeout)

	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	for index, value := range readiness.Body {
		regexDetectBigBrackets := regexp.MustCompile(`{{([^{\n}]*)}}`)
		matches := regexDetectBigBrackets.FindAllStringSubmatch(value, -1)

		if len(matches) > 0 {
			format := f.NewFromString(matches[0][1])

			if format.IsValid() && format.Kind == "secret" {
				continue
			} else {
				readiness.Body[index] = strings.Replace(readiness.Body[index], matches[0][0], container.Runtime.Configuration[format.Group], 1)
			}
		}
	}

	var bodyUnpack map[string]string
	bodyUnpack, err = UnpackSecretsReadiness(client, readiness.Body)

	if err != nil {
		cancel()
		return nil, err
	}

	return &Readiness{
		Name:       readiness.Name,
		Operator:   readiness.Operator,
		Timeout:    readiness.Timeout,
		Body:       readiness.Body,
		BodyUnpack: bodyUnpack,
		Ctx:        ctx,
		Cancel:     cancel,
	}, nil
}

func (container *Container) Ready(client *http.Client, channel chan *ReadinessState, logger *zap.Logger) (bool, error) {
	for _, ready := range container.Static.Definition.Spec.Container.Readiness {
		readiness, err := NewReadinessFromDefinition(client, container, ready)

		if err != nil {
			return false, err
		}

		readiness.Function = func() error {
			return SolveReadiness(client, container, logger, readiness, channel)
		}

		backOff := backoff.WithContext(backoff.NewExponentialBackOff(), readiness.Ctx)

		err = backoff.Retry(readiness.Function, backOff)
		if err != nil {
			channel <- &ReadinessState{
				State: FAILED,
			}

			return false, err
		}
	}

	channel <- &ReadinessState{
		State: SUCCESS,
	}

	container.Status.LastReadiness = true
	container.Status.LastReadinessTimestamp = time.Now()

	return true, nil
}

func SolveReadiness(client *http.Client, container *Container, logger *zap.Logger, readiness *Readiness, channel chan *ReadinessState) error {
	if !container.Status.IfStateIs(status.STATUS_READINESS_CHECKING) {
		readiness.Cancel()
	}

	channel <- &ReadinessState{
		State: CHECKING,
	}

	format := f.NewFromString(readiness.Name)
	URL := fmt.Sprintf("https://%s/api/v1/operators/%s/%s", static.SMR_AGENT_URL, format.Kind, readiness.Operator)

	jsonBytes, err := json.Marshal(readiness.BodyUnpack)

	logger.Info("readiness probe", zap.String("URL", URL), zap.String("data", string(jsonBytes)))

	var req *http.Request

	req, err = http.NewRequest("POST", URL, bytes.NewBuffer(jsonBytes))

	if err != nil {
		return errors.New("readiness request creation failed")
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return errors.New("readiness request failed")
	} else {
		if resp.StatusCode == http.StatusOK {
			return nil
		} else {
			return errors.New("readiness request failed")
		}
	}
}
