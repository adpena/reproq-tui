#!/usr/bin/env bash
set -euo pipefail

WORKER_URL="${REPROQ_WORKER_URL:-}"
WORKER_METRICS_URL="${REPROQ_WORKER_METRICS_URL:-}"
WORKER_HEALTH_URL="${REPROQ_WORKER_HEALTH_URL:-}"
DJANGO_URL="${REPROQ_DJANGO_URL:-}"
DJANGO_STATS_URL="${REPROQ_DJANGO_STATS_URL:-}"
EVENTS_URL="${REPROQ_TUI_EVENTS_URL:-${REPROQ_EVENTS_URL:-}}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --worker-url)
      WORKER_URL="$2"
      shift 2
      ;;
    --worker-metrics-url)
      WORKER_METRICS_URL="$2"
      shift 2
      ;;
    --worker-health-url)
      WORKER_HEALTH_URL="$2"
      shift 2
      ;;
    --events-url)
      EVENTS_URL="$2"
      shift 2
      ;;
    --django-url)
      DJANGO_URL="$2"
      shift 2
      ;;
    --django-stats-url)
      DJANGO_STATS_URL="$2"
      shift 2
      ;;
    *)
      break
      ;;
  esac
done

if [[ -z "${WORKER_URL}" && -z "${WORKER_METRICS_URL}" ]]; then
  echo "Missing --worker-url or --worker-metrics-url." >&2
  exit 1
fi

args=(setup)
if [[ -n "${WORKER_URL}" ]]; then
  args+=(--worker-url "${WORKER_URL}")
fi
if [[ -n "${WORKER_METRICS_URL}" ]]; then
  args+=(--worker-metrics-url "${WORKER_METRICS_URL}")
fi
if [[ -n "${WORKER_HEALTH_URL}" ]]; then
  args+=(--worker-health-url "${WORKER_HEALTH_URL}")
fi
if [[ -n "${EVENTS_URL}" ]]; then
  args+=(--events-url "${EVENTS_URL}")
fi
if [[ -n "${DJANGO_URL}" ]]; then
  args+=(--django-url "${DJANGO_URL}")
fi
if [[ -n "${DJANGO_STATS_URL}" ]]; then
  args+=(--django-stats-url "${DJANGO_STATS_URL}")
fi

exec reproq-tui "${args[@]}" "$@"
