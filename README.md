[![Coverage Status](https://img.shields.io/coveralls/thrawn01/args-backends.svg)](https://coveralls.io/github/thrawn01/args)
[![Build Status](https://img.shields.io/travis/thrawn01/args-backends/master.svg)](https://travis-ci.org/thrawn01/args)

**NOTE: This is alpha software, the api will continue to evolve**

## Introduction
This repo provides the key=value storage backends for use with
 [args](http://github.com/thrawn01/args)

## Installation
```
go get github.com/thrawn01/args-backends
```

## Development Guide
Fetch the source
```
go get -d github.com/thrawn01/args-backends
cd $GOPATH/src/github.com/thrawn01/args-backends
```

Install glide and fetch the dependencies via glide
```
make get-deps
```

Run make to build the example and run the tests
```
make test
```

