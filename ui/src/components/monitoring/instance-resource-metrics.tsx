import { useQuery } from "@tanstack/react-query"
import {
  AlertCircle,
  ArrowDown,
  ArrowUp,
  CircleSlash,
  Cpu,
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
  DialogDescription,
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
      <DialogContent className="sm:max-w-[90vw] w-full min-h-[70vh] flex flex-col">
        <DialogHeader>
          <DialogTitle>Instance Metrics</DialogTitle>
          <DialogDescription className="flex items-center justify-between">
            <span>{podName}</span>
            <MetricsTimeRangeSelector value={timeRange} onChange={setTimeRange} />
          </DialogDescription>
        </DialogHeader>

        {isLoading ? (
          <div className="flex items-center justify-center min-h-125">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
        ) : error ? (
          <div className="flex flex-col items-center justify-center min-h-125 bg-destructive/5 rounded-md border border-destructive/20 border-dashed">
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
          <div className="flex flex-col items-center justify-center min-h-125 text-center bg-muted/20 rounded-md border border-dashed">
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
            <div className="grid gap-4 md:grid-cols-2">
              <Card>
                <CardHeader className="pb-4">
                  <CardDescription className="flex items-center justify-between">
                    <span className="flex items-center gap-1"><Network className="h-3 w-3" />Ingress Traffic</span>
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
                    </LineChart>
                  </ChartContainer>
                </CardContent>
              </Card>
              <Card>
                <CardHeader className="pb-4">
                  <CardDescription className="flex items-center justify-between">
                    <span className="flex items-center gap-1"><Network className="h-3 w-3" />Egress Traffic</span>
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
                      <Line dataKey="egress" name="Egress" type="monotone" stroke="var(--color-egress)" strokeWidth={2} dot={false} />
                    </LineChart>
                  </ChartContainer>
                </CardContent>
              </Card>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog >
  )
}
