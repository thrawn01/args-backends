[![Coverage Status](https://img.shields.io/coveralls/thrawn01/args-etcd.svg)](https://coveralls.io/github/thrawn01/args-etcd)
[![Build Status](https://img.shields.io/travis/thrawn01/argsetcd/master.svg)](https://travis-ci.org/thrawn01/argsetcd)

## Introduction
This repo provides an etcd key=value storage backend for use with
 [args](http://github.com/thrawn01/args)

## Installation
```
$ go get github.com/thrawn01/argsetcd
```

## Usage
```go
    import (
    	etcd "github.com/coreos/etcd/clientv3"
    	"github.com/thrawn01/args"
    	"github.com/thrawn01/argsetcd"
    )

	parser := args.NewParser()
	parser.AddFlag("--etcd-endpoints").IsStringSlice().Alias("-e").
	    Default("localhost:2379").
		Help("A Comma Separated list of etcd server endpoints")

	client, err := etcd.New(etcd.Config{
		Endpoints:   opts.StringSlice("etcd-endpoints"),
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}

	backend := argsetcd.NewV3Backend(client, "/etcd-root")

	// Read the config and arguments from etcd
	opts, err = parser.FromBackend(backend)
	if err != nil {
		fmt.Printf("Etcd error - %s\n", err.Error())
	}

	// Watch etcd for any configuration changes
	cancelWatch := parser.Watch(backend, func(event *args.ChangeEvent, err error) {
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Printf("Change Event - %+v\n", event)
		// This takes a ChangeEvent and update the opts with the latest changes
		parser.Apply(opts.FromChangeEvent(event))
	})
	// Stop the watcher
	cancelWatch()
```

## Examples
More example functions are available in the `cli/` package and a cli called 
`args-etcd` which provides a complete cli and server example.

## Development Guide
Fetch the source
```
$ mkdir -p $GOPATH/src/github.com/thrawn01
$ cd $GOPATH/src/github.com/thrawn01
$ git clone git@github.com:thrawn01/argsetcd.git
```

Install glide and fetch the dependencies via glide
```
$ make glide-deps
```

Build the examples and run the tests (requires docker)
```
$ make
$ make test
```

