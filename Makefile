MODULE      := github.com/Logiphys/lgp-mcp
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE  := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS     := -s -w -X main.version=$(VERSION) -X main.buildDate=$(BUILD_DATE)
SERVERS     := autotask-mcp itglue-mcp datto-rmm-mcp rocketcyber-mcp
PLATFORMS   := darwin/arm64 darwin/amd64 windows/amd64

.PHONY: build
build:
	@for s in $(SERVERS); do \
		echo "Building $$s..."; \
		go build -ldflags "$(LDFLAGS)" -o dist/$$s ./cmd/$$s; \
	done

.PHONY: build-all
build-all:
	@for s in $(SERVERS); do \
		for p in $(PLATFORMS); do \
			os=$${p%%/*}; arch=$${p##*/}; \
			ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
			echo "Building $$s ($$os/$$arch)..."; \
			GOOS=$$os GOARCH=$$arch \
			go build -ldflags "$(LDFLAGS)" \
				-o dist/$$s-$$os-$$arch$$ext \
				./cmd/$$s; \
		done \
	done

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

.PHONY: clean
clean:
	rm -rf dist/
