.PHONY: all build run test test-unit test-integration test-jobs test-coverage test-coverage-html test-ci test-report test-report-quick test-report-ci test-open-report clean proto docker docker-compose docker-test docker-test-build docker-test-up docker-test-down docker-test-logs docker-test-shell test-with-docker test-monolithic help

# Variables
APP_NAME := arcana-cloud-go
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "1.0.0")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags "-w -s -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := gofmt

# Directories
CMD_DIR := ./cmd/server
BIN_DIR := ./bin
PROTO_DIR := ./api/proto
PROTO_OUT := ./api/proto/pb

all: clean build

## build: Build the application
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(APP_NAME) $(CMD_DIR)

## run: Run the application
run: build
	@echo "Running $(APP_NAME)..."
	$(BIN_DIR)/$(APP_NAME)

## dev: Run with hot reload (requires air)
dev:
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "Installing air..."; \
		go install github.com/cosmtrek/air@latest; \
		air; \
	fi

## test: Run all unit tests (requires Redis)
test:
	@echo "Running all unit tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

## test-unit: Run unit tests without external dependencies
test-unit:
	@echo "Running unit tests (no external dependencies)..."
	$(GOTEST) -v -race -short -coverprofile=coverage.out ./...

## test-jobs: Run only job system tests
test-jobs:
	@echo "Running job system tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./internal/jobs/...

## test-integration: Run integration tests with Docker Compose
test-integration: test-env-up
	@echo "Running integration tests..."
	@sleep 3
	$(GOTEST) -v -race -tags=integration -coverprofile=coverage.integration.out ./tests/integration/...
	@$(MAKE) test-env-down

## test-integration-only: Run integration tests (assumes env is already up)
test-integration-only:
	@echo "Running integration tests (environment must be running)..."
	$(GOTEST) -v -race -tags=integration -coverprofile=coverage.integration.out ./tests/integration/...

## test-coverage: Generate coverage report for all tests
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@echo "Coverage report generated: coverage.out"

## test-coverage-html: Generate HTML coverage report
test-coverage-html: test-coverage
	@echo "Generating HTML coverage report..."
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"
	@$(GOCMD) tool cover -func=coverage.out | tail -1

## test-coverage-jobs: Generate coverage for job system only
test-coverage-jobs:
	@echo "Generating job system coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.jobs.out -covermode=atomic ./internal/jobs/...
	$(GOCMD) tool cover -html=coverage.jobs.out -o coverage.jobs.html
	@echo "Job system coverage report: coverage.jobs.html"
	@$(GOCMD) tool cover -func=coverage.jobs.out | tail -1

## test-ci: Run all tests in CI mode (with environment)
test-ci: test-env-up
	@echo "Running CI tests..."
	@sleep 5
	$(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GOTEST) -v -race -tags=integration -coverprofile=coverage.integration.out -covermode=atomic ./tests/integration/...
	@$(MAKE) test-env-down
	@echo "All CI tests completed"

## test-env-up: Start test environment (Redis + MySQL)
test-env-up:
	@echo "Starting test environment..."
	docker-compose -f docker-compose.test.yml up -d
	@echo "Waiting for services to be ready..."
	@sleep 3

## test-env-down: Stop test environment
test-env-down:
	@echo "Stopping test environment..."
	docker-compose -f docker-compose.test.yml down -v

## test-bench: Run benchmarks
test-bench:
	@echo "Running benchmarks..."
	$(GOTEST) -v -bench=. -benchmem -run=^$$ ./internal/jobs/...

## test-bench-jobs: Run job system benchmarks
test-bench-jobs:
	@echo "Running job system benchmarks..."
	$(GOTEST) -v -bench=. -benchmem -run=^$$ ./internal/jobs/...

## test-report: Generate fancy HTML test reports to docs directory
test-report:
	@echo "Generating comprehensive test report..."
	@chmod +x scripts/generate-test-report.sh
	@./scripts/generate-test-report.sh

## test-report-quick: Generate quick test report (unit tests only)
test-report-quick:
	@echo "Generating quick test report..."
	@mkdir -p docs/coverage docs/test-reports
	$(GOTEST) -v -race -short -coverprofile=docs/coverage/coverage.out -covermode=atomic ./... 2>&1 | tee docs/test-reports/test-output.txt || true
	$(GOCMD) tool cover -html=docs/coverage/coverage.out -o docs/coverage/coverage.html
	$(GOCMD) tool cover -func=docs/coverage/coverage.out > docs/coverage/coverage-func.txt
	@echo "Report generated: docs/coverage/coverage.html"
	@$(GOCMD) tool cover -func=docs/coverage/coverage.out | tail -1

## test-report-ci: Generate CI-compatible test report with JUnit XML
test-report-ci:
	@echo "Generating CI test report..."
	@mkdir -p docs/coverage docs/test-reports
	@if command -v gotestsum > /dev/null; then \
		gotestsum --junitfile docs/test-reports/junit.xml --format testname -- -v -race -coverprofile=docs/coverage/coverage.out -covermode=atomic ./...; \
	else \
		echo "Installing gotestsum..."; \
		go install gotest.tools/gotestsum@latest; \
		gotestsum --junitfile docs/test-reports/junit.xml --format testname -- -v -race -coverprofile=docs/coverage/coverage.out -covermode=atomic ./...; \
	fi
	$(GOCMD) tool cover -html=docs/coverage/coverage.out -o docs/coverage/coverage.html
	$(GOCMD) tool cover -func=docs/coverage/coverage.out > docs/coverage/coverage-func.txt
	@echo "Reports generated in docs/ directory"

## test-open-report: Open test report in browser
test-open-report:
	@if [ -f docs/test-reports/summary.html ]; then \
		open docs/test-reports/summary.html 2>/dev/null || xdg-open docs/test-reports/summary.html 2>/dev/null || echo "Open docs/test-reports/summary.html in your browser"; \
	elif [ -f docs/coverage/coverage.html ]; then \
		open docs/coverage/coverage.html 2>/dev/null || xdg-open docs/coverage/coverage.html 2>/dev/null || echo "Open docs/coverage/coverage.html in your browser"; \
	else \
		echo "No report found. Run 'make test-report' first."; \
	fi

## lint: Run linters
lint:
	@echo "Running linters..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run ./...; \
	fi

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

## proto: Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	@mkdir -p $(PROTO_OUT)
	protoc --go_out=$(PROTO_OUT) --go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_OUT) --go-grpc_opt=paths=source_relative \
		-I $(PROTO_DIR) $(PROTO_DIR)/*.proto

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BIN_DIR)
	@rm -f coverage.out coverage.html coverage.integration.out coverage.jobs.out coverage.jobs.html

## docker: Build Docker image
docker:
	@echo "Building Docker image..."
	docker build -t $(APP_NAME):$(VERSION) -t $(APP_NAME):latest .

## docker-compose: Run with Docker Compose (monolithic)
docker-compose:
	@echo "Starting with Docker Compose..."
	docker-compose -f deployment/docker/docker-compose.yaml up --build

## docker-compose-layered: Run with Docker Compose (layered)
docker-compose-layered:
	@echo "Starting with Docker Compose (layered)..."
	docker-compose -f deployment/docker/docker-compose.layered.yaml up --build

## docker-compose-down: Stop Docker Compose
docker-compose-down:
	docker-compose -f deployment/docker/docker-compose.yaml down -v

## k8s-deploy: Deploy to Kubernetes
k8s-deploy:
	@echo "Deploying to Kubernetes..."
	kubectl apply -f deployment/kubernetes/namespace.yaml
	kubectl apply -f deployment/kubernetes/configmap.yaml
	kubectl apply -f deployment/kubernetes/secret.yaml
	kubectl apply -f deployment/kubernetes/deployment.yaml
	kubectl apply -f deployment/kubernetes/hpa.yaml
	kubectl apply -f deployment/kubernetes/ingress.yaml

## k8s-delete: Delete from Kubernetes
k8s-delete:
	@echo "Deleting from Kubernetes..."
	kubectl delete -f deployment/kubernetes/ --ignore-not-found

## swagger: Generate Swagger documentation
swagger:
	@echo "Generating Swagger documentation..."
	@if command -v swag > /dev/null; then \
		swag init -g cmd/server/main.go -o docs; \
	else \
		echo "Installing swag..."; \
		go install github.com/swaggo/swag/cmd/swag@latest; \
		swag init -g cmd/server/main.go -o docs; \
	fi

## migrate: Run database migrations
migrate:
	@echo "Running migrations..."
	$(GOBUILD) -o $(BIN_DIR)/migrate ./cmd/migrate
	$(BIN_DIR)/migrate up

## docker-test: Run all tests in Docker with MySQL and Redis (Monolithic mode)
docker-test:
	@echo "Starting Docker test environment..."
	@mkdir -p coverage test-reports
	docker-compose -f docker-compose.test.yml up --build --abort-on-container-exit test-runner
	@echo "Tests completed. Reports available in ./coverage and ./test-reports"
	@$(MAKE) docker-test-down

## docker-test-build: Build the test Docker image
docker-test-build:
	@echo "Building test Docker image..."
	docker-compose -f docker-compose.test.yml build test-runner

## docker-test-up: Start test environment (MySQL + Redis) without running tests
docker-test-up:
	@echo "Starting test environment..."
	docker-compose -f docker-compose.test.yml up -d mysql-test redis-test
	@echo "Waiting for services to be healthy..."
	@docker-compose -f docker-compose.test.yml exec mysql-test mysqladmin ping -h localhost -u root -ptestroot --wait=30 2>/dev/null || sleep 10
	@echo "Test environment ready!"
	@echo "MySQL: localhost:3307 (user: arcana_test, password: arcana_test)"
	@echo "Redis: localhost:6380"

## docker-test-down: Stop and remove test environment
docker-test-down:
	@echo "Stopping test environment..."
	docker-compose -f docker-compose.test.yml down -v

## docker-test-logs: Show test container logs
docker-test-logs:
	docker-compose -f docker-compose.test.yml logs -f

## docker-test-shell: Open shell in test runner container
docker-test-shell:
	docker-compose -f docker-compose.test.yml run --rm test-runner sh

## test-with-docker: Run tests locally using Docker services
test-with-docker: docker-test-up
	@echo "Running tests with Docker services..."
	@sleep 3
	ARCANA_DATABASE_HOST=localhost \
	ARCANA_DATABASE_PORT=3307 \
	ARCANA_DATABASE_NAME=arcana_test \
	ARCANA_DATABASE_USER=arcana_test \
	ARCANA_DATABASE_PASSWORD=arcana_test \
	ARCANA_REDIS_HOST=localhost \
	ARCANA_REDIS_PORT=6380 \
	JWT_SECRET=test-secret-key-for-testing-purposes-only-32chars \
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	@$(MAKE) docker-test-down

## test-monolithic: Run tests in monolithic mode with Docker
test-monolithic: docker-test-up
	@echo "Running tests in Monolithic mode..."
	@sleep 3
	ARCANA_APP_ENVIRONMENT=test \
	ARCANA_DATABASE_HOST=localhost \
	ARCANA_DATABASE_PORT=3307 \
	ARCANA_DATABASE_NAME=arcana_test \
	ARCANA_DATABASE_USER=arcana_test \
	ARCANA_DATABASE_PASSWORD=arcana_test \
	ARCANA_REDIS_HOST=localhost \
	ARCANA_REDIS_PORT=6380 \
	ARCANA_DEPLOYMENT_MODE=monolithic \
	ARCANA_DEPLOYMENT_LAYER= \
	ARCANA_DEPLOYMENT_PROTOCOL=http \
	JWT_SECRET=test-secret-key-for-testing-purposes-only-32chars \
	$(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@$(GOCMD) tool cover -func=coverage.out | tail -1
	@$(MAKE) docker-test-down

## help: Show this help message
help:
	@echo "Arcana Cloud Go - Available commands:"
	@echo ""
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
