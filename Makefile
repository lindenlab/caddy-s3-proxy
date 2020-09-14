
.PHONY: build
build:
	@go build

.PHONY: docker
docker:  ## build a docker image for caddy with the s3proxy
	@docker build -t caddy .

.PHONY: lint
lint:  ## Run golint on source base
	@golangci-lint run --no-config ./...

.DEFAULT_GOAL := help
.PHONY: help
help:   ## Display this help message
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

.PHONY: version
version:  ## Show the version the Makefile will build
	@echo ${VERSION}
	@echo client version: $(shell cat client/Version)
