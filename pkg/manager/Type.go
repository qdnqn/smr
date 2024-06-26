package manager

import (
	"github.com/dgraph-io/badger/v4"
	"github.com/qdnqn/smr/pkg/dns"
	"github.com/qdnqn/smr/pkg/gitops"
	"github.com/qdnqn/smr/pkg/keys"
	"github.com/qdnqn/smr/pkg/objectdependency"
	"github.com/qdnqn/smr/pkg/reconciler"
	"github.com/qdnqn/smr/pkg/registry"
	"github.com/qdnqn/smr/pkg/runtime"
)
import "github.com/qdnqn/smr/pkg/config"

type Manager struct {
	Config             *config.Config
	Runtime            *runtime.Runtime
	Registry           *registry.Registry
	Reconciler         *reconciler.Reconciler
	Keys               *keys.Keys
	Badger             *badger.DB
	BadgerEncrypted    *badger.DB
	DnsCache           *dns.Records
	RepositoryWatchers *gitops.RepositoryWatcher
	ContainersWatchers *reconciler.ContainersWatcher
	DefinitionRegistry *objectdependency.DefinitionRegistry
}
