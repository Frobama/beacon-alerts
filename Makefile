CONTRACTS_URL ?= https://github.com/Frobama/beacon-contracts.git
CONTRACTS_TAG ?= v0.2.0
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
TEST_DATABASE_URL ?= postgres://beacon:beacon@localhost:5433/alerts_test?sslmode=disable
THRESHOLD ?= 60

.PHONY: proto build test test-integration run docker coverage

proto: ## Genera los stubs desde beacon-contracts
	buf generate "$(CONTRACTS_URL)#tag=$(CONTRACTS_TAG)" --template buf.gen.yaml

build: ## Compila el binario
	go build -o bin/beacon-alerts ./cmd/alerts

test: ## Corre los tests unitarios (los de base se saltan sin TEST_DATABASE_URL)
	go test ./... -race

test-integration: ## Corre todo, incluidos los tests de base
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test ./... -race

run: ## Ejecuta el servicio en local
	go run ./cmd/alerts

docker: ## Construye la imagen con versión y commit
	docker build --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) -t beacon-alerts:dev .

coverage: ## Falla si la cobertura total baja del umbral
	bash scripts/check-coverage.sh $(THRESHOLD)
