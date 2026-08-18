.PHONY: all build build-go install clean web-install web-build web-dev fmt vet test test-cover compile serve watch tag release-check

BINARY=alaws
INSTALL_DIR=$(HOME)/go/bin
WEB_DIR=web

all: build test

# Full build: rebuilds the embedded web UI, then the alaws binary.
# internal/server embeds web/dist via go:embed (web/embed.go), so the UI
# must exist before `go build` will succeed.
build: web-build
	go build -o $(BINARY) ./cmd/alaws

# Go-only rebuild, skipping the web UI. Assumes web/dist already exists
# from a previous `make build`/`make web-build`.
build-go:
	go build -o $(BINARY) ./cmd/alaws

install: build
	cp $(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "Installed to $(INSTALL_DIR)/$(BINARY)"

clean:
	rm -f $(BINARY)
	rm -rf $(WEB_DIR)/dist
	rm -f coverage.out

# --- Web UI ---

web-install:
	cd $(WEB_DIR) && npm install

web-build: web-install
	cd $(WEB_DIR) && npm run build

web-dev:
	cd $(WEB_DIR) && npm run dev

# --- Go checks ---

fmt:
	gofmt -l .

vet:
	go vet ./...

test:
	go test ./... -count=1

test-cover:
	go test ./... -coverprofile=coverage.out -count=1
	go tool cover -func=coverage.out

# --- Try it against the bundled fixture lawbook (fixtures/basic) ---

compile: build
	./$(BINARY) compile fixtures/basic

serve: build
	./$(BINARY) serve fixtures/basic

watch: build
	./$(BINARY) watch fixtures/basic

# --- Publishing ---

# Tag a new version and push to GitHub. Usage: make tag V=v0.1.0
tag:
	@test -n "$(V)" || (echo "Usage: make tag V=v0.1.0" && exit 1)
	git tag -a $(V) -m "Release $(V)"
	git push origin $(V)
	@echo "Tagged $(V). pkg.go.dev will index it within minutes."
	@echo "Visit: https://pkg.go.dev/github.com/shrsv/AgentLaws@$(V)"

# Run pre-release checks: all tests and vet.
release-check: test
	go vet ./...
	@echo "All checks passed. Ready to tag with: make tag V=v0.1.0"
