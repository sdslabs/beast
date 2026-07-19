#!/usr/bin/env bash

set -euo pipefail

if [[ "${BEAST_RUN_INTEGRATION:-}" != "1" ]]; then
  echo "integration tests are destructive; run 'make integration-test' explicitly" >&2
  exit 2
fi

repo_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
beast_bin=${BEAST_BIN:-"$(go env GOPATH)/bin/beast"}
challenge_name="simple"
challenge_dir="${repo_dir}/_examples/${challenge_name}"
challenge_port=10005
deployed=false

if [[ ! -x "${beast_bin}" ]]; then
  echo "Beast executable not found: ${beast_bin}" >&2
  exit 1
fi

cleanup() {
  if [[ "${deployed}" == true ]]; then
    "${beast_bin}" challenge purge "${challenge_name}" --delete-entry || true
  fi
}
trap cleanup EXIT INT TERM

if timeout 1 bash -c "</dev/tcp/127.0.0.1/${challenge_port}" 2>/dev/null; then
  echo "port ${challenge_port} is already in use" >&2
  exit 1
fi

"${beast_bin}" challenge deploy --local-directory "${challenge_dir}"
deployed=true

for _ in {1..30}; do
  if timeout 1 bash -c "</dev/tcp/127.0.0.1/${challenge_port}" 2>/dev/null; then
    echo "challenge ${challenge_name} is reachable on port ${challenge_port}"
    exit 0
  fi
  sleep 1
done

echo "challenge ${challenge_name} did not become reachable" >&2
exit 1
