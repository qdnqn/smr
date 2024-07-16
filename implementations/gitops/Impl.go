package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mitchellh/mapstructure"
	"github.com/simplecontainer/smr/implementations/gitops/reconcile"
	"github.com/simplecontainer/smr/implementations/gitops/shared"
	"github.com/simplecontainer/smr/implementations/gitops/watcher"
	"github.com/simplecontainer/smr/pkg/definitions/v1"
	"github.com/simplecontainer/smr/pkg/f"
	"github.com/simplecontainer/smr/pkg/httpcontract"
	"github.com/simplecontainer/smr/pkg/logger"
	"github.com/simplecontainer/smr/pkg/manager"
	"github.com/simplecontainer/smr/pkg/objects"
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

	implementation.Shared.Watcher = &watcher.RepositoryWatcher{}
	implementation.Shared.Watcher.Repositories = make(map[string]*watcher.Gitops)

	return nil
}

func (implementation *Implementation) GetShared() interface{} {
	return implementation.Shared
}

func (implementation *Implementation) Apply(jsonData []byte) (httpcontract.ResponseImplementation, error) {
	var gitopsDefinition v1.Gitops

	if err := json.Unmarshal(jsonData, &gitopsDefinition); err != nil {
		return httpcontract.ResponseImplementation{
			HttpStatus:       400,
			Explanation:      "invalid configuration sent: json is not valid",
			ErrorExplanation: "invalid configuration sent: json is not valid",
			Error:            true,
			Success:          false,
		}, err
	}

	valid, err := gitopsDefinition.Validate()

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

	mapstructure.Decode(data["spec"], &gitopsDefinition)

	var format *f.Format

	format = f.New("gitops", gitopsDefinition.Meta.Group, gitopsDefinition.Meta.Name, "object")
	obj := objects.New(implementation.Shared.Client)
	err = obj.Find(format)

	var jsonStringFromRequest string
	jsonStringFromRequest, err = gitopsDefinition.ToJsonString()

	logger.Log.Debug("server received gitops object", zap.String("definition", jsonStringFromRequest))

	if obj.Exists() {
		if obj.Diff(jsonStringFromRequest) {
			err = obj.Update(format, jsonStringFromRequest)
		}
	} else {
		err = obj.Add(format, jsonStringFromRequest)
	}

	if obj.ChangeDetected() || !obj.Exists() {
		GroupIdentifier := fmt.Sprintf("%s.%s", gitopsDefinition.Meta.Group, gitopsDefinition.Meta.Name)

		gitopsFromDefinition := implementation.Shared.Watcher.Find(GroupIdentifier)

		if gitopsFromDefinition == nil {
			gitopsFromDefinition = reconcile.NewWatcher(&gitopsDefinition, implementation.Shared.Manager)
			go reconcile.HandleTickerAndEvents(implementation.Shared, gitopsFromDefinition)

			gitopsFromDefinition.Logger.Info("new gitops object created")
		} else {
			gitopsFromDefinition.Definition = gitopsDefinition
			gitopsFromDefinition.Logger.Info("gitops object modified")
		}

		implementation.Shared.Watcher.AddOrUpdate(GroupIdentifier, gitopsFromDefinition)
	} else {
		return httpcontract.ResponseImplementation{
			HttpStatus:       200,
			Explanation:      "object is same as the one on the server",
			ErrorExplanation: "",
			Error:            false,
			Success:          true,
		}, errors.New("object is same on the server")
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
	var gitopsDefinition v1.Gitops

	if err := json.Unmarshal(jsonData, &gitopsDefinition); err != nil {
		return httpcontract.ResponseImplementation{
			HttpStatus:       400,
			Explanation:      "invalid configuration sent: json is not valid",
			ErrorExplanation: "invalid configuration sent: json is not valid",
			Error:            true,
			Success:          false,
		}, err
	}

	data := make(map[string]interface{})
	err := json.Unmarshal(jsonData, &data)
	if err != nil {
		panic(err)
	}

	mapstructure.Decode(data["gitops"], &gitopsDefinition)

	var format *f.Format

	format = f.New("gitops", gitopsDefinition.Meta.Group, gitopsDefinition.Meta.Name, "object")
	obj := objects.New(implementation.Shared.Client)
	err = obj.Find(format)

	var jsonStringFromRequest string
	jsonStringFromRequest, err = gitopsDefinition.ToJsonString()

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
	return httpcontract.ResponseImplementation{
		HttpStatus:       200,
		Explanation:      "object in sync",
		ErrorExplanation: "",
		Error:            false,
		Success:          true,
	}, nil
}

var Gitops Implementation = Implementation{
	Started: false,
	Shared:  &shared.Shared{},
}
