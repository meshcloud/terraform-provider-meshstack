#!/usr/bin/env bash
#
# Gate the acceptance job on the backend service containers being ready. Used by
# .github/workflows/test.yml:
#
#   wait-for-backend.sh
#
# Each service is polled until its health endpoint answers HTTP 200, or until its deadline passes — at
# which point every service is diagnosed and the script fails the step.
#
# Ports are the services' *actuator* (management) ports, not their app ports: meshfed-api, replicator
# and block-coordinator-api all set management.server.port, which takes /actuator/health off the app
# port entirely. Probing an app port instead reaches the application's own filter chain, which answers
# 401 and looks indistinguishable from "not ready yet" — see the block-coordinator note below.
#
# ARC runs the job in kubernetes container mode, where the job container and the service containers
# share a network namespace, so every service is reachable on localhost. That mode also means there is
# no docker or kubectl in the job container: a service container's own logs are unreachable, and its
# HTTP response is the only window into why it is unhealthy. Hence diagnose_backend below, which prints
# the status and body rather than discarding them.
set -euo pipefail

# name|health url, in the order they are waited for.
HEALTH_ENDPOINTS=(
  "meshfed-api|http://localhost:8180/actuator/health"
  "replicator|http://localhost:7180/actuator/health"
  "block-coordinator-api|http://localhost:8183/actuator/health"
  "mux|http://localhost:8309/healthz"
)

# Print what a health endpoint actually returned, separating the two failure modes the readiness gate
# otherwise reports identically: no HTTP response at all means the container is down or crash-looping
# (e.g. the OOM-kill the service heap caps in the workflow guard against), while a 4xx/5xx means the
# process is up — and for a real actuator endpoint the body names the component that is unhealthy.
probe_health() {
  local url="$1" out status rc=0
  out=$(curl -s -m 10 -w $'\n%{http_code}' "$url" 2>&1) || rc=$?
  if [ "$rc" -ne 0 ]; then
    echo "    no HTTP response (curl exit $rc): container is down, restarting, or not listening"
    return
  fi
  status="${out##*$'\n'}"
  echo "    HTTP $status: $(head -c 1500 <<<"${out%$'\n'*}")"
}

# Called only when a service misses its deadline, so a green run stays quiet.
diagnose_backend() {
  echo "::group::backend diagnostics"
  local entry
  for entry in "${HEALTH_ENDPOINTS[@]}"; do
    echo "  ${entry%%|*}:"
    probe_health "${entry#*|}"
  done
  # Service containers share the pod's memory budget, so record the headroom: a container killed for
  # overcommitting is the documented cause of a truncated acceptance run.
  echo "  job container memory (bytes):"
  echo "    max=$(cat /sys/fs/cgroup/memory.max 2>/dev/null || echo n/a)" \
    "current=$(cat /sys/fs/cgroup/memory.current 2>/dev/null || echo n/a)" \
    "peak=$(cat /sys/fs/cgroup/memory.peak 2>/dev/null || echo n/a)"
  echo "  node memory:"
  free -m 2>/dev/null | sed 's/^/    /' || echo "    free(1) unavailable"
  echo "::endgroup::"
}

# wait_http <url> <name> [deadline seconds]
wait_http() {
  local url="$1" name="$2" deadline=$((SECONDS + ${3:-360})) status
  echo "waiting for $name ($url)..."
  while true; do
    # Capture the status instead of relying on curl's exit code, so a timeout can report what the
    # endpoint last answered. curl already writes 000 for a connection-level failure, so only an empty
    # result needs covering.
    status=$(curl -s -o /dev/null -m 10 -w '%{http_code}' "$url" 2>/dev/null || true)
    if [ "${status:-000}" = 200 ]; then
      echo "$name healthy"
      return 0
    fi
    if [ "$SECONDS" -gt "$deadline" ]; then
      echo "::error::timeout waiting for $name ($url), last status HTTP ${status:-000}"
      diagnose_backend
      return 1
    fi
    sleep 3
  done
}

# meshfed-api is the gate: it bootstraps API keys into Keycloak, so tests run before it is healthy
# fail with auth errors.
wait_http http://localhost:8180/actuator/health meshfed-api 420
wait_http http://localhost:7180/actuator/health replicator 240
# block-coordinator-api answers /actuator/health only on its management port (8183) since
# meshfed-release set management.server.port for Prometheus scraping. The app port 8083 serves the
# runner API, whose filter chain requires the BLOCK_RUNNER role, so probing it yields 401 and the gate
# waits out the whole deadline on a healthy container. meshfed-release fixed its own CI the same way
# in 6ce17457ed.
wait_http http://localhost:8183/actuator/health block-coordinator-api 240
wait_http http://localhost:8309/healthz mux 120

echo "mux routing:"
curl -s http://localhost:8309/status || true
