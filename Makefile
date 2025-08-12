APP_NAME = redis-export
APP_VSN ?= $(shell git rev-parse --short HEAD)

.PHONY: help
help: #: Show this help message
	@echo "$(APP_NAME):$(APP_VSN)"
	@awk '/^[A-Za-z_ -]*:.*#:/ {printf("%c[1;32m%-15s%c[0m", 27, $$1, 27); for(i=3; i<=NF; i++) { printf("%s ", $$i); } printf("\n"); }' Makefile* | sort

CGO_ENABLED ?= 0
GO = CGO_ENABLED=$(CGO_ENABLED) go
GO_BUILD_FLAGS = -ldflags "-X main.Version=${APP_VSN}"

### Dev

.PHONY: run
run: #: Run the application
	$(GO) run $(GO_BUILD_FLAGS) .

.PHONY: code-check
code-check: #: Run the linter
	golangci-lint run

### Build

.PHONY: build
build: #: Build the app locally
build: clean 
	$(GO) build $(GO_BUILD_FLAGS) -o ./$(APP_NAME)

.PHONY: build-all
build-all: #: Build for all platforms
build-all: clean
	GOOS=linux GOARCH=amd64 $(GO) build $(GO_BUILD_FLAGS) -o ./$(APP_NAME)-linux
	GOOS=windows GOARCH=amd64 $(GO) build $(GO_BUILD_FLAGS) -o ./$(APP_NAME).exe
	GOOS=darwin GOARCH=arm64 $(GO) build $(GO_BUILD_FLAGS) -o ./$(APP_NAME)-mac

.PHONY: clean
clean: #: Clean up build artifacts
clean:
	$(RM) ./$(APP_NAME) ./$(APP_NAME)-linux ./$(APP_NAME).exe ./$(APP_NAME)-mac

### Test

.PHONY: test
test: #: Run Go unit tests
test:
	$(GO) test -v ./...

.PHONY: test-race
test-race: #: Run Go tests with race detection
test-race:
	$(GO) test -v -race ./...

.PHONY: test-cover
test-cover: #: Run tests with coverage
test-cover:
	$(GO) test -v -cover ./... -coverprofile=coverage.out

.PHONY: cover
cover: #: Open coverage report in browser
cover: test-cover
	go tool cover -html=coverage.out

### Release

.PHONY: tag-release
tag-release: #: Create a release tag (requires VERSION=X.Y.Z, optional RELEASE_TYPE=patch|minor|major|prerelease)
tag-release:
ifndef VERSION
	$(error VERSION is required. Use: make tag-release VERSION=0.1.0)
endif
	gh workflow run tag-release.yml -f version=$(VERSION) -f release_type=$(or $(RELEASE_TYPE),patch)

.PHONY: check-release
check-release: #: Check latest release status
check-release:
	gh release list --limit 5