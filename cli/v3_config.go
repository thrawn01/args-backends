package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	etcdv3 "github.com/coreos/etcd/clientv3"
	"github.com/thrawn01/args"
	"github.com/thrawn01/argsetcd"
)


func V3ConfigSet(parser *args.Parser, data interface{}) (int, error) {
	parser.Desc(args.Dedent(`set config items in etcd for the server to pickup

	Examples:
		$ args-etcd config set name "James Dean"
		$ args-etcd config set age 12
		$ args-etcd config set sex male
		$ args-etcd config set config-version 1`))
	parser.AddArgument("key").Required().Help("The key to set")
	parser.AddArgument("value").Required().Help("The value to set")
	opts := parser.ParseSimple(nil)
	if opts == nil {
		return 1, nil
	}

	// Get our config options
	configs := args.NewParser()
	addConfigOptions(configs)

	// Ensure we only allow the user to set these config options
	rule := configs.GetRule(opts.String("key"))
	if rule == nil {
		fmt.Printf("Invalid config name '%s' valid options are:\n", opts.String("key"))
		for _, rule := range configs.GetRules() {
			fmt.Printf("- %s\n", rule.Name)
		}
		return 1, nil
	}

	// Ask the rule for our backend key
	key := rule.BackendKey("/args-config")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Put the key
	client := data.(*etcdv3.Client)
	fmt.Printf("Set Config '%s' - '%s'\n", key, opts.String("value"))
	if _, err := client.Put(ctx, key, opts.String("value")); err != nil {
		return 1, err
	}
	return 0, nil
}

func addConfigOptions(parser *args.Parser) {
	parser.AddConfig("name").Help("The name of our user")
	parser.AddConfig("age").IsInt().Help("The age of our user")
	parser.AddConfig("sex").Help("The sex of our user")
	parser.AddConfig("config-version").IsInt().Default("0").
		Help("When version is changed, the service will update the config")
}

func V3ConfigServer(parser *args.Parser, data interface{}) (int, error) {
	// A Command line only option
	parser.AddFlag("--bind").Alias("-b").Default("localhost:1234").
		Help("Interface to bind the server too")

	// Create some configuration items we can read from etcd
	addConfigOptions(parser)
	opts := parser.ParseSimple(nil)
	if opts == nil {
		return 1, nil
	}

	// Simple handler that returns a list of endpoints to the caller
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// GetOpts is a thread safe way to get the current options
		conf := parser.GetOpts()

		// Marshal the endpoints and our api-key to json
		payload, err := json.Marshal(map[string]interface{}{
			"name":           conf.String("name"),
			"age":            conf.Int("age"),
			"sex":            conf.String("sex"),
			"config-version": conf.Int("config-version"),
		})
		if err != nil {
			fmt.Println("error:", err)
		}
		// Write the response to the user
		w.Header().Set("Content-Type", "application/json")
		w.Write(payload)
	}))
	srv := &http.Server{Addr: opts.String("bind"), Handler: mux}

	backend := argsetcd.NewV3Backend(data.(*etcdv3.Client), "/args-config")
	// Read the config values from etcd
	opts, err := parser.FromBackend(backend)
	if err != nil {
		fmt.Printf("Etcd error - %s\n", err.Error())
	}

	// Watch etcd for any configuration changes
	stagedOpts := parser.GetOpts()
	cancelWatch := parser.Watch(backend, func(event *args.ChangeEvent, err error) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "-- %s\n", err.Error())
			return
		}

		fmt.Printf("Change Event - %+v\n", event)
		// This takes a ChangeEvent and updates the stagedOpts with the latest changes
		stagedOpts.FromChangeEvent(event)

		// Only apply our config change once the config values
		// have all be collected and our version number changes
		if event.KeyName == "config-version" {
			// Apply the new config to the parser
			opts, err := parser.Apply(stagedOpts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "-- %s\n", err.Error())
				return
			}
			// Clear the staged config values
			stagedOpts = parser.GetOpts()
			fmt.Printf("Config updated to version %d\n", opts.Int("config-version"))
		}
	})

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, os.Kill)

	go func() {
		fmt.Printf("Listening for requests on %s...\n", opts.String("bind"))
		if err := srv.ListenAndServe(); err != nil {
			fmt.Printf("Serve error: %s\n", err)
			syscall.Kill(syscall.Getpid(), syscall.SIGINT)
		}
	}()

	sig := <-signalChan
	fmt.Printf("Captured %v. Exiting...\n", sig)

	backend.Close()
	cancelWatch()

	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	srv.Shutdown(ctx)

	return 0, nil
}
