package reconcile

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/simplecontainer/smr/implementations/gitops/gitops"
	"github.com/simplecontainer/smr/implementations/gitops/shared"
	"github.com/simplecontainer/smr/implementations/gitops/watcher"
	"github.com/simplecontainer/smr/pkg/definitions"
	v1 "github.com/simplecontainer/smr/pkg/definitions/v1"
	"github.com/simplecontainer/smr/pkg/httpcontract"
	"github.com/simplecontainer/smr/pkg/manager"
	"go.uber.org/zap"
	"io"
	"net/http"
	"path"
	"time"
)

func NewWatcher(gitopsObj *v1.Gitops, mgr *manager.Manager) *watcher.Gitops {
	interval := 5 * time.Second
	ctx, fn := context.WithCancel(context.Background())

	cfg := zap.NewProductionConfig()
	cfg.OutputPaths = []string{fmt.Sprintf("/tmp/gitops.%s.%s.log", gitopsObj.Meta.Group, gitopsObj.Meta.Name)}

	loggerObj, err := cfg.Build()

	if err != nil {
		panic(err)
	}

	watcher := &watcher.Gitops{
		Gitops: &gitops.Gitops{
			RepoURL:          gitopsObj.Spec.RepoURL,
			Revision:         gitopsObj.Spec.Revision,
			DirectoryPath:    gitopsObj.Spec.DirectoryPath,
			PoolingInterval:  gitopsObj.Spec.PoolingInterval,
			AutomaticSync:    gitopsObj.Spec.AutomaticSync,
			InSync:           false,
			CertKeyRef:       gitopsObj.Spec.CertKeyRef,
			HttpAuthRef:      gitopsObj.Spec.HttpAuthRef,
			LastSyncedCommit: plumbing.Hash{},
			CertKey:          nil,
			HttpAuth:         nil,
			Definition:       gitopsObj,
		},
		BackOff: watcher.BackOff{
			BackOff: false,
			Failure: 0,
		},
		Tracking:    true,
		Syncing:     false,
		GitopsQueue: make(chan *gitops.Gitops),
		Ctx:         ctx,
		Cancel:      fn,
		Ticker:      time.NewTicker(interval),
		Logger:      loggerObj,
	}

	return watcher
}

func HandleTickerAndEvents(shared *shared.Shared, gitopsWatcher *watcher.Gitops) {
	for {
		select {
		case <-gitopsWatcher.Ctx.Done():
			gitopsWatcher.Ticker.Stop()
			close(gitopsWatcher.GitopsQueue)
			shared.Watcher.Remove(fmt.Sprintf("%s.%s", gitopsWatcher.Gitops.Definition.Meta.Group, gitopsWatcher.Gitops.Definition.Meta.Name))

			return
		case <-gitopsWatcher.GitopsQueue:
			if gitopsWatcher.BackOff.BackOff {
				gitopsWatcher.Logger.Info("gitops reconcile is invalid, delete old and apply new",
					zap.String("repository", gitopsWatcher.Gitops.RepoURL),
				)
			} else {
				go ReconcileGitops(shared, gitopsWatcher)
			}
			break
		case <-gitopsWatcher.Ticker.C:
			if gitopsWatcher.BackOff.BackOff {
				gitopsWatcher.Logger.Info("gitops reconcile is invalid, delete old and apply new",
					zap.String("repository", gitopsWatcher.Gitops.RepoURL),
				)
			}

			if !gitopsWatcher.Gitops.AutomaticSync {
				if gitopsWatcher.Gitops.LastSyncedCommit.IsZero() {
					// For manual sync:
					// CheckInSync only after first successful sync
					break
				}
			}

			CheckInSync(shared, gitopsWatcher)

			if gitopsWatcher.Gitops.AutomaticSync && !gitopsWatcher.Gitops.InSync {
				go ReconcileGitops(shared, gitopsWatcher)
			}

			break
		}
	}
}

func ReconcileGitops(shared *shared.Shared, gitopsWatcher *watcher.Gitops) {
	var orderedByDependencies []map[string]string
	var auth transport.AuthMethod
	var hash plumbing.Hash
	var err error

	if gitopsWatcher.Syncing {
		gitopsWatcher.Logger.Info("gitops already reconciling, waiting for the free slot")
		return
	}

	gitopsWatcher.Syncing = true

	gitopsObj := gitopsWatcher.Gitops
	auth, err = GetAuth(gitopsObj)

	if err != nil {
		gitopsWatcher.Logger.Error("gitops reconcile auth configuration failed",
			zap.String("repository", gitopsObj.RepoURL),
			zap.Error(err),
		)

		gitopsWatcher.Backoff()
	}

	localPath := fmt.Sprintf("/tmp/%s", path.Base(gitopsObj.RepoURL))
	hash, err = Clone(gitopsObj, auth, localPath)

	if err != nil {
		gitopsWatcher.Logger.Error("gitops reconcile git clone failed",
			zap.String("repository", gitopsObj.RepoURL),
			zap.Error(err),
		)

		gitopsWatcher.Backoff()
	}

	if gitopsObj.LastSyncedCommit != hash || !gitopsObj.InSync {
		orderedByDependencies, err = SortFiles(gitopsObj, localPath, shared)

		if err != nil {
			gitopsWatcher.Logger.Error("gitops reconcile dependency order failed",
				zap.String("repository", gitopsObj.RepoURL),
				zap.Error(err),
			)

			gitopsWatcher.Backoff()
		}

		for _, fileInfo := range orderedByDependencies {
			fileName := fileInfo["name"]

			definition := definitions.ReadFile(fmt.Sprintf("%s/%s/%s", localPath, gitopsObj.DirectoryPath, fileName))
			if err != nil {
				gitopsWatcher.Logger.Info("gitops reconcile unable to read file",
					zap.String("repository", gitopsObj.RepoURL),
					zap.Error(err),
				)

				gitopsWatcher.Backoff()
			}

			response := sendRequest(shared.Client, "https://localhost:1443/api/v1/apply", definition, gitopsObj)

			if response.Success {
				gitopsWatcher.Logger.Info("gitops reconcile apply success",
					zap.String("repository", gitopsObj.RepoURL),
					zap.String("file", fileName),
					zap.Error(err),
				)

				gitopsObj.LastSyncedCommit = hash
			} else {
				gitopsWatcher.Logger.Info("gitops reconcile invalid object sent",
					zap.String("repository", gitopsObj.RepoURL),
					zap.Error(err),
				)

				gitopsWatcher.Backoff()
			}
		}
	} else {
		gitopsWatcher.Logger.Info("gitops reconcile hash is same as the last synced",
			zap.String("repository", gitopsObj.RepoURL),
		)
	}

	gitopsWatcher.Syncing = false
}

func CheckInSync(shared *shared.Shared, gitopsWatcher *watcher.Gitops) {
	var auth transport.AuthMethod
	var err error

	if gitopsWatcher.Syncing {
		gitopsWatcher.Logger.Info("container already reconciling, waiting for the free slot")
		return
	}

	gitopsObj := gitopsWatcher.Gitops
	auth, err = GetAuth(gitopsObj)

	if err != nil {
		gitopsWatcher.Logger.Info("gitops reconcile auth configuration failed",
			zap.String("repository", gitopsObj.RepoURL),
			zap.Error(err),
		)

		gitopsWatcher.Backoff()
	}

	localPath := fmt.Sprintf("/tmp/%s", path.Base(gitopsObj.RepoURL))
	_, err = Clone(gitopsObj, auth, localPath)

	if err != nil {
		gitopsWatcher.Logger.Info("gitops reconcile git clone failed",
			zap.String("repository", gitopsObj.RepoURL),
			zap.Error(err),
		)

		gitopsWatcher.Backoff()
	}

	gitopsWatcher.Logger.Info("checking if everything is in sync",
		zap.String("repoUrl", gitopsObj.RepoURL),
	)

	orderedByDependencies, err := SortFiles(gitopsObj, localPath, shared)

	if err != nil {
		gitopsWatcher.Logger.Error("gitops reconcile git clone failed",
			zap.String("repository", gitopsObj.RepoURL),
			zap.Error(err),
		)

		gitopsWatcher.Backoff()
	}

	inSync := true

	for _, fileInfo := range orderedByDependencies {
		fileName := fileInfo["name"]

		gitopsWatcher.Logger.Info("checking in sync", zap.String("file", fileName))

		definition := definitions.ReadFile(fmt.Sprintf("%s/%s/%s", localPath, gitopsObj.DirectoryPath, fileName))
		if err != nil {
			gitopsWatcher.Logger.Debug("gitops reconcile unable to read file",
				zap.String("repository", gitopsObj.RepoURL),
				zap.Error(err),
			)

			gitopsWatcher.Backoff()
		}

		response := sendRequest(shared.Client, "https://localhost:1443/api/v1/compare", definition, gitopsObj)

		switch response.HttpStatus {
		case http.StatusOK:
			gitopsWatcher.Logger.Info("file is in sync", zap.String("file", fileName))
			break
		case http.StatusTeapot:
			gitopsWatcher.Logger.Info("file is drifted", zap.String("file", fileName))
			inSync = false
			break
		case http.StatusBadRequest:
			gitopsWatcher.Logger.Info("gitops reconcile invalid object sent",
				zap.String("repository", gitopsObj.RepoURL),
				zap.Error(err),
			)

			gitopsWatcher.Backoff()
			break
		}
	}

	gitopsObj.InSync = inSync
}

func sendRequest(client *http.Client, URL string, data string, gitopsObj *gitops.Gitops) *httpcontract.ResponseImplementation {
	var req *http.Request
	var err error

	if len(data) > 0 {
		if err != nil {
			return &httpcontract.ResponseImplementation{
				HttpStatus:       0,
				Explanation:      "failed to marshal data for sending request",
				ErrorExplanation: err.Error(),
				Error:            true,
				Success:          false,
			}
		}

		req, err = http.NewRequest("POST", URL, bytes.NewBuffer([]byte(data)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Owner", fmt.Sprintf("gitops.%s.%s", gitopsObj.Definition.Meta.Group, gitopsObj.Definition.Meta.Name))
	} else {
		req, err = http.NewRequest("GET", URL, nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Owner", fmt.Sprintf("gitops.%s.%s", gitopsObj.Definition.Meta.Group, gitopsObj.Definition.Meta.Name))
	}

	if err != nil {
		return &httpcontract.ResponseImplementation{
			HttpStatus:       0,
			Explanation:      "failed to craft request",
			ErrorExplanation: err.Error(),
			Error:            true,
			Success:          false,
		}
	}

	resp, err := client.Do(req)

	if err != nil {
		return &httpcontract.ResponseImplementation{
			HttpStatus:       0,
			Explanation:      "failed to connect to the smr-agent",
			ErrorExplanation: err.Error(),
			Error:            true,
			Success:          false,
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &httpcontract.ResponseImplementation{
			HttpStatus:       0,
			Explanation:      "invalid response from the smr-agent",
			ErrorExplanation: err.Error(),
			Error:            true,
			Success:          false,
		}
	}

	var response httpcontract.ResponseImplementation
	err = json.Unmarshal(body, &response)

	if err != nil {
		return &httpcontract.ResponseImplementation{
			HttpStatus:       0,
			Explanation:      "failed to unmarshal body response from smr-agent",
			ErrorExplanation: err.Error(),
			Error:            true,
			Success:          false,
		}
	}

	return &response
}
