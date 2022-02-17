GO := go
AIR := ${GOPATH}/bin/air	

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
	@echo "* tools: Set up required tools for beast which includes - docker-enter, importenv"
	@echo ""

# Build beast
build: tools
	@./scripts/build/build.sh

# Run development environment
dev: 
	@echo "Starting development server for beast..."
	@$(AIR)

cmdref: build
	@${GOPATH}/bin/beast cmdref

# Check go formatting
check_format:
	@echo "[*] Checking for formatting errors using gofmt"
	@./scripts/build/check_gofmt.sh

# Add more tests later on for this
test: check_format
	@echo "[*] Running tests for example challenges"
	@./scripts/test/test_examples.sh

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
	@rm -rf site/
	@echo ">>> Building Documentation"
	@mkdocs build
	@python scripts/tools/swagger-docs.py

installenv:
	@echo 'Setting up environment for beast.'
	@./scripts/installenv.sh

.PHONY: build format test check_format tools docs installenv
