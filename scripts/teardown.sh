#!/usr/bin/env bash

set -euo pipefail

purge_data=false
case "${1:-}" in
  "") ;;
  --purge-data) purge_data=true ;;
  *) echo "usage: $0 [--purge-data]" >&2; exit 2 ;;
esac

beast_dir=${BEAST_HOME:-"${HOME}/.beast"}
lock_file="${beast_dir}/controller.lock"

if [[ -f "${lock_file}" ]]; then
  if ! command -v flock >/dev/null 2>&1; then
    echo "flock is required to verify the Beast controller lock" >&2
    exit 1
  fi
  exec 9<>"${lock_file}"
  if flock --nonblock 9; then
    flock --unlock 9
  else
    read -r controller_pid <&9 || true
    if [[ ! "${controller_pid:-}" =~ ^[0-9]+$ ]] || ! kill -0 "${controller_pid}" 2>/dev/null; then
      echo "controller lock is held but its PID is invalid; data was not removed" >&2
      exit 1
    fi
    executable=$(readlink -f "/proc/${controller_pid}/exe" 2>/dev/null || true)
    if [[ $(basename -- "${executable}") != beast ]]; then
      echo "refusing to signal PID ${controller_pid}: it is not a Beast process" >&2
      exit 1
    fi
    if [[ $(stat -c '%u' "/proc/${controller_pid}" 2>/dev/null || true) != "$(id -u)" ]]; then
      echo "refusing to signal PID ${controller_pid}: it belongs to another user" >&2
      exit 1
    fi
    kill -TERM "${controller_pid}"
    for _ in {1..60}; do
      if ! kill -0 "${controller_pid}" 2>/dev/null; then
        break
      fi
      sleep 1
    done
    if kill -0 "${controller_pid}" 2>/dev/null; then
      echo "Beast did not stop within 60 seconds; data was not removed" >&2
      exit 1
    fi
  fi
fi

if [[ "${purge_data}" == true ]]; then
  if [[ -z "${beast_dir}" || "${beast_dir}" == / || "${beast_dir}" == "${HOME}" ]]; then
    echo "refusing unsafe Beast data path: ${beast_dir}" >&2
    exit 1
  fi
  rm -rf -- "${beast_dir}"
  echo "Removed Beast local data at ${beast_dir}. External PostgreSQL and Redis data were not removed."
else
  echo "Beast is stopped. Use --purge-data to remove ${beast_dir}."
fi
