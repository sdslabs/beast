#!/usr/bin/env bash

set -euo pipefail

if [[ -z "${BEAST_TEST_PG_DSN:-}" ]]; then
  echo "set BEAST_TEST_PG_DSN to an isolated PostgreSQL database" >&2
  echo "the test creates and drops temporary schemas" >&2
  exit 2
fi

repo_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
cd "${repo_dir}"

go test -race ./core/database \
  -run 'TestConcurrentCorrectSubmissionsAwardOnce|TestConcurrentWrongSubmissionsRespectMaxAttempts|TestDynamicFlagClaimFirstClaimWins|TestDynamicScoreDirtyCoalescesConcurrentMarks' \
  -count=1
