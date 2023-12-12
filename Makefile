GOPATH := $(shell go env GOPATH)

build:
	go build
	cp good $(GOPATH)/bin/