#!/bin/bash

set -euxo pipefail

CWD=$PWD

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
	docker run -d -p 80:80 \
		-v ~/.beast/staging:/beast \
		-v ~/.beast/.static.beast.htpasswd:/.static.beast.htpasswd \
		beast-static
fi

cd $CWD
echo -e "\n\nBuilding beast extras sidecar images: MYSQL\n"
cd "${CWD}/extras/sidecars/mysql"

if docker images | grep -q 'beast-mysql'; then
	echo "Image for beast-mysql container already exists."
else
	docker build . --tag beast-mysql:latest
fi

if docker network ls | grep -q 'beast-mysql'; then
	echo "Network for beast-mysql sidecar already exists."
else
	docker nework create beast-mysql
fi

if docker ps -a | grep -q 'mysql'; then
	echo "Container for mysql sidecar with name mysql already exists."
else
	docker run -d -p 127.0.0.1:9500:9500 \
		--name mysql --network beast-mysql \
		--env MYSQL_ROOT_PASSWORD=$(openssl rand -hex 20) \
		beast-mysql
fi
