package argsetcd

import (
	"fmt"
	"path"
	"sync"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/pkg/errors"
	"github.com/thrawn01/args"
	"golang.org/x/net/context"
)

type V3Backend struct {
	Root       string
	Client     *etcd.Client
	done       chan struct{}
	changeChan chan *args.ChangeEvent
	wg         sync.WaitGroup
}

func NewV3Backend(client *etcd.Client, root string) args.Backend {
	return &V3Backend{
		Root:   root,
		Client: client,
	}
}

// Get retrieves a value from a K/V store for the provided key.
func (s *V3Backend) Get(ctx context.Context, key string) (args.Pair, error) {
	resp, err := s.Client.Get(ctx, key)
	if err != nil {
		return args.Pair{}, err
	}
	if len(resp.Kvs) == 0 {
		return args.Pair{}, errors.New(fmt.Sprintf("'%s' not found", key))
	}
	return args.Pair{Key: string(resp.Kvs[0].Key), Value: resp.Kvs[0].Value}, nil
}

// List retrieves all keys and values under a provided key.
func (s *V3Backend) List(ctx context.Context, key string) ([]args.Pair, error) {
	resp, err := s.Client.Get(ctx, key, etcd.WithPrefix())
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
func (s *V3Backend) Set(ctx context.Context, key string, value []byte) error {
	return nil
}

// Watch monitors store for changes to key.
func (s *V3Backend) Watch(ctx context.Context, key string) <-chan *args.ChangeEvent {
	watchChan := s.Client.Watch(ctx, key, etcd.WithPrefix())
	s.changeChan = make(chan *args.ChangeEvent)
	s.done = make(chan struct{})

	s.wg.Add(1)
	go func() {
		var resp etcd.WatchResponse
		var ok bool
		defer s.wg.Done()

		for {
			select {
			case resp, ok = <-watchChan:
				if !ok {
					return
				}
				if resp.Canceled {
					s.changeChan <- NewChangeError(errors.Wrap(resp.Err(),
						"V3Backend.Watch(): ETCD server cancelled watch"))
					return
				}
				for _, event := range resp.Events {
					s.changeChan <- NewChangeEvent(event)
				}
			}
		}
	}()
	return s.changeChan
}

func (s *V3Backend) Close() {
	if s.Client != nil {
		s.Client.Close()
	}
	s.wg.Wait()
	if s.changeChan != nil {
		close(s.changeChan)
	}
}

func (s *V3Backend) GetRootKey() string {
	return s.Root
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
