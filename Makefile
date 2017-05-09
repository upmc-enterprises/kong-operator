# Makefile for the Docker image upmcenterprises/kong-controller
# MAINTAINER: Steve Sloka <slokas@upmc.edu>

.PHONY: all build container push clean test

TAG ?= 0.0.0
PREFIX ?= upmcenterprises

all: container

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -o _output/bin/kong-operator --ldflags '-w' ./cmd/operator/main.go

container: build
	docker build -t $(PREFIX)/kong-operator:$(TAG) .

push:
	docker push $(PREFIX)/kong-operator:$(TAG)

clean:
	rm -f kong-operator

test: clean
	go test $$(go list ./... | grep -v /vendor/)