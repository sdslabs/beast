#!/usr/bin/env bash
set -euo pipefail

HYDRA_NET_NAME="${HYDRA_NET_NAME:-hydra-net}"
HYDRA_BRIDGE_NAME="${HYDRA_BRIDGE_NAME:-hydra0}"
IPTABLES_COMMENT_PREFIX="beast:${HYDRA_NET_NAME}"

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

ensure_iptables_chain() {
    if ! iptables_cmd -nL DOCKER-USER >/dev/null 2>&1; then
        iptables_cmd -N DOCKER-USER
        iptables_cmd -I FORWARD -j DOCKER-USER
    fi
}

ensure_iptables_rule() {
    local chain="$1"
    local position="$2"
    shift 2

    if iptables_cmd -C "$chain" "$@" >/dev/null 2>&1; then
        return
    fi

    if [ "$position" = "append" ]; then
        iptables_cmd -A "$chain" "$@"
    else
        iptables_cmd -I "$chain" "$position" "$@"
    fi
}

ensure_hydra_network() {
    if docker network inspect "$HYDRA_NET_NAME" >/dev/null 2>&1; then
        local driver
        local bridge
        driver="$(docker network inspect "$HYDRA_NET_NAME" --format '{{.Driver}}')"
        bridge="$(docker network inspect "$HYDRA_NET_NAME" --format '{{index .Options "com.docker.network.bridge.name"}}')"

        if [ "$driver" != "bridge" ]; then
            echo "existing network $HYDRA_NET_NAME must use bridge driver, got: $driver" >&2
            exit 1
        fi
        if [ "$bridge" != "$HYDRA_BRIDGE_NAME" ]; then
            echo "existing network $HYDRA_NET_NAME must use bridge $HYDRA_BRIDGE_NAME, got: $bridge" >&2
            exit 1
        fi
        echo "hydra network already exists: $HYDRA_NET_NAME ($HYDRA_BRIDGE_NAME)"
        return
    fi

    docker network create \
        --driver bridge \
        --opt "com.docker.network.bridge.name=${HYDRA_BRIDGE_NAME}" \
        --opt "com.docker.network.bridge.enable_icc=false" \
        --label "com.sdslabs.beast.managed=true" \
        "$HYDRA_NET_NAME" >/dev/null

    echo "created hydra network: $HYDRA_NET_NAME ($HYDRA_BRIDGE_NAME)"
}

ensure_firewall_rules() {
    ensure_iptables_chain

    ensure_iptables_rule DOCKER-USER 1 \
        -i "$HYDRA_BRIDGE_NAME" \
        -m conntrack --ctstate ESTABLISHED,RELATED \
        -m comment --comment "${IPTABLES_COMMENT_PREFIX}:allow-established" \
        -j RETURN

    ensure_iptables_rule DOCKER-USER append \
        -i "$HYDRA_BRIDGE_NAME" \
        -m conntrack --ctstate NEW \
        -m comment --comment "${IPTABLES_COMMENT_PREFIX}:drop-forward-new" \
        -j DROP

    ensure_iptables_rule INPUT append \
        -i "$HYDRA_BRIDGE_NAME" \
        -m conntrack --ctstate NEW \
        -m comment --comment "${IPTABLES_COMMENT_PREFIX}:drop-host-new" \
        -j DROP

    echo "hydra firewall rules are installed"
}

need_cmd docker
need_cmd iptables

ensure_hydra_network
ensure_firewall_rules
