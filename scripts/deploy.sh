#!/bin/bash

export GOROOT=/usr/local/go
export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin/
export GOPATH=$HOME/go

go version

cd beast
make build
make tools

sudo cp "beast.service.example" "/etc/systemd/system/beast.service"
sudo systemctl start beast
