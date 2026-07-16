#!/usr/bin/env bash

set -euo pipefail

repo_dir=${BEAST_REPOSITORY:-"${HOME}/beast"}
exec "${repo_dir}/setup.sh"
