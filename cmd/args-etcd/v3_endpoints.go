package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	etcdv3 "github.com/coreos/etcd/clientv3"
	"github.com/thrawn01/args"
	"github.com/thrawn01/args-etcd"
)

func v3Endpoints(parser *args.ArgParser, data interface{}) (int, error) {
	parser.SetDesc(args.Dedent(`
	Example client and server for fetching unknown number of endpoints from etcd
	`))
	parser.AddCommand("server", v3EndpointsServer).Help("Start the endpoints server")
	parser.AddCommand("add", Add).Help("Add an endpoint to the etcd store")
	parser.AddCommand("delete", Delete).Help("Deletes an endpoint from the etcd store")
	return parser.ParseAndRun(nil, data)
}

func Add(parser *args.ArgParser, data interface{}) (int, error) {
	parser.AddPositional("name").Required().Help("The name of the new endpoint")
	parser.AddPositional("url").Required().Help("The url of the new endpoint")
	opts := parser.ParseSimple(nil)
	if opts == nil {
		return 1, nil
	}

	// Create our context
	key := fmt.Sprintf("/etcd-endpoints-service/nginx-endpoints/%s", opts.String("name"))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Put the key
	fmt.Printf("Adding New Endpoint '%s' - '%s'\n", key, opts.String("url"))
	client := data.(*etcdv3.Client)
	if _, err := client.Put(ctx, key, opts.String("url")); err != nil {
		return 1, nil
	}
	return 0, nil
}

func Delete(parser *args.ArgParser, data interface{}) (int, error) {
	parser.AddPositional("name").Required().Help("The name of the endpoint to delete")
	opts := parser.ParseSimple(nil)
	if opts == nil {
		return 1, nil
	}

	// Create our context
	key := fmt.Sprintf("/etcd-endpoints-service/nginx-endpoints/%s", opts.String("name"))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	fmt.Printf("Deleting Endpoint '%s'\n", key)
	client := data.(*etcdv3.Client)
	if _, err := client.Delete(ctx, key); err != nil {
		return 1, err
	}
	return 0, nil
}

func v3EndpointsServer(parser *args.ArgParser, data interface{}) (int, error) {
	parser.AddOption("--bind").Alias("-b").Default("localhost:1234").
		Help("Interface to bind the server too")

	// Create some configuration items we can read from ETCD
	parser.AddConfig("api-key").Alias("-k").Default("default-key").
		Help("A fake api-key")
	// This represents an etcd prefix of /etcd-endpoints-service/nginx-endpoints; any key/value
	// stored under this prefix in etcd will be in the 'nginx-endpoints' group
	parser.AddConfigGroup("nginx-endpoints").
		Help("a list of nginx endpoints")

	// Parse the command line arguments
	opts := parser.ParseSimple(nil)
	if opts == nil {
		return 1, nil
	}

	// Simple handler that returns a list of endpoints to the caller
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// GetOpts is a thread safe way to get the current options
		conf := parser.GetOpts()

		// Convert the nginx-endpoints group to a map
		endpoints := conf.Group("nginx-endpoints").ToMap()

		// Marshal the endpoints and our api-key to json
		payload, err := json.Marshal(map[string]interface{}{
			"endpoints": endpoints,
			"api-key":   conf.String("api-key"),
		})
		if err != nil {
			fmt.Println("error:", err)
		}
		// Write the response to the user
		w.Header().Set("Content-Type", "application/json")
		w.Write(payload)
	})

	backend := argsetcd.NewV3Backend(data.(*etcdv3.Client), "/etcd-endpoints-service")

	// Read all the available config values like 'api-key' or 'nginx-endpoints' from etcd
	opts, err := parser.FromBackend(backend)
	if err != nil {
		fmt.Printf("Etcd error - %s\n", err.Error())
	}

	// Watch etcd for any configuration changes
	cancelWatch := parser.Watch(backend, func(event *args.ChangeEvent, err error) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "-- %s\n", err.Error())
			return
		}
		fmt.Printf("Change Event - %+v\n", event)
		// This takes a ChangeEvent and updates the opts with the latest changes
		parser.Apply(opts.FromChangeEvent(event))
	})

	// Listen and serve requests
	fmt.Printf("Listening for requests on %s", opts.String("bind"))
	if err = http.ListenAndServe(opts.String("bind"), nil); err != nil {
		return 1, err
	}
	cancelWatch()
	return 0, nil
}
