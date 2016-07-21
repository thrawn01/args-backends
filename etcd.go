package backends

import (
	"path"

	etcd "github.com/coreos/etcd/clientv3"
	"golang.org/x/net/context"
	"github.com/thrawn01/args"
	"fmt"
	"github.com/pkg/errors"
)



type EtcdBackend struct {
	Root string
	Client *etcd.Client
}


func NewEtcdBackend(client etcd.Client, root string) args.Backend {
	return &EtcdBackend {
		Root: root,
		Client: client,
	}
}

// Get retrieves a value from a K/V store for the provided key.
func (self *EtcdBackend) Get(ctx context.Context, key string) (*args.Pair, error) {
	resp, err := self.Client.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) == 0 {
		return errors.New(fmt.Sprintf("'%s' not found", key))
	}
	return args.NewPair(string(resp.Kvs[0].Key), resp.Kvs[0].Value), nil
}

// List retrieves all keys and values under a provided key.
func (self *EtcdBackend) List(ctx context.Context, key string) ([]args.Pair, error) {
	return nil, nil
}

// Set the provided key to value.
func (self *EtcdBackend) Set(ctx context.Context, key string, value []byte) error {
	return nil
}

// Watch monitors store for changes to key.
func (self *EtcdBackend) Watch(ctx context.Context, key string) <-chan args.ChangeEvent {
	return nil
}

// Given args.Rules and etcd.Response, attempt to match the response to the rules and return
// a new ChangeEvent.
func NewChangeEvent(rules args.Rules, event *etcd.Event) args.ChangeEvent {
	return &args.Event{
		Key:     path.Base(string(event.Kv.Key)),
		Value:   string(event.Kv.Value),
		Deleted: event.Type.String() == "DELETE",
		Error: nil,
	}
}

