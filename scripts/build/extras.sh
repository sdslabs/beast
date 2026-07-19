#!/usr/bin/env bash

set -euo pipefail

repo_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
docker build --pull --tag beast-static:latest "${repo_dir}/extras/static-content"

echo "Built beast-static:latest. Deploy it through the authenticated Beast management API."
