#!/usr/bin/env bash
set -euo pipefail

# Deploy helper for Kubernetes (deploy/k8s)
# Usage: bash deploy/deploy.sh --sha <git-sha>

NAMESPACE=${NAMESPACE:-socialpublish}
KUBECTL=${KUBECTL:-kubectl}
TIMEOUT_JOB=${TIMEOUT_JOB:-300s}
TIMEOUT_ROLLOUT=${TIMEOUT_ROLLOUT:-120s}

show_help(){
  cat <<EOF
Usage: $0 --sha <git-sha> [--server-image image] [--worker-image image] [--migrate-image image]

This script applies kustomize manifests, updates images, runs migrations and waits for rollouts.
EOF
}

if [ "$#" -eq 0 ]; then
  show_help
  exit 1
fi

SHA=""
SERVER_IMAGE=""
WORKER_IMAGE=""
MIGRATE_IMAGE=""

while [ "$#" -gt 0 ]; do
  case "$1" in
    --sha) SHA="$2"; shift 2;;
    --server-image) SERVER_IMAGE="$2"; shift 2;;
    --worker-image) WORKER_IMAGE="$2"; shift 2;;
    --migrate-image) MIGRATE_IMAGE="$2"; shift 2;;
    -h|--help) show_help; exit 0;;
    *) echo "Unknown arg: $1"; show_help; exit 1;;
  esac
done

if [ -z "$SHA" ] && [ -z "$SERVER_IMAGE" ]; then
  echo "ERROR: --sha or --server-image is required"
  show_help
  exit 2
fi

# Default image names when SHA provided
if [ -n "$SHA" ]; then
  : ${SERVER_IMAGE:="ghcr.io/mohitsharma-in/socialpublish:server-${SHA}"}
  : ${WORKER_IMAGE:="ghcr.io/mohitsharma-in/socialpublish:worker-${SHA}"}
  : ${MIGRATE_IMAGE:="ghcr.io/mohitsharma-in/socialpublish:migrate-${SHA}"}
fi

echo "Namespace: $NAMESPACE"
echo "Server image: $SERVER_IMAGE"
echo "Worker image: $WORKER_IMAGE"
echo "Migrate image: $MIGRATE_IMAGE"

# Pre-flight
command -v "$KUBECTL" >/dev/null || { echo "kubectl not found in PATH"; exit 1; }

echo "Applying manifests (kustomize)..."
$KUBECTL apply -k deploy/k8s

echo "Setting deployment images..."
$KUBECTL -n "$NAMESPACE" set image deployment/socialpublish-api api="$SERVER_IMAGE" --record
$KUBECTL -n "$NAMESPACE" set image deployment/socialpublish-worker worker="$WORKER_IMAGE" --record

# Replace migrate job image and apply
echo "Running migration job..."
$KUBECTL -n "$NAMESPACE" delete job socialpublish-migrate --ignore-not-found
sed "s|ghcr.io/mohitsharma-in/socialpublish:migrate-latest|${MIGRATE_IMAGE}|g" deploy/k8s/migrate-job.yaml | $KUBECTL -n "$NAMESPACE" apply -f -

# Wait for job completion and print logs on failure
echo "Waiting for migration job to complete (timeout: $TIMEOUT_JOB)"
if ! $KUBECTL -n "$NAMESPACE" wait --for=condition=complete --timeout=$TIMEOUT_JOB job/socialpublish-migrate; then
  echo "Migration job failed or timed out. Collecting logs..."
  PODS=$($KUBECTL -n "$NAMESPACE" get pods -l job-name=socialpublish-migrate -o jsonpath='{.items[*].metadata.name}') || PODS=""
  for pod in $PODS; do
    echo "---- logs from pod: $pod ----"
    $KUBECTL -n "$NAMESPACE" logs "$pod" || true
  done
  exit 1
fi

# Wait for deployments to roll out
echo "Waiting for API rollout (timeout: $TIMEOUT_ROLLOUT)"
$KUBECTL -n "$NAMESPACE" rollout status deployment/socialpublish-api --timeout=$TIMEOUT_ROLLOUT

echo "Waiting for Worker rollout (timeout: $TIMEOUT_ROLLOUT)"
$KUBECTL -n "$NAMESPACE" rollout status deployment/socialpublish-worker --timeout=$TIMEOUT_ROLLOUT

echo "Deployment successful"

# Helpful final status
$KUBECTL -n "$NAMESPACE" get deploy -o wide
$KUBECTL -n "$NAMESPACE" get pods -o wide

exit 0
