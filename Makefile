SHELL := /bin/bash

watch: tools/modd/bin/modd
	tools/modd/bin/modd

run-with-env: .env
	( set -o allexport && source .env && set +o allexport && $(value CMD))

build: build-server

build-%: generate
	CGO_ENABLED=0 \
		go build \
			-o ./bin/$* ./cmd/$*

purge:
	rm -rf data

generate: tools/templ/bin/templ
	tools/templ/bin/templ generate

bin/templ: tools/templ/bin/templ
	mkdir -p bin
	ln -fs $(PWD)/tools/templ/bin/templ bin/templ

tools/templ/bin/templ:
	mkdir -p tools/templ/bin
	GOBIN=$(PWD)/tools/templ/bin go install github.com/a-h/templ/cmd/templ@v0.3.960

tools/modd/bin/modd:
	mkdir -p tools/modd/bin
	GOBIN=$(PWD)/tools/modd/bin go install github.com/cortesi/modd/cmd/modd@latest

GORELEASER_ARGS ?= release --auto-snapshot --clean

goreleaser: tools/goreleaser/bin/goreleaser
	REPO_OWNER=$(shell whoami) tools/goreleaser/bin/goreleaser $(GORELEASER_ARGS)

tools/goreleaser/bin/goreleaser:
	mkdir -p tools/goreleaser/bin
	GOBIN=$(PWD)/tools/goreleaser/bin go install github.com/goreleaser/goreleaser/v2@latest

.env:
	cp .env.dist .env

-include misc/*/*.mk

