GO := go
AIR := $(shell $(GO) env GOPATH)/bin/air
BEAST_BIN := $(if $(BEAST_OUTPUT),$(BEAST_OUTPUT),$(shell $(GO) env GOPATH)/bin/beast)

pkgs  = $(shell $(GO) list ./... | grep -v vendor)

help:
	@echo "BEAST: An automated challenge deployment tool for backdoor"
	@echo ""
	@echo "* build: Build beast and copy binary to PATH set for go build binaries."
	@echo "* dev: Run development environment with hot-reloading enabled"
	@echo "* requirements: Build beast extra artifacts requirements."
	@echo "* check_format: Check for formatting errors using gofmt"
	@echo "* format: format the go files using go_fmt in the project directory."
	@echo "* test: Run tests for beast"
	@echo ""

# Build beast
build:
	@BEAST_OUTPUT="$(BEAST_BIN)" ./scripts/build/build.sh

# Run development environment
dev: 
	@echo "Starting development server for beast..."
	@$(AIR)

cmdref: build
	@rm -rf docs/cmdref
	@"$(BEAST_BIN)" cmdref --reference-directory docs/cmdref

# Check go formatting
check_format:
	@echo "[*] Checking for formatting errors using gofmt"
	@./scripts/build/check_gofmt.sh

# Add more tests later on for this
test: check_format
	@echo "[*] Running unit tests"
	@$(GO) test ./...

test-race:
	@echo "[*] Running race-enabled tests"
	@$(GO) test -race ./...

integration-test: build
	@BEAST_RUN_INTEGRATION=1 ./scripts/test/test_examples.sh

# Format code using gofmt
format:
	@echo "[*] Formatting code"
	@$(GO) fmt $(pkgs)

# Vet code using go vet
govet:
	@echo "[*] Vetting code, checking for mistakes"
	@$(GO) vet $(pkgs)

requirements:
	@echo ">>> Building beast extras..."
	@./scripts/build/extras.sh

docs:
	@rm -rf site/
	@echo ">>> Building Documentation"
	@mkdocs build --strict
	@python3 scripts/tools/swagger-docs.py

installenv:
	@echo 'Setting up environment for beast.'
	@./scripts/installenv.sh

.PHONY: build cmdref format test test-race integration-test check_format docs installenv govet requirements
