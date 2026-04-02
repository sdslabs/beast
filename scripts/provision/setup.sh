#!/bin/bash

echo -e "Setting up sample environment for beast..."

# Creating required directories
mkdir -p "$HOME/.beast" "$HOME/.beast/assets/logo" "$HOME/.beast/remote" "$HOMER/.beast/uploads" "$HOME/.beast/secrets" "$HOME/.beast/scripts" "$HOME/.beast/staging"

# Creating random authorized_keys and secret.key files
echo -e "auth_keys" > $HOME/.beast/authorized_keys
echo -e "auth_keys" > $HOME/.beast/secret.key

# Set beast folder location in vagant box
BEAST_FOLDER=$HOME/beast

BEAST_GLOBAL_CONFIG=$HOME/.beast/config.toml
EXAMPLE_CONFIG_FILE=$BEAST_FOLDER/_examples/example.config.toml

if [ -f "$BEAST_GLOBAL_CONFIG" ]; then
    echo -e "Found $BEAST_GLOBAL_CONFIG"
else
    if [ -f "$EXAMPLE_CONFIG_FILE" ]; then
        echo -e "Copying example config file"
        cp $BEAST_FOLDER/_examples/example.config.toml $BEAST_GLOBAL_CONFIG
    else
        echo -e '\e[93mCould not find example.config.toml'
        echo -e 'Downloading example.config.toml'
        wget https://raw.githubusercontent.com/sdslabs/beast/master/_examples/example.config.toml
        cp ./example.config.toml $BEAST_GLOBAL_CONFIG
        exit
    fi
    sed -i "s/vsts/$USER/g" $BEAST_GLOBAL_CONFIG
fi

echo -e "Created .beast folder..."

echo -e "Building beast..."

export GO111MODULES=on

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

