#!/bin/bash
# GateKey OpenVPN Hook Script
# This script is called by OpenVPN for various hook events.
# It delegates to the gatekey-gateway binary for actual processing.

set -e

# Configuration
gatekey_GATEWAY_BIN="${gatekey_GATEWAY_BIN:-/usr/local/bin/gatekey-gateway}"
gatekey_CONFIG="${gatekey_CONFIG:-/etc/gatekey/gateway.yaml}"
HOOK_TYPE="${script_type:-unknown}"

# Log function
log() {
    logger -t "gatekey-hook" "$1"
}

# Determine hook type from script name or environment
case "$(basename "$0")" in
    auth-user-pass-verify*)
        HOOK_TYPE="auth-user-pass-verify"
        ;;
    tls-verify*)
        HOOK_TYPE="tls-verify"
        ;;
    client-connect*)
        HOOK_TYPE="client-connect"
        ;;
    client-disconnect*)
        HOOK_TYPE="client-disconnect"
        ;;
esac

# Execute the gateway binary with the hook type
exec "$gatekey_GATEWAY_BIN" hook \
    --config "$gatekey_CONFIG" \
    --type "$HOOK_TYPE" \
    "$@"
