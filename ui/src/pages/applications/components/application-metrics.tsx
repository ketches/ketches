import { clustersApi } from "@/api/clusters"
import { type App } from "@/api/apps"
import { EmptyState } from "@/components/shared/empty-state"
import { Card, CardContent, CardHeader, CardDescription } from "@/components/ui/card"
import { ChartContainer, ChartTooltip, ChartTooltipContent } from "@/components/ui/chart"
import { Skeleton } from "@/components/ui/skeleton"
import { type TimeRange } from "@/components/monitoring/use-time-range"
import { Activity, ArrowDown, ArrowUp, ChartLine, Cpu, MemoryStick, Network } from "lucide-react"
import { useQuery } from "@tanstack/react-query"
import { CartesianGrid, Line, LineChart, XAxis, YAxis } from "recharts"

interface MetricDataPoint {
  timestamp: number
  time: string
  cpuRequest?: number
  cpuLimit?: number
  memRequest?: number
  memLimit?: number
  [key: string]: number | string | undefined
}

function getMetricValue(point: MetricDataPoint | undefined, key: string): number {
  const value = point?.[key]
  return typeof value === "number" ? value : 0
}

interface ApplicationMetricsProps {
  clusterId: string
  projectId?: string
  prometheusAvailable?: boolean
  namespace: string
  appSlug: string
  app: App
  timeRange: TimeRange
  rangeSeconds: number
  step: string
}

export function ApplicationMetrics({
  clusterId,
  projectId,
  prometheusAvailable,
  namespace,
  appSlug,
  app,
  timeRange,
  rangeSeconds,
  step,
}: ApplicationMetricsProps) {
  const prometheusLoading = false

  const { data: metricsData, isLoading } = useQuery({
    queryKey: ["app-metrics-v6", clusterId, namespace, appSlug, timeRange],
    queryFn: async () => {
      const now = Math.floor(Date.now() / 1000)
      const start = now - rangeSeconds

      const queries = {
        cpu: `sum(rate(container_cpu_usage_seconds_total{namespace="${namespace}", pod=~"${appSlug}-.*", container!=""}[5m])) by (pod) * 1000`,
        mem: `sum(container_memory_working_set_bytes{namespace="${namespace}", pod=~"${appSlug}-.*", container!=""}) by (pod) / 1024 / 1024 / 1024`,
        ingress: `sum(rate(container_network_receive_bytes_total{namespace="${namespace}", pod=~"${appSlug}-.*"}[5m])) by (pod) / 1024`,
        egress: `sum(rate(container_network_transmit_bytes_total{namespace="${namespace}", pod=~"${appSlug}-.*"}[5m])) by (pod) / 1024`,
      }

      const results = await Promise.all(
        Object.entries(queries).map(async ([key, query]) => {
          try {
            const response = await clustersApi.prometheusQueryRange(
              clusterId,
              query,
              start.toString(),
              now.toString(),
              step,
              projectId
            )
            return { key, results: response?.result || [] }
          } catch {
            return { key, results: [] }
          }
        })
      )

      const timeMap = new Map<number, MetricDataPoint>()
      const podNames = new Set<string>()

      results.forEach(({ key, results: queryResults }) => {
        queryResults.forEach((result) => {
          const pod = result.metric.pod
          podNames.add(pod)
          result.values?.forEach(([timestamp, value]: [number, string]) => {
            if (!timeMap.has(timestamp)) {
              timeMap.set(timestamp, {
                timestamp,
                time: new Date(timestamp * 1000).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }),
              })
            }
            const point = timeMap.get(timestamp)
            if (point) {
              point[`${key}_${pod}`] = parseFloat(value) || 0
            }
          })
        })
      })

      timeMap.forEach((value) => {
        value.cpuRequest = app.request_cpu || 0
        value.cpuLimit = app.limit_cpu || 0
        value.memRequest = (app.request_memory || 0) / 1024
        value.memLimit = (app.limit_memory || 0) / 1024

        podNames.forEach((pod) => {
          const cpu = getMetricValue(value, `cpu_${pod}`)
          const mem = getMetricValue(value, `mem_${pod}`)
          const cpuLimit = value.cpuLimit || 0
          const memLimit = value.memLimit || 0
          value[`cpuUtil_${pod}`] = cpuLimit > 0 ? (cpu / cpuLimit) * 100 : 0
          value[`memUtil_${pod}`] = memLimit > 0 ? (mem / memLimit) * 100 : 0
        })
      })

      return {
        chartData: Array.from(timeMap.values()).sort((a, b) => a.timestamp - b.timestamp),
        pods: Array.from(podNames),
      }
    },
    refetchInterval: 30000,
    enabled: !!clusterId && !!namespace && !!appSlug && !!app,
  })

  if (prometheusLoading || isLoading) {
    return (
      <div className="min-h-125 flex items-center justify-center">
        <Skeleton className="h-64 w-full" />
      </div>
    )
  }

  if (prometheusAvailable === false) {
    return (
      <EmptyState
        title="Prometheus Not Available"
        description="The cluster does not have a Prometheus integration configured. Please contact your administrator to enable Prometheus monitoring."
        icon={Activity}
      />
    )
  }

  if (!metricsData || metricsData.chartData.length === 0) {
    return (
      <EmptyState
        title="No Metrics Data"
        description="No monitoring data is available for this application. Metrics will appear once the application is running and producing data."
        icon={Activity}
      />
    )
  }

  const { chartData, pods } = metricsData
  const lastPoint = chartData.at(-1)

  const totalCpu = pods.reduce((sum, pod) => sum + getMetricValue(lastPoint, `cpu_${pod}`), 0)
  const totalMem = pods.reduce((sum, pod) => sum + getMetricValue(lastPoint, `mem_${pod}`), 0)
  const totalIngress = pods.reduce((sum, pod) => sum + getMetricValue(lastPoint, `ingress_${pod}`), 0)
  const totalEgress = pods.reduce((sum, pod) => sum + getMetricValue(lastPoint, `egress_${pod}`), 0)

  const maxCpu = Math.max(...chartData.flatMap((point) => pods.map((pod) => getMetricValue(point, `cpu_${pod}`))))
  const maxMem = Math.max(...chartData.flatMap((point) => pods.map((pod) => getMetricValue(point, `mem_${pod}`))))
  const maxNet = Math.max(...chartData.flatMap((point) => pods.map((pod) => Math.max(getMetricValue(point, `ingress_${pod}`), getMetricValue(point, `egress_${pod}`)))))

  return (
    <div className="space-y-4 min-h-125">
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardHeader>
            <CardDescription className="flex items-center justify-between">
              <span className="flex items-center gap-1"><Cpu className="h-3 w-3" />CPU Usage</span>
              <span className="font-mono text-xs text-muted-foreground">{totalCpu.toFixed(2)} mCores (Total)</span>
            </CardDescription>
          </CardHeader>
          <CardContent className="pb-2">
            <ChartContainer config={{}} className="h-40 w-full">
              <LineChart data={chartData}>
                <CartesianGrid vertical={false} strokeDasharray="3 3" />
                <XAxis dataKey="time" tickLine={false} axisLine={false} tick={{ fontSize: 10 }} interval="preserveStartEnd" />
                <YAxis tickLine={false} axisLine={false} tick={{ fontSize: 10 }} width={40} domain={[0, () => Math.max(maxCpu * 1.2, (app.request_cpu || 0) * 1.1)]} />
                <ChartTooltip content={<ChartTooltipContent />} />
                {pods.map((pod, index) => <Line key={pod} name={pod} dataKey={`cpu_${pod}`} type="monotone" stroke={`var(--chart-${(index % 5) + 1})`} strokeWidth={2} dot={false} />)}
                <Line name="Request" dataKey="cpuRequest" type="stepAfter" stroke="#94a3b8" strokeWidth={1.5} strokeDasharray="4 4" dot={false} />
                <Line name="Limit" dataKey="cpuLimit" type="stepAfter" stroke="#ef4444" strokeWidth={1.5} strokeDasharray="2 2" dot={false} />
              </LineChart>
            </ChartContainer>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardDescription className="flex items-center justify-between">
              <span className="flex items-center gap-1"><Cpu className="h-3 w-3" />CPU Utilization</span>
            </CardDescription>
          </CardHeader>
          <CardContent className="pb-2">
            <ChartContainer config={{}} className="h-40 w-full">
              <LineChart data={chartData}>
                <CartesianGrid vertical={false} strokeDasharray="3 3" />
                <XAxis dataKey="time" tickLine={false} axisLine={false} tick={{ fontSize: 10 }} interval="preserveStartEnd" />
                <YAxis tickLine={false} axisLine={false} tick={{ fontSize: 10 }} width={30} domain={[0, "auto"]} />
                <ChartTooltip content={<ChartTooltipContent hideLabel />} />
                {pods.map((pod, index) => <Line key={pod} name={pod} dataKey={`cpuUtil_${pod}`} type="monotone" stroke={`var(--chart-${(index % 5) + 1})`} strokeWidth={2} dot={false} />)}
              </LineChart>
            </ChartContainer>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardDescription className="flex items-center justify-between">
              <span className="flex items-center gap-1"><MemoryStick className="h-3 w-3" />Memory Usage</span>
              <span className="font-mono text-xs text-muted-foreground">{totalMem.toFixed(2)} GiB (Total)</span>
            </CardDescription>
          </CardHeader>
          <CardContent className="pb-2">
            <ChartContainer config={{}} className="h-40 w-full">
              <LineChart data={chartData}>
                <CartesianGrid vertical={false} strokeDasharray="3 3" />
                <XAxis dataKey="time" tickLine={false} axisLine={false} tick={{ fontSize: 10 }} interval="preserveStartEnd" />
                <YAxis tickLine={false} axisLine={false} tick={{ fontSize: 10 }} width={40} domain={[0, () => Math.max(maxMem * 1.2, (app.request_memory / 1024 || 0) * 1.1)]} />
                <ChartTooltip content={<ChartTooltipContent />} />
                {pods.map((pod, index) => <Line key={pod} name={pod} dataKey={`mem_${pod}`} type="monotone" stroke={`var(--chart-${(index % 5) + 1})`} strokeWidth={2} dot={false} />)}
                <Line name="Request" dataKey="memRequest" type="stepAfter" stroke="#94a3b8" strokeWidth={1.5} strokeDasharray="4 4" dot={false} />
                <Line name="Limit" dataKey="memLimit" type="stepAfter" stroke="#ef4444" strokeWidth={1.5} strokeDasharray="2 2" dot={false} />
              </LineChart>
            </ChartContainer>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardDescription className="flex items-center justify-between">
              <span className="flex items-center gap-1"><ChartLine className="h-3 w-3" />Memory Utilization</span>
            </CardDescription>
          </CardHeader>
          <CardContent className="pb-2">
            <ChartContainer config={{}} className="h-40 w-full">
              <LineChart data={chartData}>
                <CartesianGrid vertical={false} strokeDasharray="3 3" />
                <XAxis dataKey="time" tickLine={false} axisLine={false} tick={{ fontSize: 10 }} interval="preserveStartEnd" />
                <YAxis tickLine={false} axisLine={false} tick={{ fontSize: 10 }} width={30} domain={[0, "auto"]} />
                <ChartTooltip content={<ChartTooltipContent hideLabel />} />
                {pods.map((pod, index) => <Line key={pod} name={pod} dataKey={`memUtil_${pod}`} type="monotone" stroke={`var(--chart-${(index % 5) + 1})`} strokeWidth={2} dot={false} />)}
              </LineChart>
            </ChartContainer>
          </CardContent>
        </Card>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Card>
          <CardHeader>
            <CardDescription className="flex items-center justify-between">
              <span className="flex items-center gap-1"><Network className="h-3 w-3" />Network Ingress</span>
              <span className="text-primary flex items-center gap-0.5"><ArrowDown className="h-2 w-2" />{totalIngress.toFixed(1)} KB/s</span>
            </CardDescription>
          </CardHeader>
          <CardContent className="pb-2">
            <ChartContainer config={{}} className="h-40 w-full">
              <LineChart data={chartData}>
                <CartesianGrid vertical={false} strokeDasharray="3 3" />
                <XAxis dataKey="time" tickLine={false} axisLine={false} tick={{ fontSize: 10 }} interval="preserveStartEnd" />
                <YAxis tickLine={false} axisLine={false} tick={{ fontSize: 10 }} width={40} domain={[0, () => maxNet * 1.2]} />
                <ChartTooltip content={<ChartTooltipContent />} />
                {pods.map((pod, index) => <Line key={`${pod}-in`} name={`${pod} In`} dataKey={`ingress_${pod}`} type="monotone" stroke={`var(--chart-${(index % 5) + 1})`} strokeWidth={2} dot={false} />)}
              </LineChart>
            </ChartContainer>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardDescription className="flex items-center justify-between">
              <span className="flex items-center gap-1"><Network className="h-3 w-3" />Network Egress</span>
              <span className="text-chart-2 flex items-center gap-0.5"><ArrowUp className="h-2 w-2" />{totalEgress.toFixed(1)} KB/s</span>
            </CardDescription>
          </CardHeader>
          <CardContent className="pb-2">
            <ChartContainer config={{}} className="h-40 w-full">
              <LineChart data={chartData}>
                <CartesianGrid vertical={false} strokeDasharray="3 3" />
                <XAxis dataKey="time" tickLine={false} axisLine={false} tick={{ fontSize: 10 }} interval="preserveStartEnd" />
                <YAxis tickLine={false} axisLine={false} tick={{ fontSize: 10 }} width={40} domain={[0, () => maxNet * 1.2]} />
                <ChartTooltip content={<ChartTooltipContent />} />
                {pods.map((pod, index) => <Line key={`${pod}-out`} name={`${pod} Out`} dataKey={`egress_${pod}`} type="monotone" stroke={`var(--chart-${(index % 5) + 1})`} strokeWidth={2} strokeDasharray={2} dot={false} />)}
              </LineChart>
            </ChartContainer>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
