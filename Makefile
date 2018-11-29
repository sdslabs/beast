GO := go
pkgs  = $(shell $(GO) list ./... | grep -v vendor)

help:
	@echo "BEAST: An automated challenge deployment tool for backdoor"
	@echo ""
	@echo "* build: Build beast and copy binary to PATH set for go build binaries."
	@echo "* check_format: Check for formatting errors using gofmt"
	@echo "* format: format the go files using go_fmt in the project directory."
	@echo "* test: Run tests for beast"
	@echo "* tools: Set up required tools for beast which includes - docker-enter, importenv"
	@echo ""

# Build beast
build: tools
	@./scripts/build/build.sh

# Check go formatting
check_format:
	@echo "[*] Checking for formatting errors using gofmt"
	@./scripts/build/check_gofmt.sh

# Add more tests later on for this
test: check_format

# Format code using gofmt
format:
	@echo "[*] Formatting code"
	@$(GO) fmt $(pkgs)

# Ensure that the required tools are installed for beast to work
tools:
	@if ! test -x "`which nsenter 2>&1;true`"; then \
	  echo 'Error: nsenter is not installed, Install it first' >&2 ; \
	fi

	@if ! test -x "`which docker-enter 2>&1;true`"; then \
	  echo 'Warn: docker-enter is not installed, building....' >&2 ; \
	  cp ./scripts/docker-enter "/usr/bin/" ; \
	fi

	@if ! test -x "`which importenv 2>&1;true`"; then \
	  echo 'Warn: importenv is not installed, building....' >&2 ; \
	  gcc -o "/usr/bin/importenv" ./scripts/importenv.c ; \
	fi


.PHONY: build format test check_format tools
