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
# ============================================================
set -euo pipefail

SRC_IMAGE="${1:?Usage: $0 <image> [protocol] [timeout]}"
PROTOCOL="${2:-grpc}"
TIMEOUT="${3:-480}"

# ── Config ────────────────────────────────────────────────────
CLUSTER_NAME="arcana-ci-go-${PROTOCOL}"
CI_IMAGE="arcana-cloud-go:ci"
NS="arcana-ci-kind-go"
NODE_PORT=30092
MANIFEST="deployment/kubernetes/ci/kind-ci-grpc.yaml"

echo "=== Kind K8s Smoke Test: ${PROTOCOL} ==="
echo "    Cluster:  ${CLUSTER_NAME}"
echo "    Image:    ${SRC_IMAGE} → ${CI_IMAGE}"
echo "    Manifest: ${MANIFEST}"
echo "    Timeout:  ${TIMEOUT}s"

# ── Cleanup trap ─────────────────────────────────────────────
cleanup() {
    echo "[cleanup] Disconnecting from kind network ..."
    docker network disconnect kind "$(hostname)" 2>/dev/null || true
    echo "[cleanup] Deleting kind cluster ${CLUSTER_NAME} ..."
    kind delete cluster --name "${CLUSTER_NAME}" 2>/dev/null || true
}
trap cleanup EXIT

# ── 1. Create Kind cluster ───────────────────────────────────
echo "[kind] Creating cluster ${CLUSTER_NAME} ..."
kind create cluster --name "${CLUSTER_NAME}" --wait 60s

# Connect Jenkins container to kind network so kubectl can reach the control plane
JENKINS_CONTAINER=$(hostname)
docker network connect kind "${JENKINS_CONTAINER}" 2>/dev/null || true

# Rewrite kubeconfig: 127.0.0.1 → kind control-plane container IP
CP_IP=$(docker inspect "${CLUSTER_NAME}-control-plane" \
    --format '{{.NetworkSettings.Networks.kind.IPAddress}}' 2>/dev/null || echo "")
if [ -n "${CP_IP}" ]; then
    echo "[kind] Rewriting kubeconfig server to ${CP_IP} (Jenkins joined kind network) ..."
    kubectl config set-cluster "kind-${CLUSTER_NAME}" \
        --server="https://${CP_IP}:6443" \
        --insecure-skip-tls-verify=true
fi

# Verify connectivity
kubectl cluster-info
echo "[kind] Cluster ready"

# ── 2. Tag and load image ───────────────────────────────────
echo "[kind] Loading image ${SRC_IMAGE} as ${CI_IMAGE} ..."
if ! docker image inspect "${SRC_IMAGE}" > /dev/null 2>&1; then
    # Jenkins container can't reach localhost:5000 (that's the host's registry)
    # Try pulling via Docker bridge gateway instead
    BRIDGE_IMAGE=$(echo "${SRC_IMAGE}" | sed 's|localhost:5000|172.17.0.1:5000|')
    echo "[kind] Pulling ${BRIDGE_IMAGE} from registry ..."
    docker pull "${BRIDGE_IMAGE}"
    docker tag "${BRIDGE_IMAGE}" "${SRC_IMAGE}"
fi
docker tag "${SRC_IMAGE}" "${CI_IMAGE}"
kind load docker-image "${CI_IMAGE}" --name "${CLUSTER_NAME}"
echo "[kind] Image loaded"

# ── Re-verify API connectivity after image load ──────────────
# Image loading can briefly disrupt Docker networking
echo "[kind] Verifying API server after image load ..."
for attempt in $(seq 1 6); do
    if kubectl cluster-info >/dev/null 2>&1; then
        echo "[kind] API server OK"
        break
    fi
    echo "[kind] API server unreachable, reconnecting (attempt ${attempt}/6) ..."
    docker network disconnect kind "${JENKINS_CONTAINER}" 2>/dev/null || true
    sleep 2
    docker network connect kind "${JENKINS_CONTAINER}" 2>/dev/null || true
    # Re-resolve IP in case it changed
    NEW_IP=$(docker inspect "${CLUSTER_NAME}-control-plane" \
        --format '{{.NetworkSettings.Networks.kind.IPAddress}}' 2>/dev/null || echo "")
    if [ -n "${NEW_IP}" ]; then
        CP_IP="${NEW_IP}"
        kubectl config set-cluster "kind-${CLUSTER_NAME}" \
            --server="https://${CP_IP}:6443" \
            --insecure-skip-tls-verify=true >/dev/null 2>&1
    fi
    sleep 3
done
kubectl cluster-info >/dev/null 2>&1 || { echo "[kind] API server unreachable after retries"; exit 1; }

# ── 3. Apply manifest ───────────────────────────────────────
echo "[k8s] Applying manifest ${MANIFEST} ..."
for attempt in $(seq 1 3); do
    if kubectl apply -f "${MANIFEST}" 2>&1; then
        break
    fi
    echo "[k8s] kubectl apply attempt ${attempt}/3 failed, retrying in 5s ..."
    sleep 5
done

# ── 4. Wait for pods to be ready ────────────────────────────
# Wait per-deployment like the Node.js project does (more robust than counting)
wait_pods() {
    local label="$1"
    local elapsed=0
    local interval=10

    echo "[k8s] Waiting for pods (${label}) to be ready ..."
    while true; do
        local ready
        ready=$(kubectl get pods -n "${NS}" -l "app=${label}" \
            --field-selector=status.phase=Running \
            -o jsonpath='{.items[*].status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "")

        if [[ "${ready}" == *"True"* ]]; then
            echo "[k8s] Pod ${label} is ready after ${elapsed}s"
            return 0
        fi

        if [[ ${elapsed} -ge ${TIMEOUT} ]]; then
            echo "[k8s] TIMEOUT waiting for ${label} pods"
            kubectl get pods -n "${NS}" 2>/dev/null || true
            kubectl describe pods -n "${NS}" -l "app=${label}" 2>/dev/null | tail -30 || true
            return 1
        fi

        sleep ${interval}
        elapsed=$((elapsed + interval))
        echo "[k8s] ...${elapsed}s elapsed, waiting for ${label}"
    done
}

echo "[k8s] Waiting for infrastructure pods ..."
kubectl wait deployment/mysql -n "${NS}" \
    --for=condition=Available --timeout=120s || true
kubectl wait deployment/redis -n "${NS}" \
    --for=condition=Available --timeout=60s || true
echo "[k8s] Infrastructure pods up"

# Wait for all 3 app layers in order
wait_pods "arcana-ci-repository"
wait_pods "arcana-ci-service"
wait_pods "arcana-ci-controller"

# ── 5. Get NodePort address ──────────────────────────────────
NODE_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}' 2>/dev/null || echo "${CP_IP}")
BASE_URL="http://${NODE_IP}:${NODE_PORT}"

echo "[test] Smoke testing ${BASE_URL} ..."

# ── 6. Wait for controller health ────────────────────────────
elapsed=0
interval=10
echo "[test] Waiting for controller health endpoint ..."
while true; do
    if curl -sf --max-time 5 "${BASE_URL}/health" > /dev/null 2>&1; then
        echo "[test] Controller is healthy after ${elapsed}s"
        break
    fi
    if [[ ${elapsed} -ge ${TIMEOUT} ]]; then
        echo "[test] TIMEOUT waiting for health endpoint"
        kubectl get pods -n "${NS}" 2>/dev/null || true
        exit 1
    fi
    sleep ${interval}
    elapsed=$((elapsed + interval))
    echo "[test] ...${elapsed}s elapsed"
done

# ── 7. Smoke tests ───────────────────────────────────────────
PASS=0
FAIL=0

run_test() {
    local desc="$1"
    local expected="$2"
    local actual="$3"
    if echo "${actual}" | grep -qF "${expected}"; then
        echo "[PASS] ${desc}"
        PASS=$((PASS + 1))
    else
        echo "[FAIL] ${desc} — expected '${expected}' in: ${actual}"
        FAIL=$((FAIL + 1))
    fi
}

HEALTH=$(curl -sf --max-time 10 "${BASE_URL}/health" || echo '{}')
run_test "GET /health" "healthy" "${HEALTH}"

TIMESTAMP=$(date +%s%3N)
TEST_USER="kindsmoke${TIMESTAMP}"
TEST_EMAIL="${TEST_USER}@test.arcana"
TEST_PASS="KindSmoke@123!"

REGISTER=$(curl -sf --max-time 15 \
    -X POST "${BASE_URL}/api/v1/auth/register" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"${TEST_USER}\",\"email\":\"${TEST_EMAIL}\",\"password\":\"${TEST_PASS}\"}" \
    || echo '{"error":"register_failed"}')
run_test "POST /api/v1/auth/register" "access_token" "${REGISTER}"

ACCESS_TOKEN=$(echo "${REGISTER}" | grep -o '"access_token":"[^"]*"' | head -1 | cut -d'"' -f4 || echo "")

if [[ -n "${ACCESS_TOKEN}" ]]; then
    LOGIN=$(curl -sf --max-time 15 \
        -X POST "${BASE_URL}/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"username_or_email\":\"${TEST_USER}\",\"password\":\"${TEST_PASS}\"}" \
        || echo '{"error":"login_failed"}')
    run_test "POST /api/v1/auth/login" "access_token" "${LOGIN}"
else
    echo "[SKIP] Skipping login test (no token from register)"
    FAIL=$((FAIL + 1))
fi

# ── Summary ──────────────────────────────────────────────────
TOTAL=$((PASS + FAIL))
echo ""
echo "=== Kind Results [${PROTOCOL}]: ${PASS}/${TOTAL} passed ==="

if [[ ${FAIL} -gt 0 ]]; then
    echo "KIND SMOKE TEST FAILED"
    exit 1
else
    echo "KIND SMOKE TEST PASSED"
    exit 0
fi
