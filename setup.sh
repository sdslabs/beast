#!/usr/bin/env bash

set -euo pipefail
umask 077

repo_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
beast_dir=${BEAST_HOME:-"${HOME}/.beast"}
config_file="${beast_dir}/config.toml"

"${repo_dir}/scripts/installenv.sh"

mkdir -p \
  "${beast_dir}/assets/logo" \
  "${beast_dir}/assets/mailTemplates" \
  "${beast_dir}/backup/cache" \
  "${beast_dir}/backup/db" \
  "${beast_dir}/cache" \
  "${beast_dir}/remote" \
  "${beast_dir}/secrets" \
  "${beast_dir}/staging" \
  "${beast_dir}/uploads"

created_config=false
if [[ ! -e "${config_file}" ]]; then
  cp "${repo_dir}/_examples/example.config.toml" "${config_file}"
  chmod 0600 "${config_file}"
  jwt_secret=$(od -An -N32 -tx1 /dev/urandom | tr -d ' \n')
  postgres_password=$(od -An -N24 -tx1 /dev/urandom | tr -d ' \n')
  redis_password=$(od -An -N24 -tx1 /dev/urandom | tr -d ' \n')
  sed -i "s/CHANGE_ME_GENERATED_BY_SETUP/${jwt_secret}/" "${config_file}"
  sed -i "0,/password = \"CHANGE_ME\"/s//password = \"${postgres_password}\"/" "${config_file}"
  sed -i "0,/password = \"CHANGE_ME\"/s//password = \"${redis_password}\"/" "${config_file}"
  created_config=true
elif [[ -L "${config_file}" || $(stat -c '%a' "${config_file}") != 600 ]]; then
  echo "existing config must be a regular file with mode 0600: ${config_file}" >&2
  exit 1
fi

BEAST_OUTPUT=${BEAST_OUTPUT:-"$(go env GOPATH)/bin/beast"} make -C "${repo_dir}" build

if [[ "${created_config}" == true ]]; then
  echo "Created ${config_file} with unique JWT and datastore credentials."
  echo "Before starting Beast, configure PostgreSQL, Redis, SMTP, and TLS certificate paths."
else
  echo "Using existing ${config_file}."
fi
echo "Beast binary: ${BEAST_OUTPUT:-$(go env GOPATH)/bin/beast}"
