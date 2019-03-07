#!/bin/bash

set -euxo pipefail

# Run beast agent
beast_agent & disown

# Run mysql entrypoint
docker-entrypoint.sh "$@"
