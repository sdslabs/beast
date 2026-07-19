#!/usr/bin/env bash

set -euo pipefail

mapfile -t go_files < <(git ls-files '*.go')
bad_files=$(gofmt -s -l "${go_files[@]}")

if [[ -n "${bad_files}" ]]; then
  echo "The following files are not properly formatted:"
  printf '%s\n' "${bad_files}"
  exit 1
fi
