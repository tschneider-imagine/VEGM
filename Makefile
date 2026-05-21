GO ?= go

.PHONY: fmt test build run

fmt:
	$(GO) fmt ./...

test:
	$(GO) test ./...

build:
	$(GO) build ./cmd/vegm

run:
	$(GO) run ./cmd/vegm -config ./example.vegm.json
