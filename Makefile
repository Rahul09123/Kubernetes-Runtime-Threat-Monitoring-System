APP ?= krtms
REGISTRY ?= ghcr.io/your-org
TAG ?= latest

.PHONY: tidy test build docker deploy

tidy:
	go mod tidy

test:
	go test ./...

build:
	go build ./...

docker:
	docker build -f deployments/docker/Dockerfile.event-collector -t $(REGISTRY)/event-collector:$(TAG) .
	docker build -f deployments/docker/Dockerfile.analyzer -t $(REGISTRY)/analyzer:$(TAG) .
	docker build -f deployments/docker/Dockerfile.alert-manager -t $(REGISTRY)/alert-manager:$(TAG) .

deploy:
	kubectl apply -f deployments/k8s/base
	kubectl apply -f deployments/k8s/monitoring
