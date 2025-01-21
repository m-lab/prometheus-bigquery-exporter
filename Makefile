GO := go

DOCKER_IMAGE_NAME ?= prometheus-bigquery-exporter
DOCKER_IMAGE_TAG ?= $(subst /,-,$(shell git rev-parse --abbrev-ref HEAD))

COLOR_BLUE := \033[0;34m
COLOR_RED := \033[0;31m
COLOR_GREEN := \033[0;32m
COLOR_YELLOW := \033[0;33m
COLOR_RESET := \033[0m


all: vet build
.PHONY: all build docker needs-query needs-project needs-gcloud-config needs-gcloud-sa-key run-query

vet:
	@echo "$(COLOR_BLUE)>> Vetting code$(COLOR_RESET)"
	@$(GO) vet $(pkgs)

build:
	@$(GO) get -t .

docker:
	@echo "$(COLOR_BLUE)>> Building docker image$(COLOR_RESET)"
	@docker build -t "$(DOCKER_IMAGE_NAME) -t "$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)" .

needs-query:
	@echo "$(COLOR_BLUE)>> Checking query file$(COLOR_RESET)"
	@if [ -z "$(query)" ]; then \
		echo "$(COLOR_RED)ERROR: Please provide a query file with query=<file>.sql$(COLOR_RESET)"; \
		exit 1; \
	fi
	@if [ ! -f "$(PWD)/ahoy/prometheus-bigquery-exporter/queries/$(query)" ]; then \
		echo "$(COLOR_RED)ERROR: Query file $(PWD)/ahoy/prometheus-bigquery-exporter/queries/$(query) does not exist$(COLOR_RESET)"; \
		exit 1; \
	fi

needs-project:
	@echo "$(COLOR_BLUE)>> Checking project ID$(COLOR_RESET)"
	@if [ -z "$(project)" ]; then \
		echo "$(COLOR_RED)ERROR: Please provide the Google project ID (numeric) with project=XXXXX$(COLOR_RESET)"; \
		exit 1; \
	fi

needs-gcloud-config:
	@echo "$(COLOR_BLUE)>> Checking gcloud config directory exists$(COLOR_RESET)"
	@if [ ! -d "$(HOME)/.config/gcloud" ]; then \
		echo "$(COLOR_RED)ERROR: Please run gcloud auth login and gcloud config set project PROJECT_ID$(COLOR_RESET)"; \
		exit 1; \
	fi

needs-gcloud-sa-key:
	@echo "$(COLOR_BLUE)>> Checking service account key file exists$(COLOR_RESET)"
	@if [ ! -f "$(PWD)/key.json" ]; then \
		echo "$(COLOR_RED)ERROR: Please ensure service account key exists at $(PWD)/key.json$(COLOR_RESET)"; \
		exit 1; \
	fi

run-query: needs-gcloud-config needs-gcloud-sa-key needs-project needs-query 
	@echo "$(COLOR_BLUE)>> Running query from $(COLOR_YELLOW)$(query)$(COLOR_RESET)"
	@echo "$(COLOR_BLUE)>> Using project ID $(COLOR_YELLOW)$(project)$(COLOR_RESET)"
	@echo "$(COLOR_BLUE)>> Review generated metrics at: $(COLOR_YELLOW)http://localhost:9348/metrics$(COLOR_RESET)"
	docker run -p 9348:9348 --rm \
		-v $(HOME)/.config/gcloud:/root/.config/gcloud \
		-v $(PWD)/key.json:/key.json \
		-v $(PWD)/ahoy/prometheus-bigquery-exporter/queries:/queries -it "$(DOCKER_IMAGE_NAME)" \
		-project=$(project) \
		-gauge-query=/queries/$(query)
