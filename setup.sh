#!/bin/bash

echo -e "Setting up sample environment for beast..."

# Creating required directories
mkdir -p "/home/$USER/.beast" "/home/$USER/.beast/assets/logo" "/home/$USER/.beast/remote" "/home/$USER/.beast/uploads" "/home/$USER/.beast/secrets" "/home/$USER/.beast/scripts" "/home/$USER/.beast/staging"

# Creating random authorized_keys and secret.key files
echo -e "auth_keys" >/home/$USER/.beast/authorized_keys
echo -e "auth_keys" >/home/$USER/.beast/secret.key

BEAST_GLOBAL_CONFIG=~/.beast/config.toml
EXAMPLE_CONFIG_FILE=./_examples/example.config.toml

if [ -f "$BEAST_GLOBAL_CONFIG" ]; then
    echo -e "Found $BEAST_GLOBAL_CONFIG"
else
    if [ -f "$EXAMPLE_CONFIG_FILE" ]; then
        echo -e "Copying example config file"
        mv ./_examples/example.config.toml $BEAST_GLOBAL_CONFIG
    else
        echo -e '\e[93mCould not find example.config.toml'
        echo -e 'Downloading example.config.toml'
        wget https://raw.githubusercontent.com/sdslabs/beast/master/_examples/example.config.toml
        mv ./example.config.toml $BEAST_GLOBAL_CONFIG
        exit
    fi
    sed -i "s/vsts/$USER/g" $BEAST_GLOBAL_CONFIG
fi

echo -e "Created .beast folder..."

echo -e "Building beast..."

export GO111MODULES=on

echo -e 'validating $GOPATH...'
if [ -z "$GOPATH" ]; then
    echo -e '\e[31m$GOPATH is not set...'
    echo -e '\e[31mAborting...'
    exit
fi

echo -e 'checking if docker is running...'
# Checking if docker deamon is running or not by checking its PID
DOCKER_PID_FILE=/var/run/docker.pid
if [ -f "$DOCKER_PID_FILE" ]; then
    echo -e "Docker is running."
else
    DOCKER_PID_FILE=/var/run/docker-desktop-proxy.pid # for Docker desktop users		
    if [ -f "$DOCKER_PID_FILE" ]; then
    	echo -e "Docker is running."   
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
