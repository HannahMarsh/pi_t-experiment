include .env
export

all: build test

run: run-bulletin-board run-node run-client run-metrics run-proxy run-web

run-bulletin-board:
	cd cmd/bulletin-board && go mod tidy && go mod download && \
	CGO_ENABLED=0 go run github.com/HannahMarsh/pi_t-experiment/cmd/bulletin-board
.PHONY: run-bulletin-board

run-node:
	cd cmd/node && go mod tidy && go mod download && \
	CGO_ENABLED=0 go run -tags migrate github.com/HannahMarsh/pi_t-experiment/cmd/node
.PHONY: run-node

run-client:
	cd cmd/client && go mod tidy && go mod download && \
	CGO_ENABLED=0 go run -tags migrate github.com/HannahMarsh/pi_t-experiment/cmd/client
.PHONY: run-client

run-metrics:
	cd cmd/metrics && go mod tidy && go mod download && \
	CGO_ENABLED=0 go run -tags migrate github.com/HannahMarsh/pi_t-experiment/cmd/metrics
.PHONY: run-metrics

run-proxy:
	cd cmd/proxy && go mod tidy && go mod download && \
	CGO_ENABLED=0 go run -tags migrate github.com/HannahMarsh/pi_t-experiment/cmd/proxy
.PHONY: run-proxy

run-web:
	cd cmd/web && go mod tidy && go mod download && \
	CGO_ENABLED=0 go run github.com/HannahMarsh/pi_t-experiment/cmd/web
.PHONY: run-web

docker-compose: docker-compose-stop docker-compose-start
.PHONY: docker-compose

docker-compose-start:
	docker-compose up --build
.PHONY: docker-compose-start

docker-compose-stop:
	docker-compose down --remove-orphans -v
.PHONY: docker-compose-stop

docker-compose-core: docker-compose-core-stop docker-compose-core-start

docker-compose-core-start:
	docker-compose -f docker-compose-core.yaml up --build
.PHONY: docker-compose-core-start

docker-compose-core-stop:
	docker-compose -f docker-compose-core.yaml down --remove-orphans -v
.PHONY: docker-compose-core-stop

docker-compose-build:
	docker-compose down --remove-orphans -v
	docker-compose build
.PHONY: docker-compose-build

wire:
	cd internal/client/app && wire && cd - && \
	cd internal/node/app && wire && cd - && \
	cd internal/metrics/app && wire && cd - && \
	cd internal/bulletin-board/app && wire && cd -
.PHONY: wire

sqlc:
	sqlc generate
.PHONY: sqlc

test:
	go test -v main.go

linter-golangci: ### check by golangci linter
	golangci-lint run
.PHONY: linter-golangci

clean:
	go clean