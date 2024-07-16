package main

import "C"
import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/simplecontainer/smr/implementations/containers/reconcile"
	"github.com/simplecontainer/smr/implementations/containers/shared"
	"github.com/simplecontainer/smr/implementations/containers/watcher"
	"github.com/simplecontainer/smr/pkg/definitions/v1"
	"github.com/simplecontainer/smr/pkg/f"
	"github.com/simplecontainer/smr/pkg/httpcontract"
	"github.com/simplecontainer/smr/pkg/logger"
	"github.com/simplecontainer/smr/pkg/manager"
	"github.com/simplecontainer/smr/pkg/objects"
	"github.com/simplecontainer/smr/pkg/plugins"
	"go.uber.org/zap"
)

func (implementation *Implementation) Start(mgr *manager.Manager) error {
	implementation.Shared.Manager = mgr
	implementation.Started = true

	client, err := manager.GenerateHttpClient(mgr.Keys)

	if err != nil {
		panic(err)
	}

	implementation.Shared.Client = client

	implementation.Shared.Watcher = &watcher.ContainersWatcher{}
	implementation.Shared.Watcher.Containers = make(map[string]*watcher.Containers)

	return nil
}

func (implementation *Implementation) GetShared() interface{} {
	return implementation.Shared
}

func (implementation *Implementation) Apply(jsonData []byte) (httpcontract.ResponseImplementation, error) {
	containersDefinition := &v1.Containers{}

	if err := json.Unmarshal(jsonData, &containersDefinition); err != nil {
		return httpcontract.ResponseImplementation{
			HttpStatus:       400,
			Explanation:      "invalid definition sent",
			ErrorExplanation: err.Error(),
			Error:            true,
			Success:          false,
		}, err
	}

	valid, err := containersDefinition.Validate()

	if !valid {
		return httpcontract.ResponseImplementation{
			HttpStatus:       400,
			Explanation:      "invalid definition sent",
			ErrorExplanation: err.Error(),
			Error:            true,
			Success:          false,
		}, err
	}

	data := make(map[string]interface{})
	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		panic(err)
	}

	var format *f.Format
	format = f.New("containers", containersDefinition.Meta.Group, containersDefinition.Meta.Name, "object")

	obj := objects.New(implementation.Shared.Client)
	err = obj.Find(format)

	var jsonStringFromRequest string
	jsonStringFromRequest, err = containersDefinition.ToJsonString()

	logger.Log.Debug("server received containers object", zap.String("definition", jsonStringFromRequest))

	if obj.Exists() {
		if obj.Diff(jsonStringFromRequest) {
			err = obj.Update(format, jsonStringFromRequest)

			if err != nil {
				return httpcontract.ResponseImplementation{
					HttpStatus:       200,
					Explanation:      "failed to update object",
					ErrorExplanation: err.Error(),
					Error:            false,
					Success:          true,
				}, err
			}
		}
	} else {
		err = obj.Add(format, jsonStringFromRequest)

		if err != nil {
			return httpcontract.ResponseImplementation{
				HttpStatus:       200,
				Explanation:      "failed to add object",
				ErrorExplanation: err.Error(),
				Error:            false,
				Success:          true,
			}, err
		}
	}

	if obj.ChangeDetected() || !obj.Exists() {
		GroupIdentifier := fmt.Sprintf("%s.%s", containersDefinition.Meta.Group, containersDefinition.Meta.Name)

		containersFromDefinition := implementation.Shared.Watcher.Find(GroupIdentifier)

		if containersFromDefinition == nil {
			containersFromDefinition = reconcile.NewWatcher(*containersDefinition, implementation.Shared.Manager)
			containersFromDefinition.Logger.Info("containers object created")

			go reconcile.HandleTickerAndEvents(implementation.Shared, containersFromDefinition)
		} else {
			containersFromDefinition.Definition = *containersDefinition
			containersFromDefinition.Logger.Info("containers object modified")
		}

		implementation.Shared.Watcher.AddOrUpdate(GroupIdentifier, containersFromDefinition)
		reconcile.ReconcileContainer(implementation.Shared, containersFromDefinition)
	} else {
		return httpcontract.ResponseImplementation{
			HttpStatus:       200,
			Explanation:      "containers object is same as the one on the server",
			ErrorExplanation: "",
			Error:            false,
			Success:          true,
		}, errors.New("containers object is same on the server")
	}

	return httpcontract.ResponseImplementation{
		HttpStatus:       200,
		Explanation:      "everything went smoothly: good job!",
		ErrorExplanation: "",
		Error:            false,
		Success:          true,
	}, nil
}

func (implementation *Implementation) Compare(jsonData []byte) (httpcontract.ResponseImplementation, error) {
	containersDefinition := &v1.Containers{}

	if err := json.Unmarshal(jsonData, &containersDefinition); err != nil {
		return httpcontract.ResponseImplementation{
			HttpStatus:       400,
			Explanation:      "invalid definition sent",
			ErrorExplanation: err.Error(),
			Error:            true,
			Success:          false,
		}, err
	}

	data := make(map[string]interface{})
	err := json.Unmarshal(jsonData, &data)
	if err != nil {
		panic(err)
	}

	var format *f.Format
	format = f.New("containers", containersDefinition.Meta.Group, containersDefinition.Meta.Name, "object")

	obj := objects.New(implementation.Shared.Client)
	err = obj.Find(format)

	var jsonStringFromRequest string
	jsonStringFromRequest, err = containersDefinition.ToJsonString()

	if obj.Exists() {
		obj.Diff(jsonStringFromRequest)
	} else {
		return httpcontract.ResponseImplementation{
			HttpStatus:       418,
			Explanation:      "object is drifted from the definition",
			ErrorExplanation: "",
			Error:            false,
			Success:          true,
		}, nil
	}

	if obj.ChangeDetected() {
		return httpcontract.ResponseImplementation{
			HttpStatus:       418,
			Explanation:      "object is drifted from the definition",
			ErrorExplanation: "",
			Error:            false,
			Success:          true,
		}, nil
	} else {
		return httpcontract.ResponseImplementation{
			HttpStatus:       200,
			Explanation:      "object in sync",
			ErrorExplanation: "",
			Error:            false,
			Success:          true,
		}, nil
	}
}

func (implementation *Implementation) Delete(jsonData []byte) (httpcontract.ResponseImplementation, error) {
	containersDefinition := &v1.Containers{}

	if err := json.Unmarshal(jsonData, &containersDefinition); err != nil {
		return httpcontract.ResponseImplementation{
			HttpStatus:       400,
			Explanation:      "invalid definition sent",
			ErrorExplanation: err.Error(),
			Error:            true,
			Success:          false,
		}, err
	}

	data := make(map[string]interface{})
	err := json.Unmarshal(jsonData, &data)
	if err != nil {
		panic(err)
	}

	var format *f.Format
	format = f.New("containers", containersDefinition.Meta.Group, containersDefinition.Meta.Name, "object")

	obj := objects.New(implementation.Shared.Client)
	err = obj.Find(format)

	if !obj.Exists() {
		return httpcontract.ResponseImplementation{
			HttpStatus:       404,
			Explanation:      "object not found on the server",
			ErrorExplanation: "",
			Error:            true,
			Success:          false,
		}, nil
	}

	_, err = obj.Remove(format)

	if err != nil {
		return httpcontract.ResponseImplementation{
			HttpStatus:       500,
			Explanation:      "object removal failed",
			ErrorExplanation: err.Error(),
			Error:            true,
			Success:          false,
		}, nil
	}

	GroupIdentifier := fmt.Sprintf("%s.%s", containersDefinition.Meta.Group, containersDefinition.Meta.Name)

	implementation.Shared.Watcher.Find(GroupIdentifier).Syncing = true
	implementation.Shared.Watcher.Find(GroupIdentifier).Cancel()

	for _, definition := range containersDefinition.Spec {
		format = f.New("container", definition.Meta.Group, definition.Meta.Name, "object")
		obj = objects.New(implementation.Shared.Client)
		obj.Find(format)

		if obj.Exists() {
			pl := plugins.GetPlugin(implementation.Shared.Manager.Config.Root, "container.so")
			pl.Delete(obj.GetDefinitionByte())
		}
	}

	return httpcontract.ResponseImplementation{
		HttpStatus:       200,
		Explanation:      "everything went good",
		ErrorExplanation: "",
		Error:            false,
		Success:          true,
	}, nil
}

// Exported
var Containers Implementation = Implementation{
	Started: false,
	Shared:  &shared.Shared{},
}
