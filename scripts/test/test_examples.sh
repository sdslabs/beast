#!/usr/bin/env bash
set -euo pipefail

root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
beast_bin=${BEAST:-beast}

for config_file in "$root_dir"/_examples/*/beast.toml; do
  challenge_dir=$(dirname "$config_file")
  printf 'Verifying %s\n' "${challenge_dir##*/}"
  "$beast_bin" verify "$challenge_dir"
done
