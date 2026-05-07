MODULE      := github.com/Logiphys/lgp-mcp-servers
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE  := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS     := -s -w -X main.version=$(VERSION) -X main.buildDate=$(BUILD_DATE)
SERVERS     := autotask-mcp itglue-mcp datto-rmm-mcp rocketcyber-mcp datto-uc-mcp datto-network-mcp myitprocess-mcp datto-backup-mcp datto-edr-mcp logiphys-ci-mcp
PLATFORMS   := darwin/arm64 darwin/amd64 windows/amd64

# logiphys-ci-mcp embeds the marketplace skill SHA, tag, and load timestamp via
# linker flags so the version tool can return audit metadata at runtime.
LOGIPHYSCI_PKG := $(MODULE)/pkg/logiphysci
SKILL_SHA      := $(shell git -C external/logiphys-marketplace rev-parse --short HEAD 2>/dev/null || echo "unknown")
SKILL_TAG      := $(shell git -C external/logiphys-marketplace describe --tags --abbrev=0 2>/dev/null || echo "none")
LOADED_AT      := $(BUILD_DATE)
LDFLAGS_LOGIPHYSCI := $(LDFLAGS) \
	-X '$(LOGIPHYSCI_PKG).SkillSHA=$(SKILL_SHA)' \
	-X '$(LOGIPHYSCI_PKG).SkillTag=$(SKILL_TAG)' \
	-X '$(LOGIPHYSCI_PKG).LoadedAt=$(LOADED_AT)'

.PHONY: build
build:
	@for s in $(SERVERS); do \
		echo "Building $$s..."; \
		if [ "$$s" = "logiphys-ci-mcp" ]; then \
			go build -ldflags "$(LDFLAGS_LOGIPHYSCI)" -o dist/$$s ./cmd/$$s; \
		else \
			go build -ldflags "$(LDFLAGS)" -o dist/$$s ./cmd/$$s; \
		fi; \
	done

.PHONY: build-all
build-all:
	@for s in $(SERVERS); do \
		for p in $(PLATFORMS); do \
			os=$${p%%/*}; arch=$${p##*/}; \
			ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
			echo "Building $$s ($$os/$$arch)..."; \
			if [ "$$s" = "logiphys-ci-mcp" ]; then \
				GOOS=$$os GOARCH=$$arch \
				go build -ldflags "$(LDFLAGS_LOGIPHYSCI)" \
					-o dist/$$s-$$os-$$arch$$ext \
					./cmd/$$s; \
			else \
				GOOS=$$os GOARCH=$$arch \
				go build -ldflags "$(LDFLAGS)" \
					-o dist/$$s-$$os-$$arch$$ext \
					./cmd/$$s; \
			fi; \
		done \
	done

.PHONY: build-logiphys-ci-mcp
build-logiphys-ci-mcp:
	go build -ldflags "$(LDFLAGS_LOGIPHYSCI)" -o dist/logiphys-ci-mcp ./cmd/logiphys-ci-mcp

.PHONY: build-%
build-%:
	go build -ldflags "$(LDFLAGS)" -o dist/$* ./cmd/$*

.PHONY: test
test:
	go test ./... -v -race -count=1

.PHONY: test-cover
test-cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: check-docs
check-docs:
	bash scripts/check-docs.sh

.PHONY: clean
clean:
	rm -rf dist/ coverage.out coverage.html

.PHONY: help
help:
	@echo "LGP MCP Servers"
	@echo ""
	@echo "  make build            Build all servers for current platform"
	@echo "  make build-all        Cross-compile for macOS + Windows"
	@echo "  make build-<name>     Build single server (e.g. make build-autotask-mcp)"
	@echo "  make test             Run all tests with race detection"
	@echo "  make test-cover       Generate HTML coverage report"
	@echo "  make lint             Run golangci-lint"
	@echo "  make check-docs       Verify docs consistency with code"
	@echo "  make clean            Remove build artifacts"
	@echo "  make help             Show this message"
