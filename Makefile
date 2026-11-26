GO := go
GO_BIN = $(or $(shell $(GO) env GOBIN),$(firstword $(subst :, ,$(shell $(GO) env GOPATH)))/bin)
AIR ?= $(GO_BIN)/air
SWAG ?= $(GO_BIN)/swag

pkgs  = $(shell $(GO) list ./... | grep -v vendor)

help:
	@echo "BEAST: An automated challenge deployment tool for backdoor"
	@echo ""
	@echo "* build: Build beast and copy binary to PATH set for go build binaries."
	@echo "* dev: Run development environment with hot-reloading enabled"
	@echo "* requirements: Build beast extra artifacts requirements."
	@echo "* check_format: Check for formatting errors using gofmt"
	@echo "* format: format the go files using go_fmt in the project directory."
	@echo "* test: Run the Go test suite"
	@echo "* test-examples: Validate all bundled challenge configurations"
	@echo "* cmdref: Regenerate the CLI reference"
	@echo "* swagger: Regenerate the API reference"
	@echo "* docs: Build the documentation site"
	@echo "* tools: Set up required tools for beast which includes - docker-enter, importenv"
	@echo ""

# Build beast
build: tools
	@./scripts/build/build.sh

# Run development environment
dev: 
	@echo "Starting development server for beast..."
	@"$(AIR)"

cmdref: build
	@"$(GO_BIN)/beast" cmdref
	@sed -i '$${/^$$/d;}' docs/cmdref/beast*.md

swagger:
	@"$(SWAG)" init --dir api,core/database --generalInfo main.go --output api/docs --parseInternal
	@cp api/docs/swagger.json api/docs/swagger.yaml api/docs/swagger/

# Check go formatting
check_format:
	@echo "[*] Checking for formatting errors using gofmt"
	@./scripts/build/check_gofmt.sh

test: check_format
	@$(GO) test ./...

test-examples:
	@BEAST="$(GO_BIN)/beast" ./scripts/test/test_examples.sh

# Format code using gofmt
format:
	@echo "[*] Formatting code"
	@$(GO) fmt $(pkgs)

# Vet code using go vet
govet:
	@echo "[*] Vetting code, checking for mistakes"
	@$(GO) vet $(pkgs)

# Ensure that the required tools are installed for beast to work
tools:
	@if ! test -x "`which nsenter 2>&1;true`"; then \
	  echo 'Error: nsenter is not installed, Install it first' >&2 ; \
	fi

	@if ! test -x "`which docker-enter 2>&1;true`"; then \
	  echo 'Warn: docker-enter is not installed, building....' >&2 ; \
	  sudo cp ./scripts/docker-enter "/usr/bin/" ; \
	  sudo cp ./scripts/docker_enter "/usr/bin/"; \
	  sudo chown root "/usr/bin/docker_enter"; \
	  sudo chmod u+s "/usr/bin/docker_enter"; \
	fi

	@if ! test -x "`which importenv 2>&1;true`"; then \
	  echo 'Warn: importenv is not installed, building....' >&2 ; \
	  sudo gcc -o "/usr/bin/importenv" ./scripts/importenv.c ; \
	fi

requirements:
	@echo ">>> Building beast extras..."
	@./scripts/build/extras.sh

docs:
	@echo ">>> Building Documentation"
	@mkdocs build
	@python3 scripts/tools/swagger-docs.py

installenv:
	@echo 'Setting up environment for beast.'
	@./scripts/installenv.sh

.PHONY: build format test test-examples check_format tools docs installenv cmdref swagger

# ── Docker Compose Targets ────────────────────────────────────────────────────
# Usage: make up NAME=myctf
#
# NAME selects the Compose project for the PostgreSQL and Redis services.
#
# Targets:
#   make up   NAME=<name>  — start PostgreSQL and Redis
#   make down NAME=<name>  — stop and remove services
#   make logs NAME=<name>  — tail PostgreSQL and Redis logs

check-name:
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is required. Usage: make up NAME=myctf"; \
		exit 1; \
	fi
	@if echo "$(NAME)" | grep -q "[[:space:]]"; then \
		echo "Error: NAME must not contain spaces"; \
		exit 1; \
	fi

check-config:
	@if [ ! -f "config.toml" ]; then \
		echo "Error: config.toml not found in current directory."; \
		echo "Place your config.toml here (see _examples/example.config.toml)."; \
		echo "Ensure psql_config.host = \"postgres\" for the compose network."; \
		exit 1; \
	fi

BEAST_DIR = $(HOME)/.$(NAME)

setup-beast-dir: check-name check-config
	@echo "[*] Setting up $(BEAST_DIR)..."
	@mkdir -p $(BEAST_DIR)/assets/logo
	@mkdir -p $(BEAST_DIR)/assets/mailTemplates
	@mkdir -p $(BEAST_DIR)/remote
	@mkdir -p $(BEAST_DIR)/uploads
	@mkdir -p $(BEAST_DIR)/secrets
	@mkdir -p $(BEAST_DIR)/scripts
	@mkdir -p $(BEAST_DIR)/staging
	@mkdir -p $(BEAST_DIR)/cache
	@mkdir -p $(BEAST_DIR)/logs
	@cp config.toml $(BEAST_DIR)/config.toml
	@echo "[*] Beast configuration prepared in $(BEAST_DIR)"

up: check-name
	@echo "[*] Starting PostgreSQL and Redis (project: $(NAME))..."
	@docker compose --project-name "$(NAME)" up -d postgres redis

down: check-name
	@docker compose --project-name "$(NAME)" down

logs: check-name
	@docker compose --project-name "$(NAME)" logs -f postgres redis

.PHONY: check-name check-config setup-beast-dir up down logs
