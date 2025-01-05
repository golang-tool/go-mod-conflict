
.PHONY: help
help:
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: install-precommit
install-precommit: ## Install pre-commit hooks
	@pre-commit install
	@pre-commit gc

.PHONY: validate
validate: ## Validate files with pre-commit hooks
	@pre-commit run --all-files

.PHONY: test-coverage
test-coverage: build ## Test coverage with open report in default browser
	@go test -cover -coverprofile=cover.out -v ./...
	@go tool cover -html=cover.out

.PHONY: go-dependency
dependency: ## Dependency maintanance
	go get -u ./...
	go mod tidy

.PHONY: lint
lint: ## Linters
	@gofmt -w .
	@golangci-lint run

.PHONY: build-linux
build-linux: lint ## Build golang app for linux. Required for cases when on Mac building a cli for Docker container.
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags='$(LDFLAGS)' -o=bin/ .

.PHONY: build-macos
build-macos: lint ## Build golang app for any platform
	@CGO_ENABLED=0 GOOS=darwin GOARCH=$(uname -m) go build -ldflags="$(LDFLAGS)" -o=bin/ .

.PHONY: run
run: ## Run the application
	echo $(GO_MOD_LOCATION)
	@go run main.go --go-mod-location $(GO_MOD_LOCATION)
