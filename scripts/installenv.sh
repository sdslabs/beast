#!/usr/bin/env bash

set -euo pipefail

install_dev_tools=false
case "${1:-}" in
  "") ;;
  --dev) install_dev_tools=true ;;
  *) echo "usage: $0 [--dev]" >&2; exit 2 ;;
esac

required_commands=(go docker git make)
missing=()
for command in "${required_commands[@]}"; do
  if ! command -v "${command}" >/dev/null 2>&1; then
    missing+=("${command}")
  fi
done
if ((${#missing[@]} > 0)); then
  echo "missing required commands: ${missing[*]}" >&2
  echo "install them with your operating system's trusted package manager" >&2
  exit 1
fi

go version
docker version --format '{{.Client.Version}}' >/dev/null
if ! docker info >/dev/null 2>&1; then
  echo "Docker is installed but the daemon is unavailable to the current user" >&2
  exit 1
fi

if [[ "${install_dev_tools}" == true ]]; then
  echo "installing pinned Air development tool"
  go install github.com/air-verse/air@v1.61.7
fi

echo "Beast prerequisites are available"
