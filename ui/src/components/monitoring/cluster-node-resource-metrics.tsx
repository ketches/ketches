import { useQuery } from "@tanstack/react-query"
import { AlertCircle, ArrowDown, ArrowUp, Box, CircleSlash, Cpu, ExternalLink, HardDrive, Loader2, MemoryStick, Network, RefreshCcw, Server } from "lucide-react"
import { useNavigate } from "react-router-dom"
import { CartesianGrid, Line, LineChart, XAxis, YAxis } from "recharts"

import { clustersApi } from "@/api/clusters"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader } from "@/components/ui/card"
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from "@/components/ui/chart"
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia } from "@/components/ui/empty"
import { useTimeRange } from "./use-time-range"
import { usePrometheusAvailable } from "./use-prometheus-available"
import { MetricsTimeRangeSelector } from "./metrics-time-range-selector"
import { getUtilizationColor, getUtilizationColorClass } from "./metrics-utils"
import { EmptyState } from "@/components/shared/empty-state"

const usageChartConfig: ChartConfig = {
  cpu: { label: "CPU (mCores)", color: "var(--chart-1)" },
  memory: { label: "Memory (GiB)", color: "var(--chart-2)" },
  storage: { label: "Storage (GiB)", color: "var(--chart-3)" },
}

const utilChartConfig: ChartConfig = {
  cpuUtil: { label: "CPU Utilization (%)", color: "var(--chart-1)" },
  memUtil: { label: "Memory Utilization (%)", color: "var(--chart-2)" },
  storageUtil: { label: "Storage Utilization (%)", color: "var(--chart-3)" },
}

const netChartConfig: ChartConfig = {
  ingress: { label: "Ingress", color: "var(--chart-1)" },
  egress: { label: "Egress", color: "var(--chart-2)" },
}

const clusterChartConfig: ChartConfig = {
  pods: { label: "Running Pods", color: "var(--chart-4)" },
  restarts: { label: "Container Restarts", color: "var(--chart-5)" },
  nodeReady: { label: "Ready Nodes", color: "var(--chart-1)" },
}

interface ClusterNodeResourceMetricsProps {
  clusterId: string
  nodeName?: string
  nodeIp?: string
}

export function ClusterNodeResourceMetrics({ clusterId, nodeName, nodeIp }: ClusterNodeResourceMetricsProps) {
  const navigate = useNavigate()
  const { timeRange, setTimeRange, rangeSeconds, step: timeStep } = useTimeRange()
  const { available: prometheusAvailable, isLoading: prometheusLoading } = usePrometheusAvailable(clusterId)

  const { data: metrics, isLoading, error } = useQuery({
    queryKey: ["cluster-node-metrics-v5", clusterId, nodeName, nodeIp, timeRange],
    queryFn: async () => {
      const now = Math.floor(Date.now() / 1000)
      const start = now - rangeSeconds
      const step = timeStep

      const filter = nodeName ? `, node="${nodeName}"` : ""
      const nodeIpFilter = nodeIp ? `, instance="${nodeIp}:9100"` : ""

      const queries = {
        cpu: `sum(rate(container_cpu_usage_seconds_total{container!=""${filter}}[5m])) * 1000`,
        cpuUtil: `sum(rate(container_cpu_usage_seconds_total{container!=""${filter}}[5m])) / sum(kube_node_status_allocatable{resource="cpu"${filter}}) * 100`,
        memory: `sum(container_memory_working_set_bytes{container!=""${filter}}) / 1024 / 1024 / 1024`,
        memUtil: `sum(container_memory_working_set_bytes{container!=""${filter}}) / sum(kube_node_status_allocatable{resource="memory"${filter}}) * 100`,
        storage: `avg(node_filesystem_size_bytes{device=~"/dev/.*", container!=""${nodeIpFilter}}-node_filesystem_free_bytes{device=~"/dev/.*", container!=""${nodeIpFilter}}) / 1024 / 1024 / 1024`,
        storageUtil: `(1-avg(node_filesystem_free_bytes{device=~"/dev/.*", container!=""${nodeIpFilter}} / node_filesystem_size_bytes{device=~"/dev/.*", container!=""${nodeIpFilter}})) * 100`,
        ingress: `sum(rate(container_network_receive_bytes_total{${nodeName ? `node="${nodeName}"` : ""}}[5m])) / 1024`,
        egress: `sum(rate(container_network_transmit_bytes_total{${nodeName ? `node="${nodeName}"` : ""}}[5m])) / 1024`,
        pods: `sum(kubelet_running_pods{${nodeName ? `node="${nodeName}"` : ""}})`,
        restarts: `sum(increase(kube_pod_container_status_restarts_total{container!=""${filter}}[5m]))`,
        nodeReady: `sum(kube_node_status_condition{condition="Ready",status="true"})`,
      }

      const results = await Promise.all(
        Object.entries(queries).map(async ([key, query]) => {
          try {
            const res = await clustersApi.prometheusQueryRange(clusterId, query, start.toString(), now.toString(), step) as any
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
              timestamp: ts
            })
          }
          timeMap.get(ts)[key] = parseFloat(val) || 0
        })
      })

      const chartData = Array.from(timeMap.values()).sort((a, b) => a.timestamp - b.timestamp)
      const last = chartData[chartData.length - 1] || {}

      return {
        chartData,
        current: {
          cpu: last.cpu || 0,
          memory: last.memory || 0,
          storage: last.storage || 0,
          cpuUtil: last.cpuUtil || 0,
          memUtil: last.memUtil || 0,
          storageUtil: last.storageUtil || 0,
          ingress: last.ingress || 0,
          egress: last.egress || 0,
          pods: last.pods || 0,
          restarts: last.restarts || 0,
          nodeReady: last.nodeReady || 0,
        }
      }
    },
    refetchInterval: 60000,
    enabled: !!clusterId,
  })

  if (prometheusLoading) {
    return (
      <div className="min-h-[600px] flex items-center justify-center py-12">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (prometheusAvailable === false) {
    return (
      <div className="min-h-[600px]">
        <EmptyState
          title="Prometheus Not Available"
          description="This cluster does not have a Prometheus integration configured. Please contact your administrator to add Prometheus monitoring to this cluster."
          icon={AlertCircle}
        />
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="min-h-[600px] flex items-center justify-center py-12">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="min-h-[600px] flex flex-col items-center justify-center py-6 bg-destructive/5 rounded-md border border-destructive/20 border-dashed">
        <Empty>
          <EmptyHeader>
            <EmptyMedia variant="icon" className="bg-destructive/10 text-destructive">
              <AlertCircle />
            </EmptyMedia>
            <EmptyDescription>Prometheus connection error</EmptyDescription>
          </EmptyHeader>
          <EmptyContent>
            <Button
              variant="destructive"
              size="sm"
              onClick={() => navigate(`/clusters/${clusterId}?tab=configuration`)}
            >
              Check Cluster Configuration
              <ExternalLink className="ml-2 h-3 w-3" />
            </Button>
          </EmptyContent>
        </Empty>
      </div>
    )
  }

  if (!metrics || metrics.chartData.length === 0) {
    return (
      <div className="min-h-[600px] flex flex-col items-center justify-center py-8 text-center bg-muted/20 rounded-md border border-dashed">
        <Empty>
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <CircleSlash />
            </EmptyMedia>
            <EmptyDescription>
              No metrics data found for this {nodeName ? "node" : "cluster"}.
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      </div>
    )
  }

  return (
    <div className="space-y-4 min-h-[600px]">
      <div className="flex items-center justify-end">
        <MetricsTimeRangeSelector value={timeRange} onChange={setTimeRange} />
      </div>
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardDescription className="flex items-center justify-between">
              <span className="flex items-center gap-1">
                <Cpu className="h-3 w-3" />
                CPU Usage
              </span>
              <span className="font-mono text-xs">{metrics.current.cpu.toFixed(2)} mCores</span>
            </CardDescription>
          </CardHeader>
          <CardContent className="pb-2">
            <ChartContainer config={usageChartConfig} className="h-32 w-full">
              <LineChart data={metrics.chartData}>
                <CartesianGrid vertical={false} strokeDasharray="3 3" />
                <XAxis dataKey="time" tickLine={false} axisLine={false} tick={{ fontSize: 10 }} interval="preserveStartEnd" />
                <YAxis tickLine={false} axisLine={false} tick={{ fontSize: 10 }} width={40} domain={[0, 'auto']} />
                <ChartTooltip content={<ChartTooltipContent />} />
                <Line dataKey="cpu" type="monotone" stroke="var(--color-cpu)" strokeWidth={2} dot={false} />
              </LineChart>
            </ChartContainer>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardDescription className="flex items-center justify-between">
              <span className="flex items-center gap-1">
                <Cpu className="h-3 w-3" />
                CPU Utilization
              </span>
              <span className={`font-mono text-xs ${getUtilizationColorClass(metrics.current.cpuUtil)}`}>{metrics.current.cpuUtil.toFixed(1)}%</span>
            </CardDescription>
          </CardHeader>
          <CardContent className="pb-2">
            <ChartContainer config={utilChartConfig} className="h-32 w-full">
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
        <Card>
          <CardHeader className="pb-2">
            <CardDescription className="flex items-center justify-between">
              <span className="flex items-center gap-1">
                <MemoryStick className="h-3 w-3" />
                Memory Usage
              </span>
              <span className="font-mono text-xs">{metrics.current.memory.toFixed(2)} GiB</span>
            </CardDescription>
          </CardHeader>
          <CardContent className="pb-2">
            <ChartContainer config={usageChartConfig} className="h-32 w-full">
              <LineChart data={metrics.chartData}>
                <CartesianGrid vertical={false} strokeDasharray="3 3" />
                <XAxis dataKey="time" tickLine={false} axisLine={false} tick={{ fontSize: 10 }} interval="preserveStartEnd" />
                <YAxis tickLine={false} axisLine={false} tick={{ fontSize: 10 }} width={40} domain={[0, 'auto']} />
                <ChartTooltip content={<ChartTooltipContent />} />
                <Line dataKey="memory" type="monotone" stroke="var(--color-memory)" strokeWidth={2} dot={false} />
              </LineChart>
            </ChartContainer>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardDescription className="flex items-center justify-between">
              <span className="flex items-center gap-1">
                <MemoryStick className="h-3 w-3" />
                Memory Utilitization
              </span>
              <span className={`font-mono text-xs ${getUtilizationColorClass(metrics.current.memUtil)}`}>{metrics.current.memUtil.toFixed(1)}%</span>
            </CardDescription>
          </CardHeader>
          <CardContent className="pb-2">
            <ChartContainer config={utilChartConfig} className="h-32 w-full">
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
      <div className="grid gap-4 md:grid-cols-3 lg:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardDescription className="flex items-center justify-between">
              <span className="flex items-center gap-1">
                <HardDrive className="h-3 w-3" />
                Storage Usage
              </span>
              <span className="font-mono text-xs">{metrics.current.storage.toFixed(2)} GiB</span>
            </CardDescription>
          </CardHeader>
          <CardContent className="pb-2">
            <ChartContainer config={usageChartConfig} className="h-32 w-full">
              <LineChart data={metrics.chartData}>
                <CartesianGrid vertical={false} strokeDasharray="3 3" />
                <XAxis dataKey="time" tickLine={false} axisLine={false} tick={{ fontSize: 10 }} interval="preserveStartEnd" />
                <YAxis tickLine={false} axisLine={false} tick={{ fontSize: 10 }} width={40} domain={[0, 'auto']} />
                <ChartTooltip content={<ChartTooltipContent />} />
                <Line dataKey="storage" type="monotone" stroke="var(--color-storage)" strokeWidth={2} dot={false} />
              </LineChart>
            </ChartContainer>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardDescription className="flex items-center justify-between">
              <span className="flex items-center gap-1">
                <HardDrive className="h-3 w-3" />
                Storage Utilization
              </span>
              <span className={`font-mono text-xs ${getUtilizationColorClass(metrics.current.storageUtil)}`}>{metrics.current.storageUtil.toFixed(1)} %</span>
            </CardDescription>
          </CardHeader>
          <CardContent className="pb-2">
            <ChartContainer config={utilChartConfig} className="h-32 w-full">
              <LineChart data={metrics.chartData}>
                <CartesianGrid vertical={false} strokeDasharray="3 3" />
                <XAxis dataKey="time" tickLine={false} axisLine={false} tick={{ fontSize: 10 }} interval="preserveStartEnd" />
                <YAxis tickLine={false} axisLine={false} tick={{ fontSize: 10 }} width={40} domain={[0, 'auto']} />
                <ChartTooltip content={<ChartTooltipContent />} />
                <Line dataKey="storageUtil" type="monotone" stroke={getUtilizationColor(metrics.current.storageUtil)} strokeWidth={2} dot={false} />
              </LineChart>
            </ChartContainer>
          </CardContent>
        </Card>
        <Card className="md:col-span-2">
          <CardHeader className="pb-4">
            <CardDescription className="flex items-center justify-between">
              <span className="flex items-center gap-1">
                <Network className="h-3 w-3" />
                Network Traffic
              </span>
              <div className="flex gap-4 font-mono text-[10px]">
                <span className="flex items-center gap-1 text-primary"><ArrowDown className="h-2.5 w-2.5" />{metrics.current.ingress.toFixed(1)} KB/s</span>
                <span className="flex items-center gap-1 text-chart-2"><ArrowUp className="h-2.5 w-2.5" />{metrics.current.egress.toFixed(1)} KB/s</span>
              </div>
            </CardDescription>
          </CardHeader>
          <CardContent className="pb-2">
            <ChartContainer config={netChartConfig} className="h-32 w-full">
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
    </div>
  )
}
