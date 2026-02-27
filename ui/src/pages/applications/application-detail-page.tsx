import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef } from "@tanstack/react-table"
import {
  Activity,
  ArrowDown,
  ArrowUp,
  Box,
  ChartLine,
  ChevronsUpDown,
  Clock,
  CloudCog,
  Cog,
  Copy,
  Cpu,
  Edit2,
  ExternalLink,
  FileCog,
  FileText,
  FolderGit2,
  FolderOpen,
  Hammer,
  HardDrive,
  Info,
  Key,
  Layers,
  Layers2,
  LayoutGrid,
  List,
  Loader2,
  MemoryStick,
  MoveVertical,
  Network,
  Orbit,
  Pencil,
  Puzzle,
  RotateCw,
  Ruler,
  Search,
  Server,
  Shapes,
  Share2,
  Telescope,
  Terminal as TerminalIcon,
  Trash2,
  Zap
} from "lucide-react"
import * as React from "react"
import { Link, useNavigate, useParams, useSearchParams } from "react-router-dom"
import { CartesianGrid, Line, LineChart, XAxis, YAxis } from "recharts"
import { toast } from "sonner"

import { appsApi, type App } from "@/api/apps"
import { clustersApi } from "@/api/clusters"
import { envsApi } from "@/api/envs"
import { AppActionButtons } from "@/components/applications/app-action-buttons"
import { TopologyView } from "@/components/applications/app-topology-view"
import { AutoScalingConfig } from "@/components/applications/auto-scaling-config"
import { CommandConfig } from "@/components/applications/command-config"
import { ConfigFilesTable } from "@/components/applications/config-files-table"
import { EditAppDialog } from "@/components/applications/edit-app-dialog"
import { EnvVarTable } from "@/components/applications/env-var-table"
import { NetworkConfig } from "@/components/applications/gateway-table"
import { HealthConfig } from "@/components/applications/health-config"
import { ImageEditor } from "@/components/applications/image-editor"
import { ResourceConfig } from "@/components/applications/resource-config"
import { SchedulingConfig } from "@/components/applications/scheduling-config"
import { VolumesTable } from "@/components/applications/volumes-table"
import { BuildList } from "@/components/builds/build-list"
import { UnifiedBuildDeployDialog } from "@/components/code-repositories/unified-build-deploy-dialog"
import { DataTable } from "@/components/data-table/data-table"
import { DeploymentHistoryList } from "@/components/deployments/deployment-history-list"
import { NotFoundPage } from "@/components/layout/not-found-page"
import { PageHeader } from "@/components/layout/page-header"
import { AppPlugins } from "@/components/plugins/app-plugins"
import { ColorBadge } from "@/components/shared/color-badge"
import { EmptyState } from "@/components/shared/empty-state"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { ChartContainer, ChartTooltip, ChartTooltipContent } from "@/components/ui/chart"
import { Checkbox } from "@/components/ui/checkbox"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { DropdownMenu, DropdownMenuContent, DropdownMenuGroup, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useBottomPanel } from "@/contexts/bottom-panel-context"
import { getAppStatusColor } from "@/lib/app-status"
import { formatDate } from "@/lib/utils"
import { useProjectStore } from "@/stores/project"
import { useProjectRole } from "@/hooks/useProjectRole"

function ScaleAppPopover({ app }: { app: App }) {
  const [replicas, setReplicas] = React.useState(app.replicas)
  const [open, setOpen] = React.useState(false)
  const queryClient = useQueryClient()

  React.useEffect(() => {
    setReplicas(app.replicas)
  }, [app.replicas])

  const scaleMutation = useMutation({
    mutationFn: (count: number) => appsApi.update(app.id, { ...app, replicas: count }),
    onSuccess: () => {
      toast.success(`Application scaling to ${replicas} replicas initiated`)
      queryClient.invalidateQueries({ queryKey: ['app', app.id] })
      setOpen(false)
    },
    onError: (error: any) => {
      toast.error("Failed to scale application", {
        description: error.response?.data?.error || error.message
      })
    }
  })

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger>
        <Button>
          <MoveVertical />
          Scale: {app.replicas}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-80">
        <div className="space-y-4">
          <div className="space-y-2">
            <h4 className="font-medium text-sm">Scale Application</h4>
            <p className="text-xs text-muted-foreground">
              Change the number of desired replicas.
            </p>
          </div>
          <div className="flex items-center gap-4">
            <Button
              variant="outline"
              size="icon"
              onClick={() => setReplicas(Math.max(0, replicas - 1))}
              disabled={replicas <= 0}
            >
              -
            </Button>
            <Input
              type="number"
              value={replicas}
              onChange={(e) => setReplicas(parseInt(e.target.value) || 0)}
              className="text-center font-bold text-lg"
            />
            <Button
              variant="outline"
              size="icon"
              onClick={() => setReplicas(replicas + 1)}
            >
              +
            </Button>
          </div>
          {app.auto_scaling && (
            <p className="text-[10px] text-destructive font-medium">
              Warning: AutoScaling is enabled. Manual scaling might be overridden by the autoscaler.
            </p>
          )}
          <div className="flex gap-2">
            <Button variant="outline" size="sm" onClick={() => setOpen(false)} className="flex-1">Cancel</Button>
            <Button size="sm" onClick={() => scaleMutation.mutate(replicas)} disabled={scaleMutation.isPending} className="flex-1">
              {scaleMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Scale
            </Button>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  )
}

function AppMetrics({ clusterId, namespace, appSlug, app }: { clusterId: string, namespace: string, appSlug: string, app: any }) {
  const { data: metricsData, isLoading } = useQuery({
    queryKey: ['app-metrics-v6', clusterId, namespace, appSlug],
    queryFn: async () => {
      const now = Math.floor(Date.now() / 1000)
      const oneHourAgo = now - 3600
      const step = "60"

      const queries = {
        cpu: `sum(rate(container_cpu_usage_seconds_total{namespace="${namespace}", pod=~"${appSlug}-.*", container!=""}[5m])) by (pod) * 1000`,
        mem: `sum(container_memory_working_set_bytes{namespace="${namespace}", pod=~"${appSlug}-.*", container!=""}) by (pod) / 1024 / 1024 / 1024`,
        ingress: `sum(rate(container_network_receive_bytes_total{namespace="${namespace}", pod=~"${appSlug}-.*"}[5m])) by (pod) / 1024`,
        egress: `sum(rate(container_network_transmit_bytes_total{namespace="${namespace}", pod=~"${appSlug}-.*"}[5m])) by (pod) / 1024`,
      }

      const results = await Promise.all(
        Object.entries(queries).map(async ([key, query]) => {
          try {
            const res = await clustersApi.prometheusQueryRange(clusterId, query, oneHourAgo.toString(), now.toString(), step) as any
            return { key, results: res?.result || [] }
          } catch {
            return { key, results: [] }
          }
        })
      )

      const timeMap = new Map<number, any>()
      const podNames = new Set<string>()

      results.forEach(({ key, results: queryResults }) => {
        queryResults.forEach((r: any) => {
          const pod = r.metric.pod
          podNames.add(pod)
          r.values.forEach(([ts, val]: [number, string]) => {
            if (!timeMap.has(ts)) {
              timeMap.set(ts, {
                timestamp: ts,
                time: new Date(ts * 1000).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
              })
            }
            timeMap.get(ts)[`${key}_${pod}`] = parseFloat(val) || 0
          })
        })
      })

      timeMap.forEach((val) => {
        val.cpuRequest = app.request_cpu || 0
        val.cpuLimit = app.limit_cpu || 0
        val.memRequest = (app.request_memory || 0) / 1024
        val.memLimit = (app.limit_memory || 0) / 1024

        podNames.forEach(pod => {
          const cpu = val[`cpu_${pod}`] || 0
          const mem = val[`mem_${pod}`] || 0
          val[`cpuUtil_${pod}`] = val.cpuLimit > 0 ? (cpu / val.cpuLimit) * 100 : 0
          val[`memUtil_${pod}`] = val.memLimit > 0 ? (mem / val.memLimit) * 100 : 0
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

  if (isLoading) return <Skeleton className="h-64 w-full mt-6" />
  if (!metricsData || metricsData.chartData.length === 0) return null

  const { chartData, pods } = metricsData
  const lastPoint = chartData[chartData.length - 1]

  const totalCpu = pods.reduce((s, p) => s + (lastPoint[`cpu_${p}`] || 0), 0)
  const totalMem = pods.reduce((s, p) => s + (lastPoint[`mem_${p}`] || 0), 0)
  const totalIngress = pods.reduce((s, p) => s + (lastPoint[`ingress_${p}`] || 0), 0)
  const totalEgress = pods.reduce((s, p) => s + (lastPoint[`egress_${p}`] || 0), 0)

  const maxCpu = Math.max(...chartData.flatMap(d => pods.map(p => d[`cpu_${p}`] || 0)))
  const maxMem = Math.max(...chartData.flatMap(d => pods.map(p => d[`mem_${p}`] || 0)))
  const maxNet = Math.max(...chartData.flatMap(d => pods.map(p => Math.max(d[`ingress_${p}`] || 0, d[`egress_${p}`] || 0))))

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardHeader className="pb-2">
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
                {pods.map((p, i) => <Line key={p} name={p} dataKey={`cpu_${p}`} type="monotone" stroke={`var(--chart-${(i % 5) + 1})`} strokeWidth={2} dot={false} />)}
                <Line name="Request" dataKey="cpuRequest" type="stepAfter" stroke="#94a3b8" strokeWidth={1.5} strokeDasharray="4 4" dot={false} />
                <Line name="Limit" dataKey="cpuLimit" type="stepAfter" stroke="#ef4444" strokeWidth={1.5} strokeDasharray="2 2" dot={false} />
              </LineChart>
            </ChartContainer>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardDescription className="flex items-center justify-between">
              <span className="flex items-center gap-1"><Cpu className="h-3 w-3" />CPU Utilization</span>
            </CardDescription>
          </CardHeader>
          <CardContent className="pb-2">
            <ChartContainer config={{}} className="h-40 w-full">
              <LineChart data={chartData}>
                <CartesianGrid vertical={false} strokeDasharray="3 3" />
                <XAxis dataKey="time" tickLine={false} axisLine={false} tick={{ fontSize: 10 }} interval="preserveStartEnd" />
                <YAxis tickLine={false} axisLine={false} tick={{ fontSize: 10 }} width={30} domain={[0, 'auto']} />
                <ChartTooltip content={<ChartTooltipContent hideLabel />} />
                {pods.map((p, i) => <Line key={p} name={p} dataKey={`cpuUtil_${p}`} type="monotone" stroke={`var(--chart-${(i % 5) + 1})`} strokeWidth={2} dot={false} />)}
              </LineChart>
            </ChartContainer>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
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
                {pods.map((p, i) => <Line key={p} name={p} dataKey={`mem_${p}`} type="monotone" stroke={`var(--chart-${(i % 5) + 1})`} strokeWidth={2} dot={false} />)}
                <Line name="Request" dataKey="memRequest" type="stepAfter" stroke="#94a3b8" strokeWidth={1.5} strokeDasharray="4 4" dot={false} />
                <Line name="Limit" dataKey="memLimit" type="stepAfter" stroke="#ef4444" strokeWidth={1.5} strokeDasharray="2 2" dot={false} />
              </LineChart>
            </ChartContainer>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardDescription className="flex items-center justify-between">
              <span className="flex items-center gap-1"><ChartLine className="h-3 w-3" />Memory Utilization</span>
            </CardDescription>
          </CardHeader>
          <CardContent className="pb-2">
            <ChartContainer config={{}} className="h-40 w-full">
              <LineChart data={chartData}>
                <CartesianGrid vertical={false} strokeDasharray="3 3" />
                <XAxis dataKey="time" tickLine={false} axisLine={false} tick={{ fontSize: 10 }} interval="preserveStartEnd" />
                <YAxis tickLine={false} axisLine={false} tick={{ fontSize: 10 }} width={30} domain={[0, 'auto']} />
                <ChartTooltip content={<ChartTooltipContent hideLabel />} />
                {pods.map((p, i) => <Line key={p} name={p} dataKey={`memUtil_${p}`} type="monotone" stroke={`var(--chart-${(i % 5) + 1})`} strokeWidth={2} dot={false} />)}
              </LineChart>
            </ChartContainer>
          </CardContent>
        </Card>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Card>
          <CardHeader className="pb-2">
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
                {pods.flatMap((p, i) => [
                  <Line key={`${p}-in`} name={`${p} In`} dataKey={`ingress_${p}`} type="monotone" stroke={`var(--chart-${(i % 5) + 1})`} strokeWidth={2} dot={false} />
                ])}
              </LineChart>
            </ChartContainer>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
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
                {pods.flatMap((p, i) => [
                  <Line key={`${p}-out`} name={`${p} Out`} dataKey={`egress_${p}`} type="monotone" stroke={`var(--chart-${(i % 5) + 1})`} strokeWidth={2} strokeDasharray={2} dot={false} />
                ])}
              </LineChart>
            </ChartContainer>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

function InstanceEventsDialog({ appId, instanceName, open, onOpenChange }: { appId: string, instanceName: string, open: boolean, onOpenChange: (open: boolean) => void }) {
  const { data: events = [], isLoading } = useQuery({
    queryKey: ['instance-events', appId, instanceName],
    queryFn: () => appsApi.listInstanceEvents(appId, instanceName),
    enabled: !!appId && !!instanceName && open,
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-180 max-h-[80vh] flex flex-col">
        <DialogHeader>
          <DialogTitle>Instance Events: {instanceName}</DialogTitle>
          <DialogDescription>
            Recent events related to this instance, such as scaling actions, health status changes, and lifecycle events.
          </DialogDescription>
        </DialogHeader>
        <div className="flex-1 overflow-auto py-4">
          <DataTable
            columns={[
              {
                accessorKey: "type",
                header: "Type",
                cell: ({ row }) => (
                  <ColorBadge color={row.original.type === "Normal" ? "blue" : "red"}>
                    {row.original.type}
                  </ColorBadge>
                )
              },
              { accessorKey: "reason", header: "Reason" },
              {
                accessorKey: "message",
                header: "Message",
                cell: ({ row }) => (
                  <div className="text-xs break-all whitespace-normal">
                    {row.original.message}
                  </div>
                )
              },
              {
                accessorKey: "count",
                header: "Count",
                cell: ({ row }) => <span className="text-xs font-mono">{row.original.count}</span>
              },
              {
                accessorKey: "createdAt",
                header: "Last Seen",
                cell: ({ row }) => <span className="text-xs text-muted-foreground whitespace-nowrap">{formatDate(row.original.createdAt)}</span>
              }
            ]}
            data={events}
            hidePagination
          />
          {isLoading && (
            <div className="flex items-center justify-center h-32">
              <Skeleton className="" />
            </div>
          )}
          {!isLoading && events.length === 0 && (
            <EmptyState
              title="No Events"
              description="No events found for this instance."
              icon={Activity}
            />
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}

const INSTANCES_VIEW_MODE_KEY = "instances_view_mode"

export function ApplicationDetailPage() {
  const navigate = useNavigate()
  const { appId } = useParams()
  const [searchParams, setSearchParams] = useSearchParams()
  const queryClient = useQueryClient()

  const currentTab = searchParams.get("tab") || "overview"
  const { openPanel } = useBottomPanel()
  const { activeProjectId, setActiveEnvId } = useProjectStore()
  const projectRole = useProjectRole()
  const isViewer = projectRole === 'viewer'

  const { data: app, isLoading, error } = useQuery<App>({
    queryKey: ['app', appId],
    queryFn: () => appsApi.get(appId!),
    enabled: !!appId,
    refetchInterval: 5000,
  })

  const { data: availableActions } = useQuery({
    queryKey: ['app-actions', appId],
    queryFn: () => appsApi.getAvailableActions(appId!),
    enabled: !!appId,
  })

  const { data: currentEnv } = useQuery({
    queryKey: ['env', app?.env_id],
    queryFn: () => envsApi.get(app!.env_id!),
    enabled: !!app?.env_id,
  })

  const projectIdToUse = activeProjectId || currentEnv?.project_id

  const { data: envs = [] } = useQuery({
    queryKey: ['envs-simple', projectIdToUse],
    queryFn: () => envsApi.listSimpleByProject(projectIdToUse!),
    enabled: !!projectIdToUse,
  })

  const safeEnvs = Array.isArray(envs) ? envs : []

  const { data: apps = [] } = useQuery({
    queryKey: ['apps-simple', app?.env_id],
    queryFn: () => appsApi.listSimple(app!.env_id!),
    enabled: !!app?.env_id,
  })

  const safeApps = Array.isArray(apps) ? apps : []

  const [viewMode, setViewMode] = React.useState<'table' | 'card'>(() => {
    const saved = localStorage.getItem(INSTANCES_VIEW_MODE_KEY)
    return (saved === 'table' || saved === 'card') ? saved : 'table'
  })
  const [searchQuery, setSearchQuery] = React.useState("")
  const [rowSelection, setRowSelection] = React.useState({})
  const [isEditImageDialogOpen, setIsEditImageDialogOpen] = React.useState(false)
  const [isEditAppDialogOpen, setIsEditAppDialogOpen] = React.useState(false)
  const [isUnifiedBuildDialogOpen, setIsUnifiedBuildDialogOpen] = React.useState(false)
  const [selectedInstanceForEvents, setSelectedInstanceForEvents] = React.useState<string | null>(null)
  const [deleteInstanceDialogOpen, setDeleteInstanceDialogOpen] = React.useState(false)
  const [deletingInstanceName, setDeletingInstanceName] = React.useState<string | null>(null)
  const [bulkDeleteDialogOpen, setBulkDeleteDialogOpen] = React.useState(false)
  const [selectedInstanceNames, setSelectedInstanceNames] = React.useState<string[]>([])

  React.useEffect(() => {
    localStorage.setItem(INSTANCES_VIEW_MODE_KEY, viewMode)
  }, [viewMode])

  const deleteInstanceMutation = useMutation({
    mutationFn: (instanceName: string) => appsApi.deleteInstance(appId!, instanceName),
    onSuccess: () => {
      toast.success("Instance deletion initiated")
      queryClient.invalidateQueries({ queryKey: ['app-instances', appId] })
    },
    onError: (error: any) => {
      toast.error("Failed to delete instance", {
        description: error.response?.data?.error || error.message
      })
    }
  })

  const { data: instances = [] } = useQuery({
    queryKey: ['app-instances', appId],
    queryFn: () => appsApi.listInstances(appId!),
    enabled: !!appId,
    refetchInterval: 5000,
  })

  const safeInstances = Array.isArray(instances) ? instances : []

  const filteredInstances = React.useMemo(() => {
    if (!searchQuery) return safeInstances
    const lowQuery = searchQuery.toLowerCase()
    return safeInstances.filter(i =>
      i.instanceName.toLowerCase().includes(lowQuery) ||
      i.ip?.toLowerCase().includes(lowQuery) ||
      i.nodeName?.toLowerCase().includes(lowQuery)
    )
  }, [safeInstances, searchQuery])

  const instanceColumns: ColumnDef<any>[] = [
    {
      id: "select",
      header: ({ table }) => (
        <Checkbox
          checked={table.getIsAllPageRowsSelected()}
          onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
          aria-label="Select all"
        />
      ),
      cell: ({ row }) => (
        <Checkbox
          checked={row.getIsSelected()}
          onCheckedChange={(value) => row.toggleSelected(!!value)}
          aria-label="Select row"
        />
      ),
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: "instanceName",
      header: "Instance",
      cell: ({ row }) => (
        <div className="flex flex-col">
          <span className="font-mono text-xs font-medium">{row.original.instanceName}</span>
          {row.original.ip && (
            <span className="font-mono text-[10px] text-muted-foreground">{row.original.ip}</span>
          )}
        </div>
      ),
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => {
        const status = row.original.status
        const isRunning = status === "Running"
        return (
          <ColorBadge color={isRunning ? "green" : "gray"}>
            {status}
          </ColorBadge>
        )
      },
    },
    {
      id: "containers",
      header: "Containers",
      cell: ({ row }) => {
        const initCount = row.original.initContainerCount
        const containerCount = row.original.containerCount
        return (
          <div className="flex items-center gap-3">
            {initCount > 0 && (
              <div className="flex items-center gap-1.5 text-muted-foreground" title="Init Containers">
                <Zap className="h-3.5 w-3.5" />
                <span>{initCount}</span>
              </div>
            )}
            <div className="flex items-center gap-1.5 text-muted-foreground" title="Containers">
              <Layers2 className="h-3.5 w-3.5" />
              <span>{containerCount}</span>
            </div>
          </div>
        )
      },
    },
    {
      accessorKey: "restartCount",
      header: "Restarts",
      cell: ({ row }) => (
        <span className={`text-xs font-mono ${row.original.restartCount > 0 ? 'text-destructive font-bold' : 'text-muted-foreground'}`}>
          {row.original.restartCount || 0}
        </span>
      ),
    },
    {
      accessorKey: "eventCount",
      header: "Events",
      cell: ({ row }) => (
        <Button
          variant="link"
          className="p-0 h-auto font-mono text-xs text-muted-foreground hover:text-primary"
          onClick={(e) => {
            e.stopPropagation()
            setSelectedInstanceForEvents(row.original.instanceName)
          }}
        >
          {row.original.eventCount || 0}
        </Button>
      ),
    },
    {
      accessorKey: "nodeName",
      header: "Node",
      cell: ({ row }) => (
        <div className="flex flex-col">
          <span className="text-xs text-muted-foreground">{row.original.nodeName}</span>
          {row.original.nodeIP && (
            <span className="font-mono text-[10px] text-muted-foreground/60">{row.original.nodeIP}</span>
          )}
        </div>
      ),
    },
    {
      accessorKey: "runningDuration",
      header: "Age",
      cell: ({ row }) => (
        <div className="flex items-center gap-1.5 text-muted-foreground">
          <Clock className="h-3.5 w-3.5" />
          <span className="text-xs">{row.original.runningDuration}</span>
        </div>
      ),
    },
    {
      id: "actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => {
        const instance = row.original
        const appContainerName = app?.slug ? `${app.slug}-app` : ""
        const containers = instance.containers || [appContainerName]
        const defaultContainer = containers.includes(appContainerName) ? appContainerName : containers[0]
        return (
          <div className="flex items-center justify-end gap-1">
            {!isViewer && (
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={(e) => {
                e.stopPropagation()
                if (app) {
                  openPanel({
                    type: "logs",
                    appId: app.id,
                    appName: app.name,
                    instanceName: instance.instanceName,
                    containerName: defaultContainer,
                    containers,
                    initContainers: instance.initContainers,
                  })
                }
              }}
              title="View Logs"
            >
              <FileText />
            </Button>
            )}
            {!isViewer && (
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={(e) => {
                e.stopPropagation()
                if (app) {
                  openPanel({
                    type: "terminal",
                    appId: app.id,
                    appName: app.name,
                    instanceName: instance.instanceName,
                    containerName: defaultContainer,
                    containers,
                    initContainers: instance.initContainers,
                  })
                }
              }}
              title="Open Terminal"
            >
              <TerminalIcon />
            </Button>
            )}
            {!isViewer && (
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={(e) => {
                e.stopPropagation()
                if (app) {
                  openPanel({
                    type: "files",
                    appId: app.id,
                    appName: app.name,
                    instanceName: instance.instanceName,
                    containerName: defaultContainer,
                    containers,
                    initContainers: instance.initContainers,
                  })
                }
              }}
              title="File Explorer"
            >
              <FolderOpen />
            </Button>
            )}
            {!isViewer && (
            <Button
              variant="ghost"
              size="icon-sm"
              className="text-destructive hover:text-destructive hover:bg-destructive/10"
              onClick={(e) => {
                e.stopPropagation()
                setDeletingInstanceName(instance.instanceName)
                setDeleteInstanceDialogOpen(true)
              }}
              disabled={deleteInstanceMutation.isPending}
              title="Delete Instance"
            >
              <Trash2 />
            </Button>
            )}
          </div>
        )
      },
    },
  ]

  const bulkDeleteMutation = useMutation({
    mutationFn: async (instanceNames: string[]) => {
      return Promise.all(instanceNames.map(name => appsApi.deleteInstance(appId!, name)))
    },
    onSuccess: () => {
      toast.success("Bulk deletion initiated")
      queryClient.invalidateQueries({ queryKey: ['app-instances', appId] })
    },
    onError: (error: any) => {
      toast.error("Failed to delete instances", {
        description: error.response?.data?.error || error.message
      })
    }
  })

  if (isLoading) {
    return (
      <div className="flex flex-col flex-1 gap-6 animate-pulse">
        <div className="flex flex-col gap-2">
          <Skeleton className="h-4 w-32" />
          <div className="flex justify-between items-start">
            <div className="flex items-center gap-4">
              <Skeleton className="h-14 w-14 rounded-lg" />
              <div className="space-y-2">
                <Skeleton className="h-8 w-48" />
                <Skeleton className="h-4 w-32" />
              </div>
            </div>
            <div className="flex flex-col items-end gap-2">
              <Skeleton className="h-6 w-20" />
              <div className="flex gap-2">
                <Skeleton className="h-9 w-24" />
                <Skeleton className="h-9 w-24" />
              </div>
            </div>
          </div>
        </div>
        <div className="space-y-4">
          <Skeleton className="h-10 w-100" />
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            <Skeleton className="h-32 w-full" />
            <Skeleton className="h-32 w-full" />
            <Skeleton className="h-32 w-full" />
            <Skeleton className="h-32 w-full" />
          </div>
          <Skeleton className="h-64 w-full" />
        </div>
      </div>
    )
  }

  if (error || !app) {
    return (
      <NotFoundPage
        resourceType="Application"
        backHref="/applications"
        backLabel="Back to Applications"
      />
    )
  }

  const breadcrumbs = [
    { label: "Applications", icon: Box },
    {
      label: currentEnv?.name || "Environment",
      icon: Orbit,
      href: '/applications',
      dropdown: safeEnvs.length > 1 ? (
        <DropdownMenu>
          <DropdownMenuTrigger render={<Button variant="ghost" size="icon-sm"><ChevronsUpDown /></Button>} />
          <DropdownMenuContent align="start" className="w-40">
            <DropdownMenuGroup>
              {safeEnvs.map(env => (
                <DropdownMenuItem
                  key={env.id}
                  onClick={() => {
                    setActiveEnvId(env.id)
                    navigate('/applications')
                  }}
                >
                  <Orbit className="mr-2 h-4 w-4" />
                  {env.name}
                </DropdownMenuItem>
              ))}
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      ) : undefined
    },
    {
      label: app.name,
      icon: Box,
      dropdown: safeApps.length > 1 ? (
        <DropdownMenu>
          <DropdownMenuTrigger render={<Button variant="ghost" size="icon-sm"><ChevronsUpDown /></Button>} />
          <DropdownMenuContent align="start" className="w-48">
            <DropdownMenuGroup>
              {safeApps.map(appItem => (
                <DropdownMenuItem
                  key={appItem.id}
                  onClick={() => navigate(`/applications/${appItem.id}`)}
                >
                  <Box className="mr-2 h-4 w-4" />
                  {appItem.name}
                </DropdownMenuItem>
              ))}
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      ) : undefined
    }
  ]

  return (
    <div className="flex flex-col flex-1 gap-6">
      <PageHeader items={breadcrumbs} />

      <div className="flex flex-col gap-4">
        <div className="flex justify-between items-start">
          <div className="flex items-center gap-4">
            <div className="p-3 bg-primary/10 rounded-lg text-primary">
              <Box className="h-8 w-8" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-2xl font-bold tracking-tight">{app.name}</h1>
                <ColorBadge color={getAppStatusColor(app.status)}>
                  {app.status.toUpperCase()}
                </ColorBadge>
              </div>
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <span className="font-mono">{app.slug}</span>
                {app.description && (
                  <>
                    <span>•</span>
                    <span>{app.description}</span>
                  </>
                )}
              </div>
            </div>
          </div>
          <div className="flex items-center gap-2">
            {!isViewer && (
            <Button
              variant="outline"
              onClick={() => setIsEditAppDialogOpen(true)}
            >
              <Pencil />
              Edit
            </Button>
            )}
            {!isViewer && availableActions && availableActions.actions && (
              <AppActionButtons
                appId={app.id}
                actions={availableActions.actions}
                onDeleteSuccess={() => navigate('/applications')}
              />
            )}
          </div>
        </div>
      </div>

      <Tabs value={currentTab} onValueChange={(v) => setSearchParams({ tab: v }, { replace: true })}>
        <TabsList>
          <TabsTrigger value="overview"><Telescope />Overview</TabsTrigger>
          <TabsTrigger value="topology"><Share2 />Topology</TabsTrigger>
          <TabsTrigger value="instances"><Shapes />Instances</TabsTrigger>
          <TabsTrigger value="resources"><Ruler />Resources</TabsTrigger>
          <TabsTrigger value="env-vars"><Key />Env Vars</TabsTrigger>
          <TabsTrigger value="config-files"><FileCog />Config Files</TabsTrigger>
          <TabsTrigger value="volumes"><HardDrive />Volumes</TabsTrigger>
          <TabsTrigger value="gateways"><Network />Gateways</TabsTrigger>
          <TabsTrigger value="plugins"><Puzzle />Plugins</TabsTrigger>
          <TabsTrigger value="build"><Hammer />Deploy</TabsTrigger>
          <TabsTrigger value="advanced"><Cog />Advanced</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-4 mt-2">
          <Card className="bg-linear-to-b/increasing from-primary/5 to-transparent data-[active=true]:bg-transparent">
            <CardHeader>
              <CardTitle className="text-sm flex items-center gap-2">
                <Info className="h-4 w-4" />
                Application Information
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-1 lg:grid-cols-3">
                <div>
                  <p className="text-xs font-medium text-muted-foreground">Slug</p>
                  <div className="flex items-center gap-2">
                    <p className="text-sm font-mono">{app.slug}</p>
                    <Button
                      variant="ghost"
                      size="icon-xs"
                      onClick={() => {
                        navigator.clipboard.writeText(app.slug)
                        toast.success("Copied to clipboard")
                      }}
                    >
                      <Copy className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                </div>
                <div>
                  <p className="text-xs font-medium text-muted-foreground">Image</p>
                  <div className="flex items-center gap-2">
                    <p className="text-sm font-mono truncate">{app.container_image}</p>
                    {app.registry_username && (
                      <Key className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                    )}
                    <Button
                      variant="ghost"
                      size="icon-xs"
                      onClick={() => {
                        navigator.clipboard.writeText(app.container_image)
                        toast.success("Copied to clipboard")
                      }}
                    >
                      <Copy className="h-3.5 w-3.5" />
                    </Button>
                    {!isViewer && (
                    <Button
                      variant="ghost"
                      size="icon-xs"
                      onClick={() => setIsEditImageDialogOpen(true)}
                    >
                      <Edit2 className="h-3.5 w-3.5" />
                    </Button>
                    )}
                  </div>
                </div>
                <div>
                  <p className="text-xs font-medium text-muted-foreground">Created At</p>
                  <p className="text-sm">{app.created_at ? formatDate(app.created_at) : "N/A"}</p>
                </div>
              </div>
            </CardContent>
          </Card>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">
                  CPU
                </CardTitle>
                <Cpu className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{app.request_cpu} / {app.limit_cpu} m</div>
                <p className="text-xs text-muted-foreground">
                  Request/Limit
                </p>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">
                  Memory
                </CardTitle>
                <MemoryStick className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{app.request_memory} / {app.limit_memory} Mi</div>
                <p className="text-xs text-muted-foreground">
                  Request/Limit
                </p>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">
                  Deploy Type
                </CardTitle>
                <CloudCog className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{app.app_type}</div>
                <p className="text-xs text-muted-foreground">
                </p>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">
                  Replicas
                </CardTitle>
                <Layers className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{app.replicas}{app.auto_scaling ? ` (AutoScaling: ${app.auto_scaling.min_replicas}-${app.auto_scaling.max_replicas})` : ""}</div>
                <p className="text-xs text-muted-foreground">
                  Desired
                </p>
              </CardContent>
            </Card>
          </div>

          {currentEnv && (
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm flex items-center gap-2">
                  <ChartLine className="h-4 w-4" />Resource Usage
                </CardTitle>
              </CardHeader>
              <CardContent>
                <AppMetrics
                  clusterId={currentEnv.cluster_id}
                  namespace={currentEnv.cluster_namespace}
                  appSlug={app.slug}
                  app={app}
                />
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="topology" className="space-y-4 mt-2">
          <Card>
            <CardHeader>
              <CardTitle className="text-sm flex items-center gap-2">
                <Share2 className="h-4 w-4" />
                Topology
              </CardTitle>
              <CardDescription>
                Resource topology of the application
              </CardDescription>
            </CardHeader>
            <CardContent>
              <TopologyView appId={app.id} />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="instances" className="space-y-4 mt-2">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm flex items-center gap-2">
                <Shapes className="h-4 w-4" /> Running Instances
              </CardTitle>
              <CardDescription>
                Manage and monitor current pod instances
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex flex-wrap items-center justify-between gap-4">
                <Input
                  className="flex flex-1 max-w-sm min-w-75"
                  placeholder="Filter pods..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                />

                <div className="flex items-center gap-2">
                  {Object.keys(rowSelection).length > 0 && !isViewer && (
                    <Button
                      variant="destructive"
                      onClick={() => {
                        const selectedIndices = Object.keys(rowSelection).filter(key => rowSelection[key as keyof typeof rowSelection])
                        const selectedNames = selectedIndices.map(idx => filteredInstances[parseInt(idx)]?.instanceName).filter(Boolean) as string[]

                        setSelectedInstanceNames(selectedNames)
                        setBulkDeleteDialogOpen(true)
                      }}
                      disabled={bulkDeleteMutation.isPending}
                    >
                      <Trash2 />
                      Delete ({Object.keys(rowSelection).filter(key => rowSelection[key as keyof typeof rowSelection]).length})
                    </Button>
                  )}
                  <Tabs value={viewMode} onValueChange={(v) => setViewMode(v as any)} className="w-auto h-7">
                    <TabsList>
                      <TabsTrigger value="table" className="px-2">
                        <List />
                      </TabsTrigger>
                      <TabsTrigger value="card">
                        <LayoutGrid />
                      </TabsTrigger>
                    </TabsList>
                  </Tabs>
                  {!isViewer && <ScaleAppPopover app={app} />}
                </div>
              </div>
              {filteredInstances.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-12 gap-4 border rounded-lg border-dashed">
                  {searchQuery ? (
                    <Search className="h-12 w-12 text-muted-foreground opacity-20" />
                  ) : (
                    <Shapes className="h-12 w-12 text-muted-foreground opacity-20" />
                  )}
                  <div className="text-center">
                    <p className="text-sm font-medium">
                      {searchQuery ? `No results for "${searchQuery}"` : "No instances running"}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {searchQuery ? "Try a different search term" : "The application might be scaled to zero or not yet deployed."}
                    </p>
                  </div>
                </div>
              ) : viewMode === 'table' ? (
                <DataTable
                  borderless
                  columns={instanceColumns}
                  data={filteredInstances}
                  rowSelection={rowSelection}
                  onRowSelectionChange={setRowSelection}
                  hidePagination
                />
              ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  {filteredInstances.map((instance) => {
                    const isRunning = instance.status === "Running"
                    const appContainerName = app?.slug ? `${app.slug}-app` : ""
                    const containers = instance.containers || [appContainerName]
                    const defaultContainer = containers.includes(appContainerName) ? appContainerName : containers[0]

                    return (
                      <Card key={instance.instanceName} className="group/card hover:shadow-md transition-shadow cursor-pointer">
                        <CardHeader className="pb-2">
                          <div className="flex items-start justify-between gap-4">
                            <div className="flex items-start gap-3 min-w-0">
                              <Avatar className="h-10 w-10 rounded-lg bg-primary/10 text-primary border-none">
                                <AvatarFallback className="rounded-lg text-lg font-bold">
                                  {instance.instanceName.charAt(0).toUpperCase()}
                                </AvatarFallback>
                              </Avatar>
                              <div className="flex flex-col min-w-0">
                                <CardTitle className="font-mono text-xs font-semibold truncate" title={instance.instanceName}>
                                  {instance.instanceName}
                                </CardTitle>
                                <CardDescription className="font-mono text-[10px] truncate">
                                  {instance.ip || 'No IP assigned'}
                                </CardDescription>
                              </div>
                            </div>
                            <ColorBadge color={isRunning ? "green" : "gray"} className="text-[10px] px-1.5 py-0 shrink-0">
                              {instance.status.toUpperCase()}
                            </ColorBadge>
                          </div>
                        </CardHeader>
                        <CardContent className="space-y-4 pt-2">
                          <div className="space-y-2">
                            <div className="flex items-center justify-between text-xs text-muted-foreground">
                              <div className="flex items-center gap-2">
                                <Server className="h-3.5 w-3.5" />
                                <span className="truncate" title={instance.nodeName}>{instance.nodeName}</span>
                              </div>
                            </div>
                            <div className="flex items-center justify-between text-xs text-muted-foreground">
                              <div className="flex items-center gap-3">
                                {instance.initContainerCount > 0 && (
                                  <div className="flex items-center gap-1.5" title="Init Containers">
                                    <Zap className="h-3.5 w-3.5" />
                                    <span>{instance.initContainerCount}</span>
                                  </div>
                                )}
                                <div className="flex items-center gap-1.5" title="Containers">
                                  <Layers2 className="h-3.5 w-3.5" />
                                  <span>{instance.containerCount}</span>
                                </div>
                                <div className={`flex items-center gap-1.5 ${instance.restartCount > 0 ? 'text-destructive font-bold' : ''}`} title="Restarts">
                                  <RotateCw className="h-3 w-3" />
                                  <span>{instance.restartCount || 0}</span>
                                </div>
                              </div>
                              <Button
                                variant="link"
                                className="p-0 h-auto text-xs"
                                onClick={(e) => {
                                  e.stopPropagation()
                                  setSelectedInstanceForEvents(instance.instanceName)
                                }}
                              >
                                Events: {instance.eventCount || 0}
                              </Button>
                            </div>
                          </div>
                          <div className="flex items-center justify-between gap-2 text-[10px] text-muted-foreground/60 border-t pt-2">
                            <div className="flex items-center gap-1.5">
                              <Clock className="h-3 w-3" />
                              <span>{instance.runningDuration}</span>
                            </div>
                            <div className="flex items-center gap-1">
                              {!isViewer && (
                              <Button
                                variant="ghost"
                                size="icon-xs"
                                className="h-6 w-6 opacity-0 group-hover/card:opacity-100 transition-opacity"
                                onClick={(e) => {
                                  e.stopPropagation()
                                  if (app) {
                                    openPanel({
                                      type: "logs",
                                      appId: app.id,
                                      appName: app.name,
                                      instanceName: instance.instanceName,
                                      containerName: defaultContainer,
                                      containers,
                                      initContainers: instance.initContainers,
                                    })
                                  }
                                }}
                                title="View Logs"
                              >
                                <FileText className="h-3.5 w-3.5" />
                              </Button>
                              )}
                              {!isViewer && (
                              <Button
                                variant="ghost"
                                size="icon-xs"
                                className="h-6 w-6 opacity-0 group-hover/card:opacity-100 transition-opacity"
                                onClick={(e) => {
                                  e.stopPropagation()
                                  if (app) {
                                    openPanel({
                                      type: "terminal",
                                      appId: app.id,
                                      appName: app.name,
                                      instanceName: instance.instanceName,
                                      containerName: defaultContainer,
                                      containers,
                                      initContainers: instance.initContainers,
                                    })
                                  }
                                }}
                                title="Open Terminal"
                              >
                                <TerminalIcon className="h-3.5 w-3.5" />
                              </Button>
                              )}
                              {!isViewer && (
                              <Button
                                variant="ghost"
                                size="icon-xs"
                                className="h-6 w-6 opacity-0 group-hover/card:opacity-100 transition-opacity"
                                onClick={(e) => {
                                  e.stopPropagation()
                                  if (app) {
                                    openPanel({
                                      type: "files",
                                      appId: app.id,
                                      appName: app.name,
                                      instanceName: instance.instanceName,
                                      containerName: defaultContainer,
                                      containers,
                                      initContainers: instance.initContainers,
                                    })
                                  }
                                }}
                                title="File Explorer"
                              >
                                <FolderOpen className="h-3.5 w-3.5" />
                              </Button>
                              )}
                              {!isViewer && (
                              <Button
                                variant="ghost"
                                size="icon-xs"
                                className="h-6 w-6 opacity-0 group-hover/card:opacity-100 transition-opacity text-destructive hover:text-destructive hover:bg-destructive/10"
                                onClick={(e) => {
                                  e.stopPropagation()
                                  setDeletingInstanceName(instance.instanceName)
                                  setDeleteInstanceDialogOpen(true)
                                }}
                                disabled={deleteInstanceMutation.isPending}
                                title="Delete Instance"
                              >
                                <Trash2 className="h-3.5 w-3.5" />
                              </Button>
                              )}
                            </div>
                          </div>
                        </CardContent>
                      </Card>
                    )
                  })}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="resources" className="space-y-4 mt-2">
          <ResourceConfig app={app} />
        </TabsContent>

        <TabsContent value="gateways" className="space-y-4 mt-2">
          <NetworkConfig app={app} />
        </TabsContent>

        <TabsContent value="env-vars" className="space-y-4 mt-2">
          <EnvVarTable app={app} />
        </TabsContent>

        <TabsContent value="volumes" className="space-y-4 mt-2">
          <VolumesTable app={app} />
        </TabsContent>

        <TabsContent value="config-files" className="space-y-4 mt-2">
          <ConfigFilesTable app={app} />
        </TabsContent>

        <TabsContent value="plugins" className="space-y-4 mt-2">
          <AppPlugins appId={app.id} projectId={projectIdToUse!} />
        </TabsContent>

        <TabsContent value="build" className="space-y-4 mt-2">
          {app.code_repository_id && (
            <Card className="bg-linear-to-b/increasing from-primary/5 to-transparent">
              <CardHeader>
                <CardTitle className="text-sm flex items-center gap-2">
                  <FolderGit2 className="h-4 w-4" />
                  Code Repository
                </CardTitle>
                <CardDescription>
                  This application was deployed from a code repository. View build history and trigger new builds.
                </CardDescription>
              </CardHeader>
              <CardContent className="flex items-center gap-2">
                <Button variant="outline" render={<Link to={`/code-repositories/${app.code_repository_id}`} className="flex items-center whitespace-nowrap" />}>
                  <ExternalLink className="h-4 w-4 mr-2" />
                  View in Code Repository
                </Button>
                <Button onClick={() => setIsUnifiedBuildDialogOpen(true)} className="flex items-center">
                  <Hammer className="h-4 w-4 mr-2" />
                  Build & Deploy
                </Button>
              </CardContent>
            </Card>
          )}
          <BuildList appId={app.id} />
          <DeploymentHistoryList appId={app.id} />
        </TabsContent>

        <TabsContent value="advanced" className="space-y-4 mt-2">
          <CommandConfig app={app} />
          <AutoScalingConfig app={app} />
          <HealthConfig app={app} />
          <SchedulingConfig app={app} />
        </TabsContent>
      </Tabs>

      <ImageEditor
        app={app}
        open={isEditImageDialogOpen}
        onOpenChange={setIsEditImageDialogOpen}
      />

      <EditAppDialog
        app={app}
        open={isEditAppDialogOpen}
        onOpenChange={setIsEditAppDialogOpen}
        onSuccess={() => queryClient.invalidateQueries({ queryKey: ['app', app.id] })}
      />

      {app && app.code_repository_id && projectIdToUse && (
        <UnifiedBuildDeployDialog
          open={isUnifiedBuildDialogOpen}
          onOpenChange={setIsUnifiedBuildDialogOpen}
          repoId={app.code_repository_id}
          projectId={projectIdToUse}
          preSelectedDeployEnvId={app.env_id}
          preSelectedDeployAppId={app.id}
        />
      )}

      {app && selectedInstanceForEvents && (
        <InstanceEventsDialog
          appId={app.id}
          instanceName={selectedInstanceForEvents}
          open={!!selectedInstanceForEvents}
          onOpenChange={(open) => !open && setSelectedInstanceForEvents(null)}
        />
      )}

      <AlertDialog open={deleteInstanceDialogOpen} onOpenChange={setDeleteInstanceDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Instance</AlertDialogTitle>
            <AlertDialogDescription>
              {deletingInstanceName
                ? `Are you sure you want to delete instance "${deletingInstanceName}"?`
                : "Are you sure you want to delete this instance?"}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (deletingInstanceName) {
                  deleteInstanceMutation.mutate(deletingInstanceName)
                }
                setDeleteInstanceDialogOpen(false)
                setDeletingInstanceName(null)
              }}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={bulkDeleteDialogOpen} onOpenChange={setBulkDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Instances</AlertDialogTitle>
            <AlertDialogDescription>
              {selectedInstanceNames.length > 0
                ? `Are you sure you want to delete ${selectedInstanceNames.length} instances? This action cannot be undone.`
                : "Are you sure you want to delete these instances?"}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (selectedInstanceNames.length > 0) {
                  bulkDeleteMutation.mutate(selectedInstanceNames)
                  setRowSelection({})
                }
                setBulkDeleteDialogOpen(false)
                setSelectedInstanceNames([])
              }}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

export default ApplicationDetailPage
