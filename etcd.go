package backends

import (
	"path"

	"fmt"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/pkg/errors"
	"github.com/thrawn01/args"
	"golang.org/x/net/context"
)

type EtcdBackend struct {
	Root   string
	Client *etcd.Client
	close  chan struct{}
}

func NewEtcdBackend(client *etcd.Client, root string) args.Backend {
	return &EtcdBackend{
		Root:   root,
		Client: client,
	}
}

// Get retrieves a value from a K/V store for the provided key.
func (self *EtcdBackend) Get(ctx context.Context, key string) (args.Pair, error) {
	resp, err := self.Client.Get(ctx, key)
	if err != nil {
		return args.Pair{}, err
	}
	if len(resp.Kvs) == 0 {
		return args.Pair{}, errors.New(fmt.Sprintf("'%s' not found", key))
	}
	return args.Pair{Key: string(resp.Kvs[0].Key), Value: resp.Kvs[0].Value}, nil
}

// List retrieves all keys and values under a provided key.
func (self *EtcdBackend) List(ctx context.Context, key string) ([]args.Pair, error) {
	resp, err := self.Client.Get(ctx, key, etcd.WithPrefix())
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) == 0 {
		return nil, errors.New(fmt.Sprintf("'%s' not found", key))
	}
	result := make([]args.Pair, 0)
	for _, node := range resp.Kvs {
		result = append(result, args.Pair{Key: string(node.Key), Value: node.Value})
	}
	return result, nil
}

// Set the provided key to value.
func (self *EtcdBackend) Set(ctx context.Context, key string, value []byte) error {
	return nil
}

// Watch monitors store for changes to key.
func (self *EtcdBackend) Watch(ctx context.Context, key string) <-chan *args.ChangeEvent {
	changeChan := make(chan *args.ChangeEvent)
	watchChan := self.Client.Watch(ctx, key, etcd.WithPrefix())

	go func() {
		var resp etcd.WatchResponse
		var ok bool
		select {
		case resp, ok = <-watchChan:
			if !ok {
				changeChan <- NewChangeError(errors.Wrap(resp.Err(),
					"etcd watch channel was closed"))
				close(changeChan)
			}
			if resp.Canceled {
				changeChan <- NewChangeError(errors.Wrap(resp.Err(),
					"EtcdBackend.WatchEtcd() ETCD Cancelled Watch"))
				close(changeChan)
			}
			for _, event := range resp.Events {
				changeChan <- NewChangeEvent(event)
			}
		case <-self.close:
			close(changeChan)
			return
		}
	}()
	return changeChan
}

func (self *EtcdBackend) Close() {
	if self.close != nil {
		close(self.close)
	}
}

func (self *EtcdBackend) GetRootKey() string {
	return self.Root
}

func NewChangeEvent(event *etcd.Event) *args.ChangeEvent {
	return &args.ChangeEvent{
		KeyName: path.Base(string(event.Kv.Key)),
		Key:     string(event.Kv.Key),
		Value:   event.Kv.Value,
		Deleted: event.Type.String() == "DELETE",
		Err:     nil,
	}
}

func NewChangeError(err error) *args.ChangeEvent {
	return &args.ChangeEvent{
		Err: err,
	}
}
