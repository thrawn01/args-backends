package main

import (
	"fmt"
	"os"
	"time"

	etcdv3 "github.com/coreos/etcd/clientv3"
	"github.com/thrawn01/args"
)

func main() {
	parser := args.NewParser(args.Name("args-etcd"), args.Desc("An args-etcd example CLI", args.IsFormated))
	parser.AddOption("--endpoints").Required().IsStringSlice().
		Help("A comma seperated list of etcd endpoints")
	parser.AddOption("--version").Default("3").IsInt().Env("ETCDCTL_API").
		Help("what version of etcd api should we use")
	parser.AddCommand("config", v3Config)
	parser.AddCommand("endpoints", v3Endpoints)

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
