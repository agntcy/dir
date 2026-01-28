#!/usr/bin/env bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0
#
# Test script for sync operations between Directory nodes.
# Deploys source (Zot-backed) and target (Zot + PostgreSQL + Reconciler) nodes.
# Usage: ./scripts/test-sync.sh

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
SOURCE_NAMESPACE="${SOURCE_NAMESPACE:-dir-source}"
TARGET_NAMESPACE="${TARGET_NAMESPACE:-dir-target}"
SOURCE_DIR_ADDRESS="${SOURCE_DIR_ADDRESS:-localhost:8881}"
TARGET_DIR_ADDRESS="${TARGET_DIR_ADDRESS:-localhost:8882}"
# Internal Kubernetes DNS addresses (used by sync workers inside the cluster)
SOURCE_DIR_INTERNAL="dir-apiserver.${SOURCE_NAMESPACE}.svc.cluster.local:8888"
TARGET_DIR_INTERNAL="dir-apiserver.${TARGET_NAMESPACE}.svc.cluster.local:8888"
SYNC_TIMEOUT="${SYNC_TIMEOUT:-120}"

# Colors
RED='\033[0;31m' GREEN='\033[0;32m' YELLOW='\033[1;33m' BLUE='\033[0;34m' NC='\033[0m'

log() { echo -e "${2:-$GREEN}[$1]${NC} $3"; }
log_info() { log "INFO" "$GREEN" "$1"; }
log_warn() { log "WARN" "$YELLOW" "$1"; }
log_error() { log "ERROR" "$RED" "$1"; }
log_step() { log "STEP" "$BLUE" "$1"; }

check_prerequisites() {
    log_info "Checking prerequisites..."
    for cmd in task dirctl kubectl helm; do
        command -v "$cmd" &>/dev/null || { log_error "$cmd not found in PATH"; exit 1; }
    done
    log_info "Prerequisites check passed"
}

deploy_source_node() {
    log_step "Deploying source Directory node (Zot-backed)..."
    cd "$ROOT_DIR"
    HELM_NAMESPACE="$SOURCE_NAMESPACE" DIRECTORY_SERVER_OASF_API_VALIDATION_DISABLE=true task deploy:kubernetes:local
    log_info "Source node deployed in namespace: $SOURCE_NAMESPACE"
}

deploy_target_node() {
    log_step "Deploying target Directory node (Zot + PostgreSQL + Reconciler)..."
    cd "$ROOT_DIR"
    HELM_NAMESPACE="$TARGET_NAMESPACE" task deploy:kubernetes:local:ghcr
    log_info "Target node deployed in namespace: $TARGET_NAMESPACE"
}

setup_port_forwarding() {
    log_step "Setting up port forwarding..."
    pkill -f "port-forward.*${SOURCE_NAMESPACE}" || true
    pkill -f "port-forward.*${TARGET_NAMESPACE}" || true
    sleep 2

    kubectl port-forward service/dir-apiserver 8881:8888 -n "$SOURCE_NAMESPACE" &
    kubectl port-forward service/dir-apiserver 8882:8888 -n "$TARGET_NAMESPACE" &
    sleep 5

    for port in 8881 8882; do
        nc -z localhost "$port" 2>/dev/null || { log_error "Port-forward failed ($port)"; return 1; }
    done
    log_info "Port forwarding established (8881->source, 8882->target)"
}

get_record() {
    local record="$SCRIPT_DIR/$1"
    [ -f "$record" ] || { log_error "Record not found: $record"; exit 1; }
    echo "$record"
}

push_record() {
    local manifest=$1 address=$2
    log_info "Pushing record to $address..." >&2
    local cid
    cid=$(dirctl push "$manifest" --server-addr "$address" --output raw 2>&1) || true
    [ -n "$cid" ] && [[ ! "$cid" =~ ^Error ]] || { log_error "Failed to push record. Output: $cid" >&2; return 1; }
    log_info "Record pushed. CID: $cid" >&2
    echo "$cid"
}

pull_record() {
    local cid=$1 address=$2 output_file="${3:-/tmp/pulled-${1}.json}"
    log_info "Pulling record $cid from $address..."
    dirctl pull "$cid" --server-addr "$address" -o "$output_file" 2>&1 && log_info "Record pulled successfully"
}

trigger_sync() {
    local src=$1 tgt=$2
    log_info "Triggering sync from $src to $tgt..."
    dirctl sync create "$src" --server-addr "$tgt" 2>&1 && log_info "Sync triggered successfully"
}

wait_for_sync() {
    local cid=$1 address=$2 timeout=$3 start_time elapsed
    start_time=$(date +%s)
    log_info "Waiting for sync (timeout: ${timeout}s)..."
    while true; do
        elapsed=$(($(date +%s) - start_time))
        [ $elapsed -gt "$timeout" ] && { log_error "Sync timeout after ${timeout}s"; return 1; }
        dirctl pull "$cid" --server-addr "$address" -o "/tmp/synced-${cid}.json" 2>/dev/null && {
            echo ""; log_info "Sync completed in ${elapsed}s"; return 0
        }
        echo -n "."; sleep 2
    done
}

check_reconciler_status() {
    log_info "Checking reconciler status..."
    kubectl get pods -n "$TARGET_NAMESPACE" -l app.kubernetes.io/component=reconciler -o wide 2>/dev/null || \
        kubectl get pods -n "$TARGET_NAMESPACE" | grep reconciler || true
    kubectl logs -n "$TARGET_NAMESPACE" -l app.kubernetes.io/component=reconciler --tail=30 2>/dev/null || \
        kubectl logs -n "$TARGET_NAMESPACE" deployment/dir-apiserver-reconciler --tail=30 2>/dev/null || true
}

check_sync_status() {
    log_info "Checking sync status in database..."
    local pg_pod
    pg_pod=$(kubectl get pods -n "$TARGET_NAMESPACE" -l app.kubernetes.io/name=postgresql -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
    [ -n "$pg_pod" ] && kubectl exec -n "$TARGET_NAMESPACE" "$pg_pod" -- psql -U dir -d dir -c \
        "SELECT id, status, remote_directory_url, created_at, updated_at FROM syncs ORDER BY created_at DESC LIMIT 5;" 2>/dev/null || \
        log_warn "PostgreSQL pod not found"
}

test_push_pull() {
    local name=$1 record=$2 address=$3 cid_var=$4
    log_step "Test: $name push/pull"
    local manifest cid
    manifest=$(get_record "$record")
    cid=$(push_record "$manifest" "$address") || return 1
    [ -n "$cid" ] || { log_error "Failed to get CID"; return 1; }
    eval "$cid_var='$cid'"
    pull_record "$cid" "$address" && log_info "$name push/pull: SUCCESS" || { log_error "$name push/pull: FAILED"; return 1; }
}

test_sync() {
    local name=$1 cid=$2 src_internal=$3 tgt_external=$4 tgt_poll=$5 show_debug=${6:-false}
    log_step "Test: $name sync"
    [ -n "$cid" ] || { log_error "CID not set"; return 1; }
    log_info "Syncing CID: $cid"
    # src_internal: Kubernetes internal address for the remote directory (used by sync worker inside cluster)
    # tgt_external: External address to talk to via port-forward (where to create the sync)
    # tgt_poll: External address to poll for sync completion
    trigger_sync "$src_internal" "$tgt_external" || { log_error "Failed to trigger sync"; return 1; }
    if wait_for_sync "$cid" "$tgt_poll" "$SYNC_TIMEOUT"; then
        log_info "$name sync: SUCCESS"
    else
        log_error "$name sync: FAILED"
        $show_debug && { check_reconciler_status; check_sync_status; }
        return 1
    fi
}

main() {
    log_info "=== Directory Sync Test Suite ==="
    check_prerequisites
    deploy_source_node
    deploy_target_node
    setup_port_forwarding
    log_info "Waiting for services to stabilize..."; sleep 10

    local failed=0 SOURCE_CID="" TARGET_CID=""

    test_push_pull "Source" "record_080_zot.json" "$SOURCE_DIR_ADDRESS" SOURCE_CID || ((failed++))
    test_push_pull "Target" "record_080_ghcr.json" "$TARGET_DIR_ADDRESS" TARGET_CID || ((failed++))
    # Source->Target: Tell target to sync from source (use internal address for remote URL)
    test_sync "Source->Target" "$SOURCE_CID" "$SOURCE_DIR_INTERNAL" "$TARGET_DIR_ADDRESS" "$TARGET_DIR_ADDRESS" true || ((failed++))
    # Target->Source: Tell source to sync from target (use internal address for remote URL)
    test_sync "Target->Source" "$TARGET_CID" "$TARGET_DIR_INTERNAL" "$SOURCE_DIR_ADDRESS" "$SOURCE_DIR_ADDRESS" false || ((failed++))

    log_step "=== Final Status ==="
    check_reconciler_status
    check_sync_status
    rm -f /tmp/pulled-*.json /tmp/synced-*.json

    if [ $failed -eq 0 ]; then
        log_info "=== All 4 tests passed! ==="
    else
        log_error "=== $failed test(s) failed ==="
        exit 1
    fi
}

main "$@"
