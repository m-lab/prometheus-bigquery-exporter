GO := go
DOCKER_IMAGE_NAME ?= prometheus-bigquery-exporter
DOCKER_IMAGE_TAG ?= $(subst /,-,$(shell git rev-parse --abbrev-ref HEAD))


all: vet build

vet:
	@echo ">> vetting code"
	@$(GO) vet $(pkgs)

build:
	@$(GO) get -t .

docker:
	@echo ">> building docker image"
	@docker build -t "$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)" .

.PHONY: all build docker
