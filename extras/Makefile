GO := go
CWD := ${PWD}

build-agents:
	@echo ">>> Building agents..."
	@echo "Building  mysql agent server..."
	@cd agents/mysql && $(GO) build -o ${GOPATH}/mysql server.go
	@cd $(CWD)
