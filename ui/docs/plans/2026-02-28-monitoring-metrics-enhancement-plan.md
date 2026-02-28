# Monitoring Metrics Enhancement — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Enrich Prometheus monitoring across cluster/environment/application/instance levels with new metrics, color-coded utilization, configurable time ranges, better empty states, and code repo stats.

**Architecture:** Hybrid utilities approach — extract `useTimeRange`, `usePrometheusAvailable`, `getUtilizationColor` as shared hooks/utils. Keep all chart JSX inline per component. New `InstanceResourceMetrics` dialog for per-pod metrics.

**Tech Stack:** React 19, TypeScript, recharts 2, TanStack Query 5, shadcn/ui, Tailwind CSS 4, lucide-react icons.

**Build/Verify:** `npm run build` (`tsc -b && vite build`) in `/home/dp/ketches/ui/`. No test framework — verify via `lsp_diagnostics` and build.

---

### Task 1: Create shared utility — `metrics-utils.ts`

**Files:**
- Create: `src/components/monitoring/metrics-utils.ts`

**Step 1: Create the utility file**

```typescript
// src/components/monitoring/metrics-utils.ts

/**
 * Returns a Tailwind text color class based on utilization percentage.
 * Blue: 0-70%, Orange: 70-85%, Red: 85%+
 */
export function getUtilizationColorClass(value: number): string {
  if (value >= 85) return "text-red-500"
  if (value >= 70) return "text-orange-500"
  return "text-blue-500"
}

/**
 * Returns a CSS color string for recharts stroke/fill based on utilization percentage.
 * Blue: 0-70%, Orange: 70-85%, Red: 85%+
 */
export function getUtilizationColor(value: number): string {
  if (value >= 85) return "hsl(0, 72%, 51%)"
  if (value >= 70) return "hsl(25, 95%, 53%)"
  return "hsl(217, 91%, 60%)"
}
```

**Step 2: Verify with lsp_diagnostics**

Run `lsp_diagnostics` on the new file. Expected: no errors.

**Step 3: Commit**

```bash
git add src/components/monitoring/metrics-utils.ts
git commit -m "feat(monitoring): add utilization color threshold utilities"
```

---

### Task 2: Create shared hook — `use-time-range.ts`

**Files:**
- Create: `src/components/monitoring/use-time-range.ts`

**Step 1: Create the hook**

```typescript
// src/components/monitoring/use-time-range.ts
import { useState } from "react"

export type TimeRange = "5m" | "15m" | "30m" | "1h" | "4h"

interface TimeRangeConfig {
  rangeSeconds: number
  step: string
  label: string
}

const TIME_RANGE_CONFIG: Record<TimeRange, TimeRangeConfig> = {
  "5m":  { rangeSeconds: 300,   step: "15",  label: "5 min" },
  "15m": { rangeSeconds: 900,   step: "30",  label: "15 min" },
  "30m": { rangeSeconds: 1800,  step: "60",  label: "30 min" },
  "1h":  { rangeSeconds: 3600,  step: "60",  label: "1 hour" },
  "4h":  { rangeSeconds: 14400, step: "120", label: "4 hours" },
}

export const TIME_RANGES: { value: TimeRange; label: string }[] = [
  { value: "5m", label: "5m" },
  { value: "15m", label: "15m" },
  { value: "30m", label: "30m" },
  { value: "1h", label: "1h" },
  { value: "4h", label: "4h" },
]

export function useTimeRange(defaultRange: TimeRange = "1h") {
  const [timeRange, setTimeRange] = useState<TimeRange>(defaultRange)
  const config = TIME_RANGE_CONFIG[timeRange]
  return {
    timeRange,
    setTimeRange,
    rangeSeconds: config.rangeSeconds,
    step: config.step,
    label: config.label,
  }
}
```

**Step 2: Verify with lsp_diagnostics**

Run `lsp_diagnostics` on the new file. Expected: no errors.

**Step 3: Commit**

```bash
git add src/components/monitoring/use-time-range.ts
git commit -m "feat(monitoring): add useTimeRange hook with configurable range/step"
```

---

### Task 3: Create shared hook — `use-prometheus-available.ts`

**Files:**
- Create: `src/components/monitoring/use-prometheus-available.ts`

**Step 1: Create the hook**

```typescript
// src/components/monitoring/use-prometheus-available.ts
import { useQuery } from "@tanstack/react-query"
import { clustersApi } from "@/api/clusters"

export function usePrometheusAvailable(clusterId: string) {
  const { data, isLoading } = useQuery({
    queryKey: ["cluster-prometheus-available", clusterId],
    queryFn: async () => {
      const integrations = await clustersApi.listIntegrations(clusterId)
      return integrations.some(i => i.integration_type === "prometheus")
    },
    enabled: !!clusterId,
    staleTime: 5 * 60 * 1000, // cache for 5 minutes
  })

  return {
    available: data,
    isLoading,
  }
}
```

**Step 2: Verify with lsp_diagnostics**

Run `lsp_diagnostics` on the new file. Expected: no errors.

**Step 3: Commit**

```bash
git add src/components/monitoring/use-prometheus-available.ts
git commit -m "feat(monitoring): add usePrometheusAvailable hook for integration check"
```

---

### Task 4: Create shared component — `metrics-time-range-selector.tsx`

**Files:**
- Create: `src/components/monitoring/metrics-time-range-selector.tsx`

**Step 1: Create the component**

```tsx
// src/components/monitoring/metrics-time-range-selector.tsx
import { Button } from "@/components/ui/button"
import { type TimeRange, TIME_RANGES } from "./use-time-range"

interface MetricsTimeRangeSelectorProps {
  value: TimeRange
  onChange: (range: TimeRange) => void
}

export function MetricsTimeRangeSelector({ value, onChange }: MetricsTimeRangeSelectorProps) {
  return (
    <div className="flex items-center gap-1">
      {TIME_RANGES.map((range) => (
        <Button
          key={range.value}
          variant={value === range.value ? "default" : "outline"}
          size="sm"
          className="h-7 px-2.5 text-xs"
          onClick={() => onChange(range.value)}
        >
          {range.label}
        </Button>
      ))}
    </div>
  )
}
```

**Step 2: Verify with lsp_diagnostics**

Run `lsp_diagnostics` on the new file. Expected: no errors.

**Step 3: Commit**

```bash
git add src/components/monitoring/metrics-time-range-selector.tsx
git commit -m "feat(monitoring): add MetricsTimeRangeSelector component"
```

---

### Task 5: Update cluster metrics — enrich, time range, color, empty states

**Files:**
- Modify: `src/components/monitoring/cluster-node-resource-metrics.tsx`

**Context:** This file currently has 320 lines. It renders cluster-level metrics: CPU/Memory/Storage usage + utilization + network traffic. All hardcoded to 1-hour range with 60s step.

**Step 1: Add imports for new shared utilities**

Add these imports at the top:
```typescript
import { useTimeRange } from "./use-time-range"
import { usePrometheusAvailable } from "./use-prometheus-available"
import { MetricsTimeRangeSelector } from "./metrics-time-range-selector"
import { getUtilizationColor, getUtilizationColorClass } from "./metrics-utils"
import { EmptyState } from "@/components/shared/empty-state"
```

Add `BarChart3, Box, RefreshCcw, Server` to the lucide-react import (for new chart icons).

**Step 2: Add hooks inside the component function**

After `const navigate = useNavigate()`, add:
```typescript
const { timeRange, setTimeRange, rangeSeconds, step: timeStep } = useTimeRange()
const { available: prometheusAvailable, isLoading: prometheusLoading } = usePrometheusAvailable(clusterId)
```

**Step 3: Add Prometheus availability check**

Before the `isLoading` check, add:
```typescript
if (prometheusLoading) {
  return (
    <div className="flex items-center justify-center py-12">
      <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
    </div>
  )
}

if (prometheusAvailable === false) {
  return (
    <EmptyState
      title="Prometheus Not Available"
      description="This cluster does not have a Prometheus integration configured. Please contact your administrator to add Prometheus monitoring to this cluster."
      icon={AlertCircle}
    />
  )
}
```

**Step 4: Replace hardcoded time range in queryFn**

Replace:
```typescript
const now = Math.floor(Date.now() / 1000)
const oneHourAgo = now - 3600
const step = "60"
```

With:
```typescript
const now = Math.floor(Date.now() / 1000)
const start = now - rangeSeconds
const step = timeStep
```

And update all `oneHourAgo` references to `start` in the `prometheusQueryRange` calls.

Add `rangeSeconds, timeStep` to the queryKey:
```typescript
queryKey: ["cluster-node-metrics-v5", clusterId, nodeName, nodeIp, timeRange],
```

**Step 5: Add new PromQL queries**

Add to the `queries` object:
```typescript
pods: `sum(kubelet_running_pods{${nodeName ? `node="${nodeName}"` : ""}})`,
restarts: `sum(increase(kube_pod_container_status_restarts_total{container!=""${filter}}[5m]))`,
nodeReady: `sum(kube_node_status_condition{condition="Ready",status="true"})`,
```

Add to `current` object:
```typescript
pods: last.pods || 0,
restarts: last.restarts || 0,
nodeReady: last.nodeReady || 0,
```

**Step 6: Add chart configs for new metrics**

```typescript
const clusterChartConfig: ChartConfig = {
  pods: { label: "Running Pods", color: "var(--chart-4)" },
  restarts: { label: "Container Restarts", color: "var(--chart-5)" },
  nodeReady: { label: "Ready Nodes", color: "var(--chart-1)" },
}
```

**Step 7: Add MetricsTimeRangeSelector to the render**

Wrap the chart grid in a container with the time range selector at top:
```tsx
<div className="space-y-4">
  <div className="flex items-center justify-between">
    <span className="text-sm text-muted-foreground">Resource Metrics</span>
    <MetricsTimeRangeSelector value={timeRange} onChange={setTimeRange} />
  </div>
  {/* existing chart grids */}
</div>
```

**Step 8: Color-code utilization charts**

For each utilization chart card header, change the current value span to use dynamic color:
```tsx
<span className={`font-mono text-xs ${getUtilizationColorClass(metrics.current.cpuUtil)}`}>
  {metrics.current.cpuUtil.toFixed(1)}%
</span>
```

For each utilization `<Line>`, change stroke to dynamic:
```tsx
<Line dataKey="cpuUtil" type="monotone" stroke={getUtilizationColor(metrics.current.cpuUtil)} strokeWidth={2} dot={false} />
```

Apply this pattern to `cpuUtil`, `memUtil`, and `storageUtil`.

**Step 9: Add new chart cards for pods, restarts, node readiness**

Add a third row below the existing network traffic row:
```tsx
<div className="grid gap-4 md:grid-cols-3">
  {/* Pod Count Card */}
  <Card>
    <CardHeader className="pb-2">
      <CardDescription className="flex items-center justify-between">
        <span className="flex items-center gap-1">
          <Box className="h-3 w-3" />
          Running Pods
        </span>
        <span className="font-mono text-xs">{metrics.current.pods.toFixed(0)}</span>
      </CardDescription>
    </CardHeader>
    <CardContent className="pb-2">
      <ChartContainer config={clusterChartConfig} className="h-32 w-full">
        <LineChart data={metrics.chartData}>
          <CartesianGrid vertical={false} strokeDasharray="3 3" />
          <XAxis dataKey="time" tickLine={false} axisLine={false} tick={{ fontSize: 10 }} interval="preserveStartEnd" />
          <YAxis tickLine={false} axisLine={false} tick={{ fontSize: 10 }} width={30} domain={[0, 'auto']} />
          <ChartTooltip content={<ChartTooltipContent />} />
          <Line dataKey="pods" type="monotone" stroke="var(--color-pods)" strokeWidth={2} dot={false} />
        </LineChart>
      </ChartContainer>
    </CardContent>
  </Card>

  {/* Container Restarts Card */}
  <Card>
    <CardHeader className="pb-2">
      <CardDescription className="flex items-center justify-between">
        <span className="flex items-center gap-1">
          <RefreshCcw className="h-3 w-3" />
          Container Restarts
        </span>
        <span className="font-mono text-xs">{metrics.current.restarts.toFixed(0)}/5m</span>
      </CardDescription>
    </CardHeader>
    <CardContent className="pb-2">
      <ChartContainer config={clusterChartConfig} className="h-32 w-full">
        <LineChart data={metrics.chartData}>
          <CartesianGrid vertical={false} strokeDasharray="3 3" />
          <XAxis dataKey="time" tickLine={false} axisLine={false} tick={{ fontSize: 10 }} interval="preserveStartEnd" />
          <YAxis tickLine={false} axisLine={false} tick={{ fontSize: 10 }} width={30} domain={[0, 'auto']} />
          <ChartTooltip content={<ChartTooltipContent />} />
          <Line dataKey="restarts" type="monotone" stroke="var(--color-restarts)" strokeWidth={2} dot={false} />
        </LineChart>
      </ChartContainer>
    </CardContent>
  </Card>

  {/* Node Readiness Card */}
  <Card>
    <CardHeader className="pb-2">
      <CardDescription className="flex items-center justify-between">
        <span className="flex items-center gap-1">
          <Server className="h-3 w-3" />
          Ready Nodes
        </span>
        <span className="font-mono text-xs">{metrics.current.nodeReady.toFixed(0)}</span>
      </CardDescription>
    </CardHeader>
    <CardContent className="pb-2">
      <ChartContainer config={clusterChartConfig} className="h-32 w-full">
        <LineChart data={metrics.chartData}>
          <CartesianGrid vertical={false} strokeDasharray="3 3" />
          <XAxis dataKey="time" tickLine={false} axisLine={false} tick={{ fontSize: 10 }} interval="preserveStartEnd" />
          <YAxis tickLine={false} axisLine={false} tick={{ fontSize: 10 }} width={30} domain={[0, 'auto']} />
          <ChartTooltip content={<ChartTooltipContent />} />
          <Line dataKey="nodeReady" type="monotone" stroke="var(--color-nodeReady)" strokeWidth={2} dot={false} />
        </LineChart>
      </ChartContainer>
    </CardContent>
  </Card>
</div>
```

**Step 10: Verify and commit**

Run `lsp_diagnostics` on `cluster-node-resource-metrics.tsx`. Expected: no errors.
Run `npm run build`. Expected: exit code 0.

```bash
git add src/components/monitoring/cluster-node-resource-metrics.tsx
git commit -m "feat(monitoring): enrich cluster metrics with pods, restarts, node readiness + time range + color thresholds + empty states"
```

---

### Task 6: Update environment metrics — enrich, time range, color, empty states

**Files:**
- Modify: `src/components/monitoring/environment-resource-metrics.tsx`

**Context:** Currently 219 lines. Renders CPU usage, memory usage, network traffic for a namespace. No utilization charts.

**Step 1: Add imports for shared utilities**

```typescript
import { useTimeRange } from "./use-time-range"
import { usePrometheusAvailable } from "./use-prometheus-available"
import { MetricsTimeRangeSelector } from "./metrics-time-range-selector"
import { getUtilizationColor, getUtilizationColorClass } from "./metrics-utils"
```

Add `Box` to the lucide-react import for pod count icon.

**Step 2: Add hooks**

After `const navigate = useNavigate()`:
```typescript
const { timeRange, setTimeRange, rangeSeconds, step: timeStep } = useTimeRange()
const { available: prometheusAvailable, isLoading: prometheusLoading } = usePrometheusAvailable(clusterId)
```

**Step 3: Add Prometheus availability check**

Same pattern as cluster. Before existing `isLoading` check:
```typescript
if (prometheusLoading) {
  return (
    <div className="flex items-center justify-center py-12">
      <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
    </div>
  )
}

if (prometheusAvailable === false) {
  return (
    <EmptyState
      title="Prometheus Not Available"
      description="This cluster does not have a Prometheus integration configured. Please contact your administrator to add Prometheus monitoring to this environment's cluster."
      icon={AlertCircle}
    />
  )
}
```

**Step 4: Replace hardcoded time range**

Same as cluster — replace `oneHourAgo = now - 3600` with `start = now - rangeSeconds`, use `timeStep` for step, update queryKey to include `timeRange`.

**Step 5: Add new PromQL queries**

Add to the `queries` object:
```typescript
cpuUtil: `sum(rate(container_cpu_usage_seconds_total{namespace="${namespace}"}[5m])) / sum(kube_resourcequota{namespace="${namespace}",resource="requests.cpu"}) * 100`,
memUtil: `sum(container_memory_working_set_bytes{namespace="${namespace}"}) / sum(kube_resourcequota{namespace="${namespace}",resource="requests.memory"}) * 100`,
pods: `count(kube_pod_info{namespace="${namespace}"})`,
```

Add to `current`:
```typescript
cpuUtil: last.cpuUtil || 0,
memUtil: last.memUtil || 0,
pods: last.pods || 0,
```

**Step 6: Add chart configs**

```typescript
const utilChartConfig: ChartConfig = {
  cpuUtil: { label: "CPU Utilization (%)", color: "var(--chart-1)" },
  memUtil: { label: "Memory Utilization (%)", color: "var(--chart-2)" },
}

const podChartConfig: ChartConfig = {
  pods: { label: "Running Pods", color: "var(--chart-4)" },
}
```

**Step 7: Expand layout to 2 rows**

Change the return JSX to have:
- A time range selector header
- Row 1: CPU Usage, CPU Utilization, Memory Usage (3 cards)
- Row 2: Memory Utilization, Pod Count, Network Traffic (3 cards)

Utilization cards use color-coded line + badge, same pattern as cluster.

**Step 8: Verify and commit**

Run `lsp_diagnostics`. Run `npm run build`. Expected: clean.

```bash
git add src/components/monitoring/environment-resource-metrics.tsx
git commit -m "feat(monitoring): enrich environment metrics with utilization, pod count + time range + color thresholds + empty states"
```

---

### Task 7: Create instance metrics dialog

**Files:**
- Create: `src/components/monitoring/instance-resource-metrics.tsx`

**Context:** New file. Dialog showing per-pod metrics. Follows the exact same chart patterns as existing monitoring components.

**Step 1: Create the full component**

This component receives `{ open, onOpenChange, clusterId, namespace, podName, app }` props and renders a `Dialog` with `DialogContent` at 90vw × 90vh.

Inside: `useQuery` fetching CPU usage, memory usage, CPU utilization, memory utilization, network ingress/egress for a single pod. Uses `useTimeRange` for configurable range.

Charts layout:
- Header: Dialog title + `MetricsTimeRangeSelector`
- 2x2 grid: CPU Usage, Memory Usage, CPU Utilization, Memory Utilization
- Full-width: Network Traffic

Key details:
- PromQL queries filter by `pod="{podName}"` (exact match)
- CPU/Memory charts show `request` and `limit` reference lines from `app` props
- Utilization charts use color-coded lines via `getUtilizationColor`
- Loading state: Loader2 spinner inside the dialog
- Error state: destructive Empty component
- No data: EmptyState

Full implementation:
```tsx
import { useQuery } from "@tanstack/react-query"
import {
  AlertCircle,
  ArrowDown,
  ArrowUp,
  CircleSlash,
  Cpu,
  ExternalLink,
  Loader2,
  MemoryStick,
  Network,
} from "lucide-react"
import { CartesianGrid, Line, LineChart, ReferenceLine, XAxis, YAxis } from "recharts"

import { clustersApi } from "@/api/clusters"
import { Card, CardContent, CardDescription, CardHeader } from "@/components/ui/card"
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from "@/components/ui/chart"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia } from "@/components/ui/empty"

import { MetricsTimeRangeSelector } from "./metrics-time-range-selector"
import { getUtilizationColor, getUtilizationColorClass } from "./metrics-utils"
import { useTimeRange } from "./use-time-range"

const cpuChartConfig: ChartConfig = {
  cpu: { label: "CPU (mCores)", color: "var(--chart-1)" },
}
const memChartConfig: ChartConfig = {
  memory: { label: "Memory (GiB)", color: "var(--chart-2)" },
}
const utilChartConfig: ChartConfig = {
  cpuUtil: { label: "CPU Utilization (%)", color: "var(--chart-1)" },
  memUtil: { label: "Memory Utilization (%)", color: "var(--chart-2)" },
}
const netChartConfig: ChartConfig = {
  ingress: { label: "Ingress", color: "var(--chart-1)" },
  egress: { label: "Egress", color: "var(--chart-2)" },
}

interface InstanceResourceMetricsProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  clusterId: string
  namespace: string
  podName: string
  app: any
}

export function InstanceResourceMetrics({
  open,
  onOpenChange,
  clusterId,
  namespace,
  podName,
  app,
}: InstanceResourceMetricsProps) {
  const { timeRange, setTimeRange, rangeSeconds, step: timeStep } = useTimeRange()

  const { data: metrics, isLoading, error } = useQuery({
    queryKey: ["instance-metrics", clusterId, namespace, podName, timeRange],
    queryFn: async () => {
      const now = Math.floor(Date.now() / 1000)
      const start = now - rangeSeconds
      const step = timeStep

      const queries = {
        cpu: `sum(rate(container_cpu_usage_seconds_total{namespace="${namespace}", pod="${podName}", container!=""}[5m])) * 1000`,
        memory: `sum(container_memory_working_set_bytes{namespace="${namespace}", pod="${podName}", container!=""}) / 1024 / 1024 / 1024`,
        ingress: `sum(rate(container_network_receive_bytes_total{namespace="${namespace}", pod="${podName}"}[5m])) / 1024`,
        egress: `sum(rate(container_network_transmit_bytes_total{namespace="${namespace}", pod="${podName}"}[5m])) / 1024`,
      }

      const results = await Promise.all(
        Object.entries(queries).map(async ([key, query]) => {
          try {
            const res = await clustersApi.prometheusQueryRange(
              clusterId, query, start.toString(), now.toString(), step
            ) as any
            return { key, values: res?.result?.[0]?.values || [] }
          } catch {
            return { key, values: [] }
          }
        })
      )

      const timeMap = new Map<number, any>()
      results.forEach(({ key, values }) => {
        values.forEach(([ts, val]: [number, string]) => {
          if (!timeMap.has(ts)) {
            timeMap.set(ts, {
              time: new Date(ts * 1000).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }),
              timestamp: ts,
            })
          }
          timeMap.get(ts)[key] = parseFloat(val) || 0
        })
      })

      // Compute utilization
      const cpuLimit = app?.limit_cpu || 0
      const memLimit = (app?.limit_memory || 0) / 1024

      timeMap.forEach((val) => {
        val.cpuRequest = app?.request_cpu || 0
        val.cpuLimit = cpuLimit
        val.memRequest = (app?.request_memory || 0) / 1024
        val.memLimit = memLimit
        val.cpuUtil = cpuLimit > 0 ? (val.cpu / cpuLimit) * 100 : 0
        val.memUtil = memLimit > 0 ? (val.memory / memLimit) * 100 : 0
      })

      const chartData = Array.from(timeMap.values()).sort((a, b) => a.timestamp - b.timestamp)
      const last = chartData[chartData.length - 1] || {}

      return {
        chartData,
        current: {
          cpu: last.cpu || 0,
          memory: last.memory || 0,
          cpuUtil: last.cpuUtil || 0,
          memUtil: last.memUtil || 0,
          ingress: last.ingress || 0,
          egress: last.egress || 0,
        },
        limits: { cpuLimit, memLimit, cpuRequest: app?.request_cpu || 0, memRequest: (app?.request_memory || 0) / 1024 },
      }
    },
    refetchInterval: 30000,
    enabled: open && !!clusterId && !!namespace && !!podName,
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[90vw] max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <div className="flex items-center justify-between">
            <DialogTitle>Instance Metrics — {podName}</DialogTitle>
            <MetricsTimeRangeSelector value={timeRange} onChange={setTimeRange} />
          </div>
        </DialogHeader>

        {isLoading ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
        ) : error ? (
          <div className="flex flex-col items-center justify-center py-6 bg-destructive/5 rounded-md border border-destructive/20 border-dashed">
            <Empty>
              <EmptyHeader>
                <EmptyMedia variant="icon" className="bg-destructive/10 text-destructive">
                  <AlertCircle />
                </EmptyMedia>
                <EmptyDescription>Failed to load metrics for this instance.</EmptyDescription>
              </EmptyHeader>
            </Empty>
          </div>
        ) : !metrics || metrics.chartData.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-8 text-center bg-muted/20 rounded-md border border-dashed">
            <Empty>
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <CircleSlash />
                </EmptyMedia>
                <EmptyDescription>No metrics data found for this instance.</EmptyDescription>
              </EmptyHeader>
            </Empty>
          </div>
        ) : (
          <div className="space-y-4">
            {/* Row 1: CPU Usage, Memory Usage */}
            <div className="grid gap-4 md:grid-cols-2">
              {/* CPU Usage */}
              <Card>
                <CardHeader className="pb-2">
                  <CardDescription className="flex items-center justify-between">
                    <span className="flex items-center gap-1"><Cpu className="h-3 w-3" />CPU Usage</span>
                    <span className="font-mono text-xs">{metrics.current.cpu.toFixed(2)} mCores</span>
                  </CardDescription>
                </CardHeader>
                <CardContent className="pb-2">
                  <ChartContainer config={cpuChartConfig} className="h-40 w-full">
                    <LineChart data={metrics.chartData}>
                      <CartesianGrid vertical={false} strokeDasharray="3 3" />
                      <XAxis dataKey="time" tickLine={false} axisLine={false} tick={{ fontSize: 10 }} interval="preserveStartEnd" />
                      <YAxis tickLine={false} axisLine={false} tick={{ fontSize: 10 }} width={40} domain={[0, 'auto']} />
                      <ChartTooltip content={<ChartTooltipContent />} />
                      <Line dataKey="cpu" type="monotone" stroke="var(--color-cpu)" strokeWidth={2} dot={false} />
                      {metrics.limits.cpuRequest > 0 && (
                        <ReferenceLine y={metrics.limits.cpuRequest} stroke="var(--chart-3)" strokeDasharray="4 4" label={{ value: "Request", position: "right", fontSize: 10 }} />
                      )}
                      {metrics.limits.cpuLimit > 0 && (
                        <ReferenceLine y={metrics.limits.cpuLimit} stroke="var(--chart-5)" strokeDasharray="4 4" label={{ value: "Limit", position: "right", fontSize: 10 }} />
                      )}
                    </LineChart>
                  </ChartContainer>
                </CardContent>
              </Card>

              {/* Memory Usage */}
              <Card>
                <CardHeader className="pb-2">
                  <CardDescription className="flex items-center justify-between">
                    <span className="flex items-center gap-1"><MemoryStick className="h-3 w-3" />Memory Usage</span>
                    <span className="font-mono text-xs">{metrics.current.memory.toFixed(2)} GiB</span>
                  </CardDescription>
                </CardHeader>
                <CardContent className="pb-2">
                  <ChartContainer config={memChartConfig} className="h-40 w-full">
                    <LineChart data={metrics.chartData}>
                      <CartesianGrid vertical={false} strokeDasharray="3 3" />
                      <XAxis dataKey="time" tickLine={false} axisLine={false} tick={{ fontSize: 10 }} interval="preserveStartEnd" />
                      <YAxis tickLine={false} axisLine={false} tick={{ fontSize: 10 }} width={40} domain={[0, 'auto']} />
                      <ChartTooltip content={<ChartTooltipContent />} />
                      <Line dataKey="memory" type="monotone" stroke="var(--color-memory)" strokeWidth={2} dot={false} />
                      {metrics.limits.memRequest > 0 && (
                        <ReferenceLine y={metrics.limits.memRequest} stroke="var(--chart-3)" strokeDasharray="4 4" label={{ value: "Request", position: "right", fontSize: 10 }} />
                      )}
                      {metrics.limits.memLimit > 0 && (
                        <ReferenceLine y={metrics.limits.memLimit} stroke="var(--chart-5)" strokeDasharray="4 4" label={{ value: "Limit", position: "right", fontSize: 10 }} />
                      )}
                    </LineChart>
                  </ChartContainer>
                </CardContent>
              </Card>
            </div>

            {/* Row 2: CPU Utilization, Memory Utilization */}
            <div className="grid gap-4 md:grid-cols-2">
              {/* CPU Utilization */}
              <Card>
                <CardHeader className="pb-2">
                  <CardDescription className="flex items-center justify-between">
                    <span className="flex items-center gap-1"><Cpu className="h-3 w-3" />CPU Utilization</span>
                    <span className={`font-mono text-xs ${getUtilizationColorClass(metrics.current.cpuUtil)}`}>{metrics.current.cpuUtil.toFixed(1)}%</span>
                  </CardDescription>
                </CardHeader>
                <CardContent className="pb-2">
                  <ChartContainer config={utilChartConfig} className="h-40 w-full">
                    <LineChart data={metrics.chartData}>
                      <CartesianGrid vertical={false} strokeDasharray="3 3" />
                      <XAxis dataKey="time" tickLine={false} axisLine={false} tick={{ fontSize: 10 }} interval="preserveStartEnd" />
                      <YAxis tickLine={false} axisLine={false} tick={{ fontSize: 10 }} width={30} domain={[0, 'auto']} />
                      <ChartTooltip content={<ChartTooltipContent hideLabel />} />
                      <Line dataKey="cpuUtil" type="monotone" stroke={getUtilizationColor(metrics.current.cpuUtil)} strokeWidth={2} dot={false} />
                    </LineChart>
                  </ChartContainer>
                </CardContent>
              </Card>

              {/* Memory Utilization */}
              <Card>
                <CardHeader className="pb-2">
                  <CardDescription className="flex items-center justify-between">
                    <span className="flex items-center gap-1"><MemoryStick className="h-3 w-3" />Memory Utilization</span>
                    <span className={`font-mono text-xs ${getUtilizationColorClass(metrics.current.memUtil)}`}>{metrics.current.memUtil.toFixed(1)}%</span>
                  </CardDescription>
                </CardHeader>
                <CardContent className="pb-2">
                  <ChartContainer config={utilChartConfig} className="h-40 w-full">
                    <LineChart data={metrics.chartData}>
                      <CartesianGrid vertical={false} strokeDasharray="3 3" />
                      <XAxis dataKey="time" tickLine={false} axisLine={false} tick={{ fontSize: 10 }} interval="preserveStartEnd" />
                      <YAxis tickLine={false} axisLine={false} tick={{ fontSize: 10 }} width={30} domain={[0, 'auto']} />
                      <ChartTooltip content={<ChartTooltipContent hideLabel />} />
                      <Line dataKey="memUtil" type="monotone" stroke={getUtilizationColor(metrics.current.memUtil)} strokeWidth={2} dot={false} />
                    </LineChart>
                  </ChartContainer>
                </CardContent>
              </Card>
            </div>

            {/* Row 3: Network Traffic - Full width */}
            <Card>
              <CardHeader className="pb-4">
                <CardDescription className="flex items-center justify-between">
                  <span className="flex items-center gap-1"><Network className="h-3 w-3" />Network Traffic</span>
                  <div className="flex gap-4 font-mono text-[10px]">
                    <span className="flex items-center gap-1 text-primary"><ArrowDown className="h-2.5 w-2.5" />{metrics.current.ingress.toFixed(1)} KB/s</span>
                    <span className="flex items-center gap-1 text-chart-2"><ArrowUp className="h-2.5 w-2.5" />{metrics.current.egress.toFixed(1)} KB/s</span>
                  </div>
                </CardDescription>
              </CardHeader>
              <CardContent className="pb-2">
                <ChartContainer config={netChartConfig} className="h-40 w-full">
                  <LineChart data={metrics.chartData}>
                    <CartesianGrid vertical={false} strokeDasharray="3 3" />
                    <XAxis dataKey="time" tickLine={false} axisLine={false} tick={{ fontSize: 10 }} interval="preserveStartEnd" />
                    <YAxis tickLine={false} axisLine={false} tick={{ fontSize: 10 }} width={40} domain={[0, 'auto']} />
                    <ChartTooltip content={<ChartTooltipContent />} />
                    <Line dataKey="ingress" name="Ingress" type="monotone" stroke="var(--color-ingress)" strokeWidth={2} dot={false} />
                    <Line dataKey="egress" name="Egress" type="monotone" stroke="var(--color-egress)" strokeWidth={2} dot={false} />
                  </LineChart>
                </ChartContainer>
              </CardContent>
            </Card>
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}
```

**Step 2: Verify with lsp_diagnostics**

Run `lsp_diagnostics` on the new file. Expected: no errors.

**Step 3: Commit**

```bash
git add src/components/monitoring/instance-resource-metrics.tsx
git commit -m "feat(monitoring): add InstanceResourceMetrics dialog with per-pod charts"
```

---

### Task 8: Add instance metrics button to application detail page

**Files:**
- Modify: `src/pages/applications/application-detail-page.tsx`

**Context:** 1525 lines. Instance actions are in `instanceColumns` definition (~line 675-773). The `AppMetrics` function is at line 175.

**Step 1: Add imports**

Add at the top:
```typescript
import { InstanceResourceMetrics } from "@/components/monitoring/instance-resource-metrics"
import { BarChart3 } from "lucide-react"
```

Also add imports for the time range and prometheus hooks for the `AppMetrics` section:
```typescript
import { useTimeRange } from "@/components/monitoring/use-time-range"
import { usePrometheusAvailable } from "@/components/monitoring/use-prometheus-available"
import { MetricsTimeRangeSelector } from "@/components/monitoring/metrics-time-range-selector"
import { getUtilizationColor, getUtilizationColorClass } from "@/components/monitoring/metrics-utils"
import { EmptyState } from "@/components/shared/empty-state"
```

**Step 2: Add state for instance metrics dialog**

In the main component function, add state:
```typescript
const [metricsInstance, setMetricsInstance] = React.useState<string | null>(null)
```

**Step 3: Add Metrics button to instance actions**

In the `instanceColumns` `actions` cell (around line 684), add a new button before the delete button:
```tsx
<Button
  variant="ghost"
  size="icon-sm"
  onClick={(e) => {
    e.stopPropagation()
    setMetricsInstance(instance.instanceName)
  }}
  title="View Metrics"
>
  <BarChart3 />
</Button>
```

**Step 4: Render the InstanceResourceMetrics dialog**

In the JSX return, add the dialog component (near the bottom, alongside other dialogs):
```tsx
<InstanceResourceMetrics
  open={!!metricsInstance}
  onOpenChange={(open) => { if (!open) setMetricsInstance(null) }}
  clusterId={app?.cluster_id || ""}
  namespace={app?.namespace || ""}
  podName={metricsInstance || ""}
  app={app}
/>
```

**Step 5: Update AppMetrics function**

In the `AppMetrics` function:
1. Add `useTimeRange()` hook
2. Add `usePrometheusAvailable(clusterId)` hook
3. Replace hardcoded `oneHourAgo = now - 3600` / `step = "60"` with hook values
4. Add `MetricsTimeRangeSelector` above the chart grid
5. Replace `return null` for no-data with `<EmptyState title="No Metrics Data" description="..." icon={CircleSlash} />`
6. Add Prometheus availability check before data loading
7. Color-code any utilization values (the app already computes `cpuUtil_${pod}` and `memUtil_${pod}`)

**Step 6: Verify and commit**

Run `lsp_diagnostics` on `application-detail-page.tsx`. Run `npm run build`. Expected: clean.

```bash
git add src/pages/applications/application-detail-page.tsx
git commit -m "feat(monitoring): add instance metrics dialog + time range + empty states to application page"
```

---

### Task 9: Add code repository stats card

**Files:**
- Modify: `src/pages/code-repositories/code-repository-detail-page.tsx`

**Context:** 856 lines. Overview tab at line 342. `builds`, `buildConfigs`, `deployments` already fetched as arrays via `useQuery`.

**Step 1: Add imports**

```typescript
import { BarChart3, CheckCircle } from "lucide-react"
```

(These may already be partially imported. Check existing imports first.)

**Step 2: Compute stats**

Before the return statement, add:
```typescript
const totalBuilds = builds.length
const successfulBuilds = builds.filter(b => b.status === "succeeded").length
const successRate = totalBuilds > 0 ? (successfulBuilds / totalBuilds) * 100 : 0
const totalDeployments = deployments.length
const totalBuildConfigs = buildConfigs.length

const getSuccessRateColor = (rate: number) => {
  if (rate >= 90) return "text-green-500"
  if (rate >= 70) return "text-orange-500"
  return "text-red-500"
}
```

**Step 3: Add stats card to Overview tab**

In `<TabsContent value="overview">`, after the existing "Repository Information" card (line ~362), add:

```tsx
<Card>
  <CardHeader>
    <CardTitle className="text-sm flex items-center gap-2">
      <BarChart3 className="h-4 w-4" />
      Repository Statistics
    </CardTitle>
    <CardDescription>
      Build and deployment activity for this repository.
    </CardDescription>
  </CardHeader>
  <CardContent>
    <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
      <div className="space-y-1">
        <p className="text-xs font-medium text-muted-foreground flex items-center gap-1">
          <Hammer className="h-3 w-3" />
          Build Configs
        </p>
        <p className="text-2xl font-bold">{totalBuildConfigs}</p>
      </div>
      <div className="space-y-1">
        <p className="text-xs font-medium text-muted-foreground flex items-center gap-1">
          <Play className="h-3 w-3" />
          Total Builds
        </p>
        <p className="text-2xl font-bold">{totalBuilds}</p>
      </div>
      <div className="space-y-1">
        <p className="text-xs font-medium text-muted-foreground flex items-center gap-1">
          <CheckCircle className="h-3 w-3" />
          Success Rate
        </p>
        <p className={`text-2xl font-bold ${totalBuilds > 0 ? getSuccessRateColor(successRate) : "text-muted-foreground"}`}>
          {totalBuilds > 0 ? `${successRate.toFixed(0)}%` : "N/A"}
        </p>
      </div>
      <div className="space-y-1">
        <p className="text-xs font-medium text-muted-foreground flex items-center gap-1">
          <Rocket className="h-3 w-3" />
          Deployments
        </p>
        <p className="text-2xl font-bold">{totalDeployments}</p>
      </div>
    </div>
  </CardContent>
</Card>
```

**Step 4: Verify and commit**

Run `lsp_diagnostics` on `code-repository-detail-page.tsx`. Run `npm run build`. Expected: clean.

```bash
git add src/pages/code-repositories/code-repository-detail-page.tsx
git commit -m "feat(code-repos): add repository statistics card with build/deploy metrics"
```

---

### Task 10: Final verification

**Step 1: Full build**

```bash
cd /home/dp/ketches/ui && npm run build
```

Expected: exit code 0.

**Step 2: Lint check**

```bash
cd /home/dp/ketches/ui && npm run lint
```

Expected: no new errors (pre-existing ones are OK).

**Step 3: Review all changed files**

Run `lsp_diagnostics` on all 9 files (5 new + 4 modified).

**Step 4: Final commit if any fixes needed**

If build or lint revealed issues, fix and commit with `fix: resolve build/lint issues`.
