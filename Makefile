.PHONY: start-containers stop-containers test examples all
.DEFAULT_GOAL := all

# GO
GOPATH := $(shell go env | grep GOPATH | sed 's/GOPATH="\(.*\)"/\1/')
GLIDE := $(GOPATH)/bin/glide
PATH := $(GOPATH)/bin:$(PATH)
export $(PATH)

export ARGS_DOCKER_HOST=localhost
DOCKER_MACHINE_IP=$(shell docker-machine ip default 2> /dev/null)
ifneq ($(DOCKER_MACHINE_IP),)
	ARGS_DOCKER_HOST=$(DOCKER_MACHINE_IP)
endif

ETCD_DOCKER_IMAGE=quay.io/coreos/etcd:latest

start-containers:
	@echo Checking Docker Containers
	@if [ $(shell docker ps -a | grep -ci args-etcd) -eq 0 ]; then \
		echo Starting Docker Container args-etcd; \
		docker run -d -v /usr/share/ca-certificates/:/etc/ssl/certs -p 4001:4001 -p 2380:2380 -p 2379:2379 \
		--name args-etcd $(ETCD_DOCKER_IMAGE) /usr/local/bin/etcd \
		--name etcd0 \
		--advertise-client-urls http://${ARGS_DOCKER_HOST}:2379,http://${ARGS_DOCKER_HOST}:4001 \
		--listen-client-urls http://0.0.0.0:2379,http://0.0.0.0:4001 \
		--initial-advertise-peer-urls http://${ARGS_DOCKER_HOST}:2380 \
		--listen-peer-urls http://0.0.0.0:2380 \
		--initial-cluster-token etcd-cluster-1 \
		--initial-cluster etcd0=http://${ARGS_DOCKER_HOST}:2380 \
		--initial-cluster-state new; \
	elif [ $(shell docker ps | grep -ci args-etcd) -eq 0 ]; then \
		echo restarting args-etcd; \
		docker start args-etcd > /dev/null; \
	fi

stop-containers:
	@if [ $(shell docker ps -a | grep -ci args-etcd) -eq 1 ]; then \
		echo Stopping Container args-etcd; \
		docker stop args-etcd > /dev/null; \
	fi

test: start-containers
	@echo Running Tests
	@go test .

bin/etcd-config-service: examples/etcd-config-service.go
	go build -o bin/etcd-config-service examples/etcd-config-service.go

bin/etcd-config-client: examples/etcd-config-client.go
	go build -o bin/etcd-config-client examples/etcd-config-client.go

bin/etcd-endpoints-service: examples/etcd-endpoints-service.go
	go build -o bin/etcd-endpoints-service examples/etcd-endpoints-service.go

bin/etcd-endpoints-client: examples/etcd-endpoints-client.go
	go build -o bin/etcd-endpoints-client examples/etcd-endpoints-client.go

all: examples

examples: bin/etcd-endpoints-service bin/etcd-endpoints-client bin/etcd-config-service bin/etcd-config-client

clean:
	rm bin/*

travis-ci: get-deps start-containers
	go get -u github.com/mattn/goveralls
	go get -u golang.org/x/tools/cmd/cover
	goveralls -service=travis-ci

$(GLIDE):
	go get -u github.com/Masterminds/glide

glide-deps: $(GLIDE)
	$(GLIDE) install
	go get golang.org/x/net/context

get-deps:
	go get $(go list ./... | grep -v /examples)
