.PHONY: all audit build up down test test-unit test-integration test-security test-deployment test-performance test-resource lint scan-vuln scan-zap backup restore clean help

SHELL := /bin/bash
API_URL ?= http://127.0.0.1:8080

all: test build

help:
	@echo "Available Makefile commands:"
	@echo "  make audit             - Audit codebase, dependencies, and formatting"
	@echo "  make build             - Compile Go backend binary and build frontend"
	@echo "  make up                - Start core Docker containers (Caddy, App, Postgres, Redis)"
	@echo "  make down              - Stop all running containers"
	@echo "  make test              - Run all automated Go test suites"
	@echo "  make test-unit         - Run unit tests in tests/unit/"
	@echo "  make test-security     - Run tenant isolation and domain security tests"
	@echo "  make test-deployment   - Run Docker and SSL deployment verification tests"
	@echo "  make test-performance  - Execute k6 load tests"
	@echo "  make test-resource     - Run CPU and memory benchmark tests"
	@echo "  make lint              - Format check and static analysis"
	@echo "  make scan-vuln         - Run Go vulnerability scanner (govulncheck)"
	@echo "  make scan-zap          - Run OWASP ZAP baseline security scan"
	@echo "  make backup            - Trigger automated database backup"
	@echo "  make restore           - Restore database from specified backup file"
	@echo "  make clean             - Clean build artifacts and temporary files"

audit:
	@echo "===> Running Codebase & Dependency Audit..."
	go vet ./...
	git status --short

build:
	@echo "===> Building Go API Server..."
	CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=1.0.0" -o bin/server ./cmd/api/main.go
	@if [ -f "package.json" ]; then \
		echo "===> Building React Frontend..."; \
		npm run build; \
	fi

up:
	@echo "===> Starting Production Containers..."
	docker compose up -d

down:
	@echo "===> Stopping Production Containers..."
	docker compose down

test: test-unit test-security test-deployment
	@echo "===> Running All Core Go Tests..."
	go test -v ./...

test-unit:
	@echo "===> Running Unit Tests..."
	go test -v ./tests/unit/...

test-integration:
	@echo "===> Running Integration & E2E Tests..."
	go test -v ./tests/e2e/...

test-security:
	@echo "===> Running Security & Tenant Isolation Tests..."
	go test -v ./tests/security/...

test-deployment:
	@echo "===> Running Deployment & SSL Tests..."
	go test -v ./tests/deployment/...

test-performance:
	@echo "===> Running k6 Load Tests..."
	docker compose --profile testing run --rm k6 run /scripts/load/load-test.js

test-resource:
	@echo "===> Running Resource Benchmarks..."
	go test -v -bench=. ./tests/resource/cpu/...
	go test -v ./tests/resource/memory/...

lint:
	@echo "===> Running Linting & Formatting Check..."
	go fmt ./...
	go vet ./...

scan-vuln:
	@echo "===> Scanning Dependencies for Known Vulnerabilities..."
	@which govulncheck > /dev/null 2>&1 && govulncheck ./... || echo "govulncheck not installed locally; run 'go install golang.org/x/vuln/cmd/govulncheck@latest'"

scan-zap:
	@echo "===> Running OWASP ZAP Baseline Security Scan..."
	docker run --rm -t --net="host" zaproxy/zap-stable zap-baseline.py -t $(API_URL) -r zap-report.html || true

backup:
	@echo "===> Triggering Automated Database Backup..."
	bash ./scripts/backup.sh

restore:
	@echo "===> Initiating Database Restore..."
	@if [ -z "$(BACKUP_FILE)" ]; then \
		echo "Error: BACKUP_FILE variable must be set. Example: make restore BACKUP_FILE=./backups/backup.sql.gz"; \
		exit 1; \
	fi
	bash ./scripts/restore.sh $(BACKUP_FILE)

clean:
	@echo "===> Cleaning Build Artifacts..."
	rm -rf bin/ dist/
	go clean -cache -testcache
