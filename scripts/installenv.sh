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

        echo 'export GOROOT=/usr/local/go' | tee -a $HOME/.bashrc
        echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin/' | tee -a $HOME/.bashrc
        echo 'export GOPATH=$HOME/go' | tee -a $HOME/.bashrc

        source ~/.bashrc

        echo 'Golang installed.'
    fi

    /usr/local/go/bin/go version
}

function install_air(){
  curl -sSfL https://raw.githubusercontent.com/cosmtrek/air/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
}

update
upgrade

sudo apt-get install -y apt-transport-https libsqlite3-dev build-essential gcc g++

install_docker
install_go
install_air

exit 0
