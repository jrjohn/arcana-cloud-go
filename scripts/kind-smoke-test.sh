#!/usr/bin/env bash
# ============================================================
# Kind Smoke Test — arcana-cloud-go K8s gRPC 3-layer CI
#
# Spins up a local Kind cluster, loads the built image,
# deploys the 3-layer gRPC stack, waits for pods, runs
# integration smoke tests via NodePort, then tears down.
#
# Usage:
#   bash scripts/kind-smoke-test.sh <SRC_IMAGE> [PROTOCOL] [TIMEOUT_SEC]
#   bash scripts/kind-smoke-test.sh localhost:5000/arcana/go-app:build-42 grpc 480
#
# Args:
#   SRC_IMAGE    Full Docker image tag built by Jenkins (required)
#   PROTOCOL     Label for smoke test output (default: grpc)
#   TIMEOUT_SEC  Max seconds to wait for pods (default: 480)
# ============================================================
set -euo pipefail

SRC_IMAGE="${1:?Usage: $0 <image> [protocol] [timeout]}"
PROTOCOL="${2:-grpc}"
TIMEOUT_SEC="${3:-480}"

# ── Config ────────────────────────────────────────────────────
NS="arcana-ci-kind-go"
NODE_PORT=30092
EXPECTED_PODS=3
APP_LABELS="app in (arcana-ci-repository,arcana-ci-service,arcana-ci-controller)"
CI_IMAGE="arcana-cloud-go:ci"
MANIFEST="deployment/kubernetes/ci/kind-ci-grpc.yaml"

# Unique cluster name per build (avoids collisions with parallel jobs)
CLUSTER_NAME="arcana-ci-$(date +%s)"
KUBECONFIG_FILE="/tmp/kind-kubeconfig-${CLUSTER_NAME}"

echo ""
echo "╔══════════════════════════════════════════════════════════╗"
echo "║  Kind Smoke Test — arcana-cloud-go K8s gRPC              ║"
echo "║  Image : ${SRC_IMAGE}"
echo "║  Cluster: ${CLUSTER_NAME}"
echo "╚══════════════════════════════════════════════════════════╝"

# ── Cleanup trap ─────────────────────────────────────────────
cleanup() {
    echo ""
    echo "▶ Cleanup: deleting Kind cluster '${CLUSTER_NAME}' ..."
    kind delete cluster --name "${CLUSTER_NAME}" 2>/dev/null || true
    rm -f "${KUBECONFIG_FILE}" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

# ── 1. Create Kind cluster ────────────────────────────────────
echo ""
echo "▶ [1/6] Creating Kind cluster '${CLUSTER_NAME}' ..."
kind create cluster --name "${CLUSTER_NAME}" --wait 60s
echo "  ✓ Cluster ready"

# ── Fix kubeconfig for Docker-in-Docker networking ───────────
# When Jenkins runs inside Docker, 127.0.0.1 in the kubeconfig refers to
# the Jenkins container, not the host. We must use the Kind node's
# internal Docker network IP (port 6443) instead.
CONTROL_PLANE="${CLUSTER_NAME}-control-plane"
KIND_NODE_IP=$(docker inspect "${CONTROL_PLANE}" \
    --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' 2>/dev/null \
    | head -1)

if [ -z "${KIND_NODE_IP}" ]; then
    # Fallback: try getting from kind network
    KIND_NODE_IP=$(docker inspect "${CONTROL_PLANE}" \
        --format '{{.NetworkSettings.Networks.kind.IPAddress}}' 2>/dev/null || echo "")
fi

echo "  Kind node IP: ${KIND_NODE_IP}"

# Write kubeconfig with internal IP replacing 127.0.0.1:<port>
kind get kubeconfig --name "${CLUSTER_NAME}" | \
    sed "s|server: https://127.0.0.1:[0-9]*|server: https://${KIND_NODE_IP}:6443|" \
    > "${KUBECONFIG_FILE}"

export KUBECONFIG="${KUBECONFIG_FILE}"
echo "  ✓ Kubeconfig configured for internal networking"

# ── 2. Tag and load image ─────────────────────────────────────
echo ""
echo "▶ [2/6] Tagging '${SRC_IMAGE}' → '${CI_IMAGE}' and loading into Kind ..."
docker tag "${SRC_IMAGE}" "${CI_IMAGE}"
kind load docker-image "${CI_IMAGE}" --name "${CLUSTER_NAME}"
echo "  ✓ Image loaded"

# ── 3. Apply manifest ────────────────────────────────────────
echo ""
echo "▶ [3/6] Applying manifest: ${MANIFEST} ..."
kubectl apply -f "${MANIFEST}" --validate=false
echo "  ✓ Manifest applied"

# ── 4. Wait for MySQL and Redis to be ready ──────────────────
echo ""
echo "▶ [4/6] Waiting for MySQL and Redis pods ..."
kubectl wait deployment/mysql -n "${NS}" \
    --for=condition=Available --timeout=120s || true
kubectl wait deployment/redis -n "${NS}" \
    --for=condition=Available --timeout=60s || true
echo "  ✓ Infrastructure pods up"

# ── 5. Wait for app pods to be ready ────────────────────────
echo ""
echo "▶ [5/6] Waiting for ${EXPECTED_PODS} app pods (timeout: ${TIMEOUT_SEC}s) ..."
ELAPSED=0
INTERVAL=10
while true; do
    READY=$(kubectl get pods -n "${NS}" -l "${APP_LABELS}" \
        --field-selector=status.phase=Running \
        -o jsonpath='{.items[*].status.containerStatuses[*].ready}' 2>/dev/null \
        | tr ' ' '\n' | grep -c "^true$" || echo "0")

    echo "  ... ${READY}/${EXPECTED_PODS} pods ready (${ELAPSED}s elapsed)"

    if [ "${READY}" -ge "${EXPECTED_PODS}" ]; then
        echo "  ✓ All ${EXPECTED_PODS} pods ready"
        break
    fi

    if [ "${ELAPSED}" -ge "${TIMEOUT_SEC}" ]; then
        echo "  ✗ Pods not ready after ${TIMEOUT_SEC}s"
        echo "  --- Pod status ---"
        kubectl get pods -n "${NS}" || true
        kubectl describe pods -n "${NS}" -l "${APP_LABELS}" | tail -60 || true
        exit 1
    fi

    sleep "${INTERVAL}"
    ELAPSED=$((ELAPSED + INTERVAL))
done

# ── 6. Run smoke test via NodePort ───────────────────────────
echo ""
echo "▶ [6/6] Running integration smoke test via NodePort ${NODE_PORT} ..."
BASE_URL="http://${KIND_NODE_IP}:${NODE_PORT}"
echo "  Base URL: ${BASE_URL}"

bash scripts/integration-smoke-test.sh "${BASE_URL}" "k8s-grpc" 120

echo ""
echo "╔══════════════════════════════════════════════════════════╗"
echo "║  ✅ Kind K8s gRPC smoke test PASSED                      ║"
echo "╚══════════════════════════════════════════════════════════╝"
echo ""
