# Makefile
# go-standard — Phase 7 Tooling

BINARY_NAME := go-standard
BUILD_DIR := ./bin
CMD_DIR := ./cmd/api
MIGRATIONS_DIR := migrations
DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/project_db?sslmode=disable
COVERAGE_THRESHOLD := 80

.PHONY: run build wire mocks swagger test test-integration coverage lint \
        migrate-up migrate-down migrate-create seed \
        docker-dev docker-down docker-build fmt check

# --------------------------------------------------------------------------- #
# Build & Run
# --------------------------------------------------------------------------- #

run: build
	$(BUILD_DIR)/$(BINARY_NAME)

build:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)

# --------------------------------------------------------------------------- #
# Code Generation
# --------------------------------------------------------------------------- #

wire:
	cd internal/di && wire

mocks:
	mockery

swagger:
	swag init -g $(CMD_DIR)/main.go -o docs/swagger --parseDependency --parseInternal

# --------------------------------------------------------------------------- #
# Testing
# --------------------------------------------------------------------------- #

test:
	go test -short -race -count=1 ./...

test-integration:
	go test -race -count=1 -run Integration ./...

coverage:
	@go test -short -race -count=1 -coverprofile=coverage.out ./... \
		-covermode=atomic
	@go tool cover -func=coverage.out
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print substr($$3, 1, length($$3)-1)}'); \
	echo "Total coverage: $$COVERAGE%"; \
	if [ $$(echo "$$COVERAGE < $(COVERAGE_THRESHOLD)" | bc -l) -eq 1 ]; then \
		echo "FAIL: coverage $$COVERAGE% is below threshold $(COVERAGE_THRESHOLD)%"; \
		exit 1; \
	fi

# --------------------------------------------------------------------------- #
# Lint & Format
# --------------------------------------------------------------------------- #

lint:
	golangci-lint run ./...

fmt:
	gofmt -s -w .

# --------------------------------------------------------------------------- #
# Migrations (golang-migrate)
# --------------------------------------------------------------------------- #

migrate-up:
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" down 1

migrate-create:
	@if [ -z "$(name)" ]; then echo "Usage: make migrate-create name=xxx"; exit 1; fi
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq -digits 14 $(name)

# --------------------------------------------------------------------------- #
# Seeds
# --------------------------------------------------------------------------- #

seed:
	@if [ -z "$(env)" ]; then echo "Usage: make seed env=dev"; exit 1; fi
	@for f in seeds/$(env)/*.sql; do \
		echo "Seeding $$f ..."; \
		psql "$(DATABASE_URL)" -f "$$f"; \
	done

# --------------------------------------------------------------------------- #
# Docker
# --------------------------------------------------------------------------- #

docker-dev:
	docker compose -f deployments/docker-compose.dev.yml up -d

docker-down:
	docker compose -f deployments/docker-compose.dev.yml down

docker-build:
	docker build -f deployments/Dockerfile -t $(BINARY_NAME):latest .

# --------------------------------------------------------------------------- #
# CI Pipeline
# --------------------------------------------------------------------------- #

check: lint test coverage