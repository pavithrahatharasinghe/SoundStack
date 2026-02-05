SHELL := /bin/bash
APP := soundstack

.PHONY: build test run

build:
	go build ./cmd/soundstack

test:
	go test ./...

run:
	go run ./cmd/$(APP)

