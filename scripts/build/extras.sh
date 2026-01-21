#!/bin/bash

set -euxo pipefail

CWD=$PWD

BEAST_STATIC_PORT=8034

echo -e "\nBuilding static content server for beast...\n"
cd "${CWD}/extras/static-content"

if docker images | grep -q 'beast-static'; then
	echo "Image for static-content already exists."
else
	docker build . --tag beast-static:latest
fi

if docker ps -a | grep -q 'beast-static'; then
	echo "Container for static-content already exists."
else
	docker run -d -p $BEAST_STATIC_PORT:80 \
		-v ~/.beast/staging:/beast \
		-v ~/.beast/.static.beast.htpasswd:/.static.beast.htpasswd \
		beast-static
fi

echo "Extras build script complete."

