export GO111MODULE=on
VERSION := $(shell cat Version)
COVER_TARGET ?= 30

# Set for running tests against localstack
export AWS_SECRET_ACCESS_KEY=dummy
export AWS_ACCESS_KEY_ID=dummy
export AWS_REGION=dummy
export AWS_ENDPOINT=http://localhost:4566

.PHONY: build
build: caddy

caddy: *.go go.mod Makefile
	# go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest  -- install xcaddy if you don't have it
	xcaddy build --output caddy --with github.com/lindenlab/caddy-s3-proxy=${CURDIR}

.PHONY: docker
docker: caddy  ## build a docker image for caddy with the s3proxy
	@docker build -t caddy .

.PHONY: test
test:  ## Run go test on source base
	@go test --race

.PHONY: cover
cover:  ## Generate test coverage results
	@go test -gcflags=-l --covermode=count -coverprofile cover.profile ${PKGS}
	@go tool cover -html cover.profile -o cover.html
	@go tool cover -func cover.profile -o cover.func
	@tail -n 1 cover.func | awk '{if (int($$3) >= ${COVER_TARGET}) {print "Coverage good: " $$3} else {print "Coverage is less than ${COVER_TARGET}%: " $$3; exit 1}}'

.PHONY: lint
lint:  ## Run golint on source base
	@golangci-lint run ./...

.PHONY: localstack
localstack:  ## Launch localstack to run tests against
	@docker-compose -f example/docker-compose.yml up -d localstack

.PHONY: example
example: docker  ## Run docker-compose up in the example directory
	@docker-compose -f example/docker-compose.yml up

.DEFAULT_GOAL := help
.PHONY: help
help:   ## Display this help message
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

.PHONY: clean
clean:  ## Delete any generated files
	@rm -f caddy

.PHONY: version
version:  ## Show the version the Makefile will build
	@echo ${VERSION}
