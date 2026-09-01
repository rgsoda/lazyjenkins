# lazyjenkins — build & install
#
# Common targets:
#   make build          plain binary, expects `jk` on PATH at runtime
#   make build-bundled  binary with the `jk` CLI embedded (no separate jk needed)
#   make install        build-bundled + install system-wide (uses sudo)
#   make install-user   build-bundled + install to ~/.local/bin (no sudo)
#   make uninstall       remove the installed binary
#   make run            build and launch the TUI
#   make clean          remove build artifacts
#
# Override the install location:
#   make install PREFIX=/opt/lazyjenkins
#   make install DESTDIR=/tmp/pkg           (staged / packaging installs)

GO      ?= go
BIN     := lazyjenkins
PREFIX  ?= /usr/local
DESTDIR ?=
BINDIR  := $(DESTDIR)$(PREFIX)/bin

USER_BINDIR ?= $(HOME)/.local/bin

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build a plain binary (requires `jk` on PATH at runtime)
	$(GO) build -trimpath -ldflags '$(LDFLAGS)' -o $(BIN) .

.PHONY: fetch-jk
fetch-jk: ## Download the pinned `jk` binaries for embedding (needs curl, tar)
	./scripts/fetch-jk.sh

.PHONY: build-bundled
build-bundled: fetch-jk ## Build a binary with `jk` embedded
	$(GO) build -trimpath -tags embedjk -ldflags '$(LDFLAGS)' -o $(BIN) .

.PHONY: install
install: build-bundled ## Install system-wide into $(PREFIX)/bin (uses sudo if needed)
	@mkdir -p "$(BINDIR)" 2>/dev/null || sudo mkdir -p "$(BINDIR)"
	@if [ -w "$(BINDIR)" ]; then \
		install -m 0755 $(BIN) "$(BINDIR)/$(BIN)"; \
	else \
		sudo install -m 0755 $(BIN) "$(BINDIR)/$(BIN)"; \
	fi
	@echo "installed $(BINDIR)/$(BIN)"

.PHONY: install-user
install-user: build-bundled ## Install into ~/.local/bin (no sudo)
	@mkdir -p "$(USER_BINDIR)"
	install -m 0755 $(BIN) "$(USER_BINDIR)/$(BIN)"
	@echo "installed $(USER_BINDIR)/$(BIN)"
	@case ":$$PATH:" in *":$(USER_BINDIR):"*) ;; \
		*) echo "note: $(USER_BINDIR) is not on your PATH — add it to your shell rc";; esac

.PHONY: uninstall
uninstall: ## Remove the installed binary from $(PREFIX)/bin and ~/.local/bin
	rm -f "$(BINDIR)/$(BIN)" "$(USER_BINDIR)/$(BIN)" 2>/dev/null || \
		sudo rm -f "$(BINDIR)/$(BIN)"
	@echo "removed $(BIN)"

.PHONY: run
run: build ## Build and launch the TUI
	./$(BIN)

.PHONY: test
test: ## Run tests
	$(GO) test ./...

.PHONY: vet
vet: ## Run go vet
	$(GO) vet ./...

.PHONY: tidy
tidy: ## Sync go.mod / go.sum
	$(GO) mod tidy

.PHONY: clean
clean: ## Remove build artifacts
	rm -f $(BIN)
	rm -rf dist internal/embedjk/binaries
	$(GO) clean

.PHONY: version
version: ## Print the version that would be built
	@echo $(VERSION)
