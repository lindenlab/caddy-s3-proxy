
.PHONEY: build
build:
	@go build

.PHONEY: docker
docker:  # build a docker image for caddy with the s3proxy
	@docker build -t caddy .
