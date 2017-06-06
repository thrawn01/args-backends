package main

import (
	"fmt"
	"os"
	"time"

	etcdv3 "github.com/coreos/etcd/clientv3"
	"github.com/thrawn01/args"
	"github.com/thrawn01/argsetcd/cli"
)

func main() {
	parser := args.NewParser().Name("args-etcd").Desc("An args-etcd example CLI", args.IsFormatted)
	parser.AddFlag("--endpoints").Required().IsStringSlice().
		Help("A comma seperated list of etcd endpoints")
	parser.AddFlag("--version").Default("3").IsInt().Env("ETCDCTL_API").
		Help("what version of etcd api should we use")

	parser.AddCommand("config", func(parser *args.Parser, data interface{}) (int, error) {
		parser.Desc(`example client and server for fetching simple config items from etcd`)
		parser.AddCommand("set", cli.V3ConfigSet).Help("Set config values in etcd")
		parser.AddCommand("server", cli.V3ConfigServer).Help("Start the config server")
		return parser.ParseAndRun(nil, data)
	})

	parser.AddCommand("endpoints", func(parser *args.Parser, data interface{}) (int, error) {
		parser.Desc(`Example client and server for fetching unknown number of endpoints from etcd`)
		parser.AddCommand("server", cli.V3EndpointsServer).Help("Start the endpoints server")
		parser.AddCommand("add", cli.V3Add).Help("Add an endpoint to the etcd store")
		parser.AddCommand("delete", cli.V3Delete).Help("Deletes an endpoint from the etcd store")
		return parser.ParseAndRun(nil, data)
	})

	opts := parser.ParseOrExit(nil)

	// Create our Client
	client, err := etcdv3.New(etcdv3.Config{
		Endpoints:   opts.StringSlice("endpoints"),
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "-- %s\n", err.Error())
		os.Exit(1)
	}
	defer client.Close()

	retCode, err := parser.RunCommand(client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "-- %s\n", err.Error())
		os.Exit(1)
	}
	os.Exit(retCode)

}
