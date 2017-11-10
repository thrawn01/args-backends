package argsetcd_test

import (
	"bytes"
	"encoding/base32"
	"fmt"
	"os"
	"path"
	"time"

	"testing"

	etcd "github.com/coreos/etcd/clientv3"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"
	"github.com/thrawn01/args"
	"github.com/thrawn01/argsetcd"
	"golang.org/x/net/context"
)

func TestEtcdV3(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "EtcdV3Backend")
}

type TestLogger struct {
	result string
}

func NewTestLogger() *TestLogger {
	return &TestLogger{""}
}

func (self *TestLogger) Print(stuff ...interface{}) {
	self.result = self.result + fmt.Sprint(stuff...) + "|"
}

func (self *TestLogger) Printf(format string, stuff ...interface{}) {
	self.result = self.result + fmt.Sprintf(format, stuff...) + "|"
}

func (self *TestLogger) Println(stuff ...interface{}) {
	self.result = self.result + fmt.Sprintln(stuff...) + "|"
}

func (self *TestLogger) GetEntry() string {
	return self.result
}

func okToTestEtcd() {
	if os.Getenv("ETCDCTL_ENDPOINTS") == "" {
		Skip("ETCDCTL_ENDPOINTS not set, skipped....")
	}
	if os.Getenv("ETCDCTL_API") != "3" {
		Skip("ETCDCTL_API wrong version number, skipped....")
	}
}

func newRootPath() string {
	var buf bytes.Buffer
	encoder := base32.NewEncoder(base32.StdEncoding, &buf)
	encoder.Write(uuid.NewRandom())
	encoder.Close()
	buf.Truncate(26)
	return path.Join("/args-tests", buf.String())
}

func etcdClientFactory() *etcd.Client {
	if os.Getenv("ETCDCTL_ENDPOINTS") == "" {
		return nil
	}

	client, err := etcd.New(etcd.Config{
		Endpoints:   args.StringToSlice(os.Getenv("ETCDCTL_ENDPOINTS")),
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		Fail(fmt.Sprintf("etcdV3ApiFactory() - %s", err.Error()))
	}
	return client
}

func etcdPut(client *etcd.Client, root, key, value string) {
	// Context Timeout for 2 seconds
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	// Set the value in the etcd store
	_, err := client.Put(ctx, path.Join(root, key), value)
	if err != nil {
		Fail(fmt.Sprintf("etcdPut() - %s", err.Error()))
	}
}

var _ = Describe("V3Backend", func() {
	var client *etcd.Client
	var backend args.Backend
	var etcdRoot string
	var log *TestLogger

	BeforeEach(func() {
		etcdRoot = newRootPath()
		client = etcdClientFactory()
		backend = argsetcd.NewV3Backend(client, etcdRoot)
		log = NewTestLogger()
	})

	AfterEach(func() {
		if backend != nil {
			backend.Close()
		}
	})

	Describe("FromBackend()", func() {
		It("Should fetch 'bind' value from /EtcdRoot/bind", func() {
			okToTestEtcd()

			parser := args.NewParser()
			parser.Log(log)
			parser.AddConfig("bind")

			etcdPut(client, etcdRoot, "/bind", "thrawn01.org:3366")
			opts, err := parser.FromBackend(backend)
			Expect(err).To(BeNil())
			Expect(log.GetEntry()).To(Equal(""))
			Expect(opts.String("bind")).To(Equal("thrawn01.org:3366"))
		})
		It("Should fetch 'endpoints' values from /EtcdRoot/endpoints", func() {
			okToTestEtcd()

			parser := args.NewParser()
			parser.Log(log)
			parser.AddConfigGroup("endpoints")

			etcdPut(client, etcdRoot, "/endpoints/endpoint1", "http://endpoint1.com:3366")

			opts, err := parser.FromBackend(backend)
			Expect(err).To(BeNil())
			Expect(log.GetEntry()).To(Equal(""))
			Expect(opts.Group("endpoints").ToMap()).To(Equal(map[string]interface{}{
				"endpoint1": "http://endpoint1.com:3366",
			}))

			etcdPut(client, etcdRoot, "/endpoints/endpoint2",
				"{ \"host\": \"endpoint2\", \"port\": \"3366\" }")

			opts, err = parser.FromBackend(backend)
			Expect(err).To(BeNil())
			Expect(log.GetEntry()).To(Equal(""))
			Expect(opts.Group("endpoints").ToMap()).To(Equal(map[string]interface{}{
				"endpoint1": "http://endpoint1.com:3366",
				"endpoint2": "{ \"host\": \"endpoint2\", \"port\": \"3366\" }",
			}))
		})
		It("Should be ok if config option not found in etcd store", func() {
			okToTestEtcd()

			parser := args.NewParser()
			parser.Log(log)
			parser.AddConfig("bind")

			etcdPut(client, etcdRoot, "/not-found", "foo")
			opts, err := parser.FromBackend(backend)
			Expect(err).To(BeNil())
			Expect(log.GetEntry()).To(ContainSubstring("not found"))
			Expect(opts.String("bind")).To(Equal(""))
		})
	})
	Describe("WatchEtcd", func() {
		It("Should watch /EtcdRoot/endpoints for new values", func() {
			okToTestEtcd()

			parser := args.NewParser()
			parser.Log(log)
			parser.AddConfigGroup("endpoints")

			etcdPut(client, etcdRoot, "/endpoints/endpoint1", "http://endpoint1.com:3366")

			_, err := parser.FromBackend(backend)
			opts := parser.GetOpts()
			Expect(err).To(BeNil())
			Expect(log.GetEntry()).To(Equal(""))
			Expect(opts.Group("endpoints").ToMap()).To(Equal(map[string]interface{}{
				"endpoint1": "http://endpoint1.com:3366",
			}))

			done := make(chan struct{})

			cancelWatch := parser.Watch(backend, func(event *args.ChangeEvent, err error) {
				// Always check for errors
				if err != nil {
					fmt.Printf("Watch Error - %s\n", err.Error())
					close(done)
					return
				}
				parser.Apply(opts.FromChangeEvent(event))
				// Tell the test to continue, Change event was handled
				close(done)
			})
			// Add a new endpoint
			etcdPut(client, etcdRoot, "/endpoints/endpoint2", "http://endpoint2.com:3366")
			// Wait until the change event is handled
			<-done
			// Stop the watch
			cancelWatch()
			// Get the updated options
			opts = parser.GetOpts()

			Expect(log.GetEntry()).To(Equal(""))
			Expect(opts.Group("endpoints").ToMap()).To(Equal(map[string]interface{}{
				"endpoint1": "http://endpoint1.com:3366",
				"endpoint2": "http://endpoint2.com:3366",
			}))
		})
		// TODO
		It("Should continue to attempt to reconnect if the etcd client disconnects", func() {})
		// TODO
		It("Should apply any change using opt.FromChangeEvent()", func() {})
	})
})
