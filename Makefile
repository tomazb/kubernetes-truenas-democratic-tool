.PHONY: help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: all
all: build-all test-all ## Build and test everything

# Go targets
.PHONY: go-deps
go-deps: ## Install Go dependencies
	cd go && go mod download

.PHONY: go-build
go-build: go-deps ## Build all Go binaries
	cd go && go build -ldflags "-X main.version=$(shell cat VERSION) -X main.gitCommit=$(shell git rev-parse --short HEAD) -X main.buildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)" -o ../bin/monitor ./cmd/monitor
	cd go && go build -ldflags "-X main.version=$(shell cat VERSION) -X main.gitCommit=$(shell git rev-parse --short HEAD) -X main.buildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)" -o ../bin/api-server ./cmd/api-server

.PHONY: go-test
go-test: ## Run Go tests
	cd go && go test ./... -v -cover -coverprofile=coverage.out

.PHONY: go-test-coverage
go-test-coverage: go-test ## Run Go tests with coverage report
	cd go && go tool cover -html=coverage.out -o coverage.html

.PHONY: go-lint
go-lint: ## Run Go linters
	cd go && golangci-lint run ./...

.PHONY: go-security
go-security: ## Run Go security checks
	cd go && gosec -fmt=json -out=security-report.json ./...

# Python targets
.PHONY: python-deps
python-deps: ## Install Python dependencies
	pip install -r python/requirements.txt
	pip install -r python/requirements-dev.txt

.PHONY: python-build
python-build: ## Build Python package
	cd python && python -m build

.PHONY: python-test
python-test: ## Run Python tests
	cd python && pytest tests/ -v --cov=. --cov-report=html --cov-report=term-missing

.PHONY: python-lint
python-lint: ## Run Python linters
	cd python && black . --check
	cd python && flake8 .
	cd python && mypy .

.PHONY: python-security
python-security: ## Run Python security checks
	cd python && bandit -r . -f json -o security-report.json
	cd python && safety check

# Combined targets
.PHONY: build-all
build-all: go-build python-build ## Build all components

.PHONY: test-all
test-all: go-test python-test ## Run all tests

.PHONY: test-unit
test-unit: ## Run unit tests only
	cd go && go test ./... -v -short
	cd python && pytest tests/unit/ -v

.PHONY: test-integration
test-integration: ## Run integration tests
	cd go && go test ./... -v -run Integration
	cd python && pytest tests/integration/ -v

.PHONY: test-e2e
test-e2e: ## Run end-to-end tests
	pytest tests/e2e/ -v

.PHONY: test-security
test-security: ## Run security tests
	pytest tests/security/ -v

.PHONY: test-idempotency
test-idempotency: ## Run idempotency tests
	pytest tests/idempotency/ -v --run-twice

.PHONY: test-watch
test-watch: ## Run tests in watch mode
	cd python && ptw tests/unit/ -- -v

.PHONY: lint-all
lint-all: go-lint python-lint ## Run all linters

.PHONY: security-scan
security-scan: go-security python-security ## Run all security scans

.PHONY: fmt
fmt: ## Format all code
	cd go && go fmt ./...
	cd python && black .

# Container targets (using Podman)
.PHONY: container-build-monitor
container-build-monitor: ## Build monitor service container
	podman build -f deploy/container/Containerfile.monitor -t truenas-monitor:latest .

.PHONY: container-build-api
container-build-api: ## Build API server container
	podman build -f deploy/container/Containerfile.api -t truenas-api:latest .

.PHONY: container-build-cli
container-build-cli: ## Build CLI tool container
	podman build -f deploy/container/Containerfile.cli -t truenas-cli:latest .

.PHONY: container-build-all
container-build-all: container-build-monitor container-build-api container-build-cli ## Build all containers

# Kubernetes/OpenShift targets
.PHONY: k8s-deploy
k8s-deploy: ## Deploy to Kubernetes
	kubectl apply -f deploy/kubernetes/

.PHONY: k8s-delete
k8s-delete: ## Delete from Kubernetes
	kubectl delete -f deploy/kubernetes/

.PHONY: helm-install
helm-install: ## Install using Helm
	helm install truenas-monitor deploy/helm/truenas-monitor \
		--namespace storage-monitoring \
		--create-namespace

.PHONY: helm-upgrade
helm-upgrade: ## Upgrade Helm deployment
	helm upgrade truenas-monitor deploy/helm/truenas-monitor \
		--namespace storage-monitoring

.PHONY: helm-uninstall
helm-uninstall: ## Uninstall Helm deployment
	helm uninstall truenas-monitor --namespace storage-monitoring

# Development targets
.PHONY: dev-setup
dev-setup: ## Set up development environment
	python -m venv venv
	./venv/bin/pip install -r python/requirements-dev.txt
	cd go && go mod download
	@echo "Development environment ready. Activate Python venv with: source venv/bin/activate"

.PHONY: clean
clean: ## Clean build artifacts
	rm -rf bin/
	rm -rf python/dist/
	rm -rf python/build/
	rm -rf python/*.egg-info
	find . -type d -name __pycache__ -exec rm -rf {} +
	find . -type f -name "*.pyc" -delete
	find . -type f -name "*.pyo" -delete
	find . -type f -name "*.cover" -delete
	find . -type f -name ".coverage" -delete
	rm -rf htmlcov/
	rm -rf .pytest_cache/
	rm -rf .mypy_cache/
	cd go && go clean -cache

# Documentation targets
.PHONY: docs
docs: ## Generate documentation
	cd python && sphinx-build -b html docs/ docs/_build/html
	cd go && godoc -http=:6060

.PHONY: docs-serve
docs-serve: ## Serve documentation locally
	cd python && sphinx-autobuild docs/ docs/_build/html

# Release targets
.PHONY: version
version: ## Show current version
	@cat VERSION

.PHONY: release
release: ## Create a new release
	@echo "Creating release..."
	@read -p "Version (current: $$(cat VERSION)): " version; \
	echo $$version > VERSION; \
	git add VERSION; \
	git commit -m "Release v$$version"; \
	git tag -a v$$version -m "Release v$$version"; \
	echo "Release v$$version created. Push with: git push origin main --tags"