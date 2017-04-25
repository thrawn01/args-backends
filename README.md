[![Coverage Status](https://img.shields.io/coveralls/thrawn01/args-etcd.svg)](https://coveralls.io/github/thrawn01/args-etcd)
[![Build Status](https://img.shields.io/travis/thrawn01/args-etcd/master.svg)](https://travis-ci.org/thrawn01/args-etcd)

## Introduction
This repo provides an etcd key=value storage backend for use with
 [args](http://github.com/thrawn01/args)

## Installation
```
$ go get github.com/thrawn01/args-backends
```

## Usage
```go
    import (
    	etcd "github.com/coreos/etcd/clientv3"
    	"github.com/thrawn01/args"
    	"github.com/thrawn01/args-backends"
    )

	parser := args.NewParser()
	parser.AddOption("--etcd-endpoints").Alias("-e").Default("localhost:2379").
		Help("A Comma Separated list of etcd server endpoints")

	client, err := etcd.New(etcd.Config{
		Endpoints:   opts.StringSlice("etcd-endpoints"),
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}

	backend := backends.NewEtcdBackend(client, "/etcd-root")

	// Read all the args defined config values from etcd
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
```

## Development Guide
Fetch the source
```
$ mkdir -p $GOPATH/src/github.com/thrawn01
$ cd $GOPATH/src/github.com/thrawn01
$ git clone git@github.com:thrawn01/args-backends.git
```

Install glide and fetch the dependencies via glide
```
$ make glide-deps
```

Run make to build the examples and run the tests
```
$ make test
$ make
```

