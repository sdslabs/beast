#!/bin/bash

set -euxo pipefail

GO_VERSION='1.13.1'

function update() {
    sudo apt-get update
}

function upgrade() {
    sudo apt-get -y upgrade
}

echo 'Installing dependecies for beast.'

function install_docker() {
    if ! [ -x "$(command -v docker)" ]; then
        echo 'Info: docker is not installed, Installing...'
        
        curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
        sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"

        update

        sudo apt-cache policy docker-ce

        sudo apt-get install -y docker-ce

        echo 'Docker installed.'
    fi

    sudo usermod -aG docker ${USER}
}

function install_go() {
    if ! [ -x "$(command -v go)" ]; then
        echo 'Info: golang is not installed, Installing...'
        
        wget -O "/tmp/go${GO_VERSION}.linux-amd64.tar.gz" "https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz"
        sudo tar -xvf "/tmp/go${GO_VERSION}.linux-amd64.tar.gz" -C /usr/local/

        rm -rf "/tmp/go${GO_VERSION}.linux-amd64.tar.gz"

        sudo echo 'export GOROOT=/usr/local/go' >> /etc/profile

        echo 'Golang installed.'
    fi

    go version
}

update
upgrade

install_docker
install_go

sudo apt-get install -y libsqlite3-dev build-essential

exit 0
