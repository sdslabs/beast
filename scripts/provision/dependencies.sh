#!/usr/bin/env bash

set -euo pipefail

readonly go_version="1.23.12"
readonly go_sha256="d3847fef834e9db11bf64e3fb34db9c04db14e068eeb064f49af747010454f90"

export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get install --yes --no-install-recommends ca-certificates curl git make

archive=$(mktemp)
trap 'rm -f "${archive}"' EXIT
curl --fail --location --proto '=https' --tlsv1.2 \
  "https://dl.google.com/go/go${go_version}.linux-amd64.tar.gz" \
  --output "${archive}"
printf '%s  %s\n' "${go_sha256}" "${archive}" | sha256sum --check --status

rm -rf /usr/local/go
tar --extract --gzip --file "${archive}" --directory /usr/local
ln -sfn /usr/local/go/bin/go /usr/local/bin/go
