#!/bin/bash

echo -e "Setting up sample environment for beast..."

# Creating required directories
mkdir -p "/home/$USER/.beast" "/home/$USER/.beast/assets/logo" "/home/$USER/.beast/remote" "/home/$USER/.beast/uploads" "/home/$USER/.beast/secrets" "/home/$USER/.beast/scripts" "/home/$USER/.beast/staging"

# Creating random authorized_keys and secret.key files
echo -e "auth_keys" >/home/$USER/.beast/authorized_keys
echo -e "auth_keys" >/home/$USER/.beast/secret.key

mv ./_examples/example.config.toml ~/.beast/config.toml

sed -i "s/vsts/$USER/g" ~/.beast/config.toml

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
    echo -e '\e[31mDocker daemon is not running'
    echo -e '\e[31mAborting...'
    echo -e "\e[31mPlease start docker daemon and restart again"
    exit
fi

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
