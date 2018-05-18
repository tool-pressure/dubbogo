package registry

import (
	"errors"
)

// Result is returned by a call to Next on
// the watcher. Actions can be create, update, delete
type Result = ServiceURLEvent

// Watcher is an interface that returns updates
// about services within the registry.
type Watcher interface {
	// Next is a blocking call
	Next() (*Result, error)
	Valid() bool // 检查watcher与registry连接是否正常
	Stop()
}

type NewRegistry func(...Option) Registry

// The registry provides an interface for service discovery
// and an abstraction over varying implementations
// {etcd, zookeeper, ...}
type Registry interface {
	// Register(conf ServiceConfig) error
	Register(conf interface{}) error
	GetServices(ServiceConfigIf) ([]*ServiceURL, error)
	Watch() (Watcher, error)
	Close()
	String() string
}

const (
	REGISTRY_CONN_DELAY = 3 // watchDir中使用，防止不断地对zk重连
)

var (
	ErrorRegistryNotFound = errors.New("registry not found")
)
