# Monitoring Metrics Enhancement Design

## Context

The Ketches platform UI has basic Prometheus-backed monitoring for clusters, environments, and applications. This design enriches metrics coverage, adds instance-level metrics, improves empty states, introduces color-coded utilization thresholds, configurable time ranges, and a code repository stats overview.

## Requirements

1. **Cluster & Environment Metrics Enrichment** — Add pod count, container restart rate, node readiness (cluster); CPU/memory utilization %, pod count (environment)
2. **Instance Metrics Dialog** — New "Metrics" action on application instances opening a 90vw×90vh dialog with per-pod monitoring charts
3. **Empty States** — Use `EmptyState` component when no data; show specific message when cluster lacks Prometheus integration
4. **Color-coded Utilization** — Blue (0-70%), orange (70-85%), red (85%+) for utilization charts (line color + header badge)
5. **Time Range Selector** — 5m, 15m, 30m, 1h, 4h options; each metrics section manages its own range
6. **Code Repository Stats** — Overview card showing build configs count, total builds, success rate, deployments

## Approach: Hybrid Utilities

Extract shared logic into small utilities/hooks while keeping chart rendering inline per component.

### Shared Utilities (new files)

| File | Purpose |
|------|---------|
| `src/components/monitoring/use-time-range.ts` | `useTimeRange()` hook — manages time range state, returns `{ timeRange, setTimeRange, rangeSeconds, step }` |
| `src/components/monitoring/use-prometheus-available.ts` | `usePrometheusAvailable(clusterId)` hook — checks if Prometheus integration exists via `clustersApi.listIntegrations` |
| `src/components/monitoring/metrics-time-range-selector.tsx` | Toggle group UI with 5m/15m/30m/1h/4h options |
| `src/components/monitoring/metrics-utils.ts` | `getUtilizationColor(value)` and `getUtilizationColorVar(value)` |

### Time Range Configuration

| Range | Step | Data Points |
|-------|------|-------------|
| 5m | 15s | ~20 |
| 15m | 30s | ~30 |
| 30m | 60s | ~30 |
| 1h | 60s | ~60 |
| 4h | 120s | ~120 |

Default: `1h` (preserves current behavior).

### Color Thresholds

| Range | Color | CSS |
|-------|-------|-----|
| 0-70% | Blue | `text-blue-500` / `hsl(217, 91%, 60%)` |
| 70-85% | Orange | `text-orange-500` / `hsl(25, 95%, 53%)` |
| 85%+ | Red | `text-red-500` / `hsl(0, 72%, 51%)` |

Applied to: utilization chart line stroke + current value badge in card header.

### Prometheus Availability Check

Before rendering any metrics, each component calls `usePrometheusAvailable(clusterId)`.

- If `available === false`: Show `EmptyState` with title "Prometheus Not Available" and description prompting the user to contact their administrator.
- If available but no data: Show `EmptyState` with "No Metrics Data" and generic description.
- If error: Show existing destructive `Empty` with "Check Cluster Configuration" button.

## Feature Details

### 1. Cluster Metrics Enrichment

**File**: `cluster-node-resource-metrics.tsx`

New PromQL queries:
- Pod count: `sum(kubelet_running_pods{})` (with optional node filter)
- Container restarts: `sum(increase(kube_pod_container_status_restarts_total{container!=""}[5m]))` (with optional node filter)
- Node readiness: `sum(kube_node_status_condition{condition="Ready",status="true"})`

Layout: Add a third row with 3 cards (pod count, restart rate, node readiness). Add `MetricsTimeRangeSelector` above all charts.

### 2. Environment Metrics Enrichment

**File**: `environment-resource-metrics.tsx`

New PromQL queries:
- CPU utilization: `sum(rate(container_cpu_usage_seconds_total{namespace="..."}[5m])) / sum(kube_resourcequota{namespace="...",resource="requests.cpu"}) * 100`
- Memory utilization: similar with memory resource quota
- Pod count: `count(kube_pod_info{namespace="..."})`

Layout: Expand from 3 to 6 cards (2 rows of 3). Add time range selector.

### 3. Instance Metrics Dialog

**New file**: `instance-resource-metrics.tsx`

Props: `{ open, onOpenChange, clusterId, namespace, podName, app }`

Dialog: `max-w-[90vw] max-h-[90vh]` with scrollable content.

Charts (2x2 grid + 1 full-width):
- CPU Usage (mCores) with request/limit reference lines
- Memory Usage (GiB) with request/limit reference lines
- CPU Utilization (%) — color-coded
- Memory Utilization (%) — color-coded
- Network Traffic (ingress + egress) — full width

PromQL queries filter by `pod="{podName}"` (exact match, not regex).

Trigger: New `BarChart3` icon button in instance actions column of `application-detail-page.tsx`.

### 4. App Metrics Updates

**File**: `application-detail-page.tsx` (AppMetrics function)

Changes:
- Add `useTimeRange()` + `MetricsTimeRangeSelector`
- Add `usePrometheusAvailable` check
- Replace `return null` when no data with `EmptyState`
- Add color-coded utilization for existing CPU/Memory util charts

### 5. Code Repository Stats Card

**File**: `code-repository-detail-page.tsx`

New card in the Overview tab below "Repository Information":
- Title: "Repository Statistics" with `BarChart3` icon
- 4-column grid of stats:
  - Build Configs (count, `Hammer` icon)
  - Total Builds (count, `Play` icon)
  - Success Rate (%, `CheckCircle` icon, color: green ≥90%, orange 70-90%, red <70%)
  - Deployments (count, `Rocket` icon)

Data from existing paginated queries (use `total` from pagination response).

## Files Changed

### New files (5):
- `src/components/monitoring/use-time-range.ts`
- `src/components/monitoring/use-prometheus-available.ts`
- `src/components/monitoring/metrics-time-range-selector.tsx`
- `src/components/monitoring/metrics-utils.ts`
- `src/components/monitoring/instance-resource-metrics.tsx`

### Modified files (4):
- `src/components/monitoring/cluster-node-resource-metrics.tsx`
- `src/components/monitoring/environment-resource-metrics.tsx`
- `src/pages/applications/application-detail-page.tsx`
- `src/pages/code-repositories/code-repository-detail-page.tsx`

### No API changes required.
