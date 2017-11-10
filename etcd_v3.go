package argsetcd

import (
	"fmt"
	"strings"
	"sync"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/pkg/errors"
	"github.com/thrawn01/args"
	"golang.org/x/net/context"
)

var KeySeparator string = "/"

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
func (s *V3Backend) Get(ctx context.Context, key args.Key) (args.Pair, error) {
	etcdKey := fmt.Sprintf("%s%s%s", s.Root, KeySeparator, key.Join(KeySeparator))
	resp, err := s.Client.Get(ctx, etcdKey)
	if err != nil {
		return args.Pair{}, err
	}
	if len(resp.Kvs) == 0 {
		return args.Pair{}, errors.New(fmt.Sprintf("'%s' not found", etcdKey))
	}
	return args.Pair{Key: key, Value: string(resp.Kvs[0].Value)}, nil
}

// List retrieves all keys and values under a provided key.
func (s *V3Backend) List(ctx context.Context, key args.Key) ([]args.Pair, error) {
	etcdKey := fmt.Sprintf("%s%s%s", s.Root, KeySeparator, key.Join(KeySeparator))
	resp, err := s.Client.Get(ctx, etcdKey, etcd.WithPrefix())
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) == 0 {
		return nil, errors.New(fmt.Sprintf("%s not found", etcdKey))
	}
	result := make([]args.Pair, 0)
	for _, node := range resp.Kvs {
		result = append(result, args.Pair{
			Key: args.Key{
				Group: key.Group,
				Name:  baseName(string(node.Key)),
			},
			Value: string(node.Value),
		})
	}
	return result, nil
}

// Set the provided key to value.
func (s *V3Backend) Set(ctx context.Context, key args.Key, value string) error {
	return nil
}

// Watch monitors store for changes to key.
func (s *V3Backend) Watch(ctx context.Context, root string) <-chan *args.ChangeEvent {
	watchChan := s.Client.Watch(ctx, root, etcd.WithPrefix())
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
	// Given a key of `/root/group/key` separate the group and key
	parts := strings.Split(string(event.Kv.Key), KeySeparator)
	if len(parts) < 2 {
		parts = []string{"key", "invalid-group", "invalid-key"}
	}
	return &args.ChangeEvent{
		Key: args.Key{
			Name:  parts[len(parts)-1],
			Group: parts[len(parts)-2],
		},
		Value:   string(event.Kv.Value),
		Deleted: event.Type.String() == "DELETE",
		Err:     nil,
	}
}

func NewChangeError(err error) *args.ChangeEvent {
	return &args.ChangeEvent{
		Err: err,
	}
}

func baseName(key string) string {
	if i := strings.LastIndex(key, KeySeparator); i >= 0 {
		return key[i+1:]
	}
	return key
}
