#!/bin/bash

# Detect if running inside a Docker container
IN_CONTAINER=false
[ -f /.dockerenv ] && IN_CONTAINER=true

echo -e "Setting up sample environment for beast..."

# In container, HOME=/root; locally, use /home/$USER
if [ "$IN_CONTAINER" = true ]; then
    BEAST_HOME="$HOME"
else
    BEAST_HOME="/home/$USER"
fi

# Creating required directories
mkdir -p \
    "${BEAST_HOME}/.beast" \
    "${BEAST_HOME}/.beast/assets/logo" \
    "${BEAST_HOME}/.beast/assets/mailTemplates" \
    "${BEAST_HOME}/.beast/remote" \
    "${BEAST_HOME}/.beast/uploads" \
    "${BEAST_HOME}/.beast/secrets" \
    "${BEAST_HOME}/.beast/scripts" \
    "${BEAST_HOME}/.beast/staging" \
    "${BEAST_HOME}/.beast/cache" \
    "${BEAST_HOME}/.beast/logs"

# Creating placeholder authorized_keys and secret.key files if absent
[ ! -f "${BEAST_HOME}/.beast/beast_authorized_keys" ] && touch "${BEAST_HOME}/.beast/beast_authorized_keys"
[ ! -f "${BEAST_HOME}/.beast/secret.key" ]            && touch "${BEAST_HOME}/.beast/secret.key"

BEAST_GLOBAL_CONFIG="${BEAST_HOME}/.beast/config.toml"
EXAMPLE_CONFIG_FILE="./_examples/example.config.toml"

if [ -f "$BEAST_GLOBAL_CONFIG" ]; then
    echo -e "Found $BEAST_GLOBAL_CONFIG"
else
    if [ "$IN_CONTAINER" = true ]; then
        echo -e "\e[31mconfig.toml not found at ${BEAST_GLOBAL_CONFIG}"
        echo -e "\e[31mMount your config.toml to ${BEAST_GLOBAL_CONFIG} and retry."
        exit 1
    fi

    if [ -f "$EXAMPLE_CONFIG_FILE" ]; then
        echo -e "Copying example config file"
        cp ./_examples/example.config.toml "$BEAST_GLOBAL_CONFIG"
    else
        echo -e '\e[93mCould not find example.config.toml'
        echo -e 'Downloading example.config.toml'
        wget https://raw.githubusercontent.com/sdslabs/beast/master/_examples/example.config.toml
        cp ./example.config.toml "$BEAST_GLOBAL_CONFIG"
        exit
    fi
    sed -i "s/vsts/$USER/g" "$BEAST_GLOBAL_CONFIG"
fi

echo -e "Created .beast folder..."

# ── Container path: binary already built, just start beast ───────────────────
if [ "$IN_CONTAINER" = true ]; then
    echo -e "Checking Docker socket..."
    if [ ! -S /var/run/docker.sock ]; then
        echo -e "\e[31mDocker socket not found at /var/run/docker.sock"
        echo -e "\e[31mMount the host Docker socket and retry."
        exit 1
    fi
    echo -e "Docker socket available. Starting beast..."
    BEAST_FLAGS="${BEAST_FLAGS:--v}"
    echo -e "Running: beast run ${BEAST_FLAGS}"
    exec beast run ${BEAST_FLAGS}
fi

# ── Local path: build then advise the user to run beast ──────────────────────
echo -e "Building beast..."

export GO111MODULES=on

echo -e 'validating $GOPATH...'
if [ -z "$GOPATH" ]; then
    echo -e '\e[31m$GOPATH is not set...'
    echo -e '\e[31mAborting...'
    exit
fi

echo -e 'checking if docker is running...'
# Checking if docker daemon is running or not by checking its PID
DOCKER_PID_FILE=/var/run/docker.pid
if [ -f "$DOCKER_PID_FILE" ]; then
    echo -e "Docker is running."
else
    echo -e '\e[31mDocker daemon is not running'
    echo -e '\e[31mAborting...'
    echo -e "\e[31mPlease start docker daemon and restart again"
    exit
fi

echo -e "Installing air for live reloading"
curl -sSfL https://raw.githubusercontent.com/cosmtrek/air/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

echo -e "Building beast..."
make build
if [ $? -eq 0 ]; then
    echo -e "\e[92mPlease run beast server by following command:-"
    echo -e "******************"
    echo -e "*  \e[5mbeast run -v  \e[25m*"
    echo -e "******************"
else
    echo -e "\e[31mBeast build failed. Please check above errors"
    exit 1
fi
