#!/usr/bin/env bash
set -euo pipefail

HYDRA_NET_NAME="${HYDRA_NET_NAME:-hydra-net}"
HYDRA_BRIDGE_NAME="${HYDRA_BRIDGE_NAME:-hydra0}"
IPTABLES_COMMENT_PREFIX="beast:${HYDRA_NET_NAME}"
REMOVE_NETWORK=false

for arg in "$@"; do
    case "$arg" in
        --remove-network)
            REMOVE_NETWORK=true
            ;;
        *)
            echo "unknown argument: $arg" >&2
            exit 1
            ;;
    esac
done

if [ "$(id -u)" -eq 0 ]; then
    SUDO=()
else
    SUDO=(sudo)
fi

need_cmd() {
    if ! command -v "$1" >/dev/null 2>&1; then
        echo "required command not found: $1" >&2
        exit 1
    fi
}

iptables_cmd() {
    "${SUDO[@]}" iptables "$@"
}

remove_iptables_rule() {
    local chain="$1"
    shift

    while iptables_cmd -C "$chain" "$@" >/dev/null 2>&1; do
        iptables_cmd -D "$chain" "$@"
    done
}

remove_firewall_rules() {
    remove_iptables_rule DOCKER-USER \
        -i "$HYDRA_BRIDGE_NAME" \
        -m conntrack --ctstate ESTABLISHED,RELATED \
        -m comment --comment "${IPTABLES_COMMENT_PREFIX}:allow-established" \
        -j RETURN

    remove_iptables_rule DOCKER-USER \
        -i "$HYDRA_BRIDGE_NAME" \
        -m conntrack --ctstate NEW \
        -m comment --comment "${IPTABLES_COMMENT_PREFIX}:drop-forward-new" \
        -j DROP

    remove_iptables_rule INPUT \
        -i "$HYDRA_BRIDGE_NAME" \
        -m conntrack --ctstate NEW \
        -m comment --comment "${IPTABLES_COMMENT_PREFIX}:drop-host-new" \
        -j DROP

    echo "hydra firewall rules are removed"
}

remove_network_if_requested() {
    if [ "$REMOVE_NETWORK" != true ]; then
        return
    fi

    if ! docker network inspect "$HYDRA_NET_NAME" >/dev/null 2>&1; then
        echo "hydra network does not exist: $HYDRA_NET_NAME"
        return
    fi

    local attached
    attached="$(docker network inspect "$HYDRA_NET_NAME" --format '{{len .Containers}}')"
    if [ "$attached" != "0" ]; then
        echo "refusing to remove $HYDRA_NET_NAME; attached containers: $attached" >&2
        exit 1
    fi

    docker network rm "$HYDRA_NET_NAME" >/dev/null
    echo "removed hydra network: $HYDRA_NET_NAME"
}

need_cmd docker
need_cmd iptables

remove_firewall_rules
remove_network_if_requested
