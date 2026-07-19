#!/usr/bin/env bash

# Build Beast

set -euo pipefail

GO_FLAGS=${GO_FLAGS:-"-tags netgo"}
GO_CMD=${GO_CMD:-"build"}
BUILD_USER=${BUILD_USER:-"${USER:-unknown}@${HOSTNAME:-unknown}"}
BUILD_DATE=${BUILD_DATE:-$(date -u +%Y%m%d-%H:%M:%S)}
VERBOSE=${VERBOSE:-}
OUTPUT=${BEAST_OUTPUT:-"$(go env GOPATH)/bin/beast"}

repo_path="github.com/sdslabs/beastv4"
main_package="github.com/sdslabs/beastv4/cmd/beast"
# Get branch revision and  version
version="0.2"
revision=$(git rev-parse --short HEAD 2> /dev/null || echo 'unknown')
branch=$(git rev-parse --abbrev-ref HEAD 2> /dev/null || echo 'unknown')

# Extract the go version
go_version=$(go version | sed -e 's/^[^0-9.]*\([0-9.]*\).*/\1/')


ldflags="
  -X ${repo_path}/version.Version=${version}
  -X ${repo_path}/version.Revision=${revision}
  -X ${repo_path}/version.Branch=${branch}
  -X ${repo_path}/version.BuildUser=${BUILD_USER}
  -X ${repo_path}/version.BuildDate=${BUILD_DATE}
  -X ${repo_path}/version.GoVersion=${go_version}"

echo ">>> Building Beast..."

if [ -n "$VERBOSE" ]; then
  echo "Building with -ldflags $ldflags"
fi

mkdir -p "$(dirname "${OUTPUT}")"
# GO_FLAGS is intentionally word-split to preserve the existing override interface.
# shellcheck disable=SC2086
go "${GO_CMD}" -o "${OUTPUT}" ${GO_FLAGS} -ldflags "${ldflags}" "${main_package}"

echo "[*] Build complete: ${OUTPUT}"
