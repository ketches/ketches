import { type App, type AppInstance } from "@/api/apps"
import { DataTable } from "@/components/data-table/data-table"
import { RefreshButtonIcon, RefreshIndicator } from "@/components/data-table/refresh-indicator"
import { useRefreshAction } from "@/components/data-table/use-refresh-action"
import { InstanceResourceMetrics } from "@/components/monitoring/instance-resource-metrics"
import { MetricsTimeRangeSelector } from "@/components/monitoring/metrics-time-range-selector"
import { type TimeRange } from "@/components/monitoring/use-time-range"
import { ColorBadge } from "@/components/shared/color-badge"
import { EmptyState } from "@/components/shared/empty-state"
import { StatCard } from "@/components/shared/stat-card"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { cn } from "@/lib/utils"
import { formatDate } from "@/lib/utils"
import { type OnChangeFn, type PaginationState, type RowSelectionState, type ColumnDef } from "@tanstack/react-table"
import {
  ChartLine,
  Clock,
  ClockCheck,
  CloudCog,
  Container,
  Copy,
  Cpu,
  Edit2,
  FileClock,
  Info,
  Key,
  Layers,
  Layers2,
  LayoutGrid,
  List,
  MemoryStick,
  RotateCw,
  Server,
  Terminal as TerminalIcon,
  Trash2,
  Zap,
  FolderOpen,
} from "lucide-react"
import { toast } from "sonner"

import { ApplicationMetrics } from "./application-metrics"
import { ApplicationScalePopover } from "./application-scale-popover"

interface ApplicationOverviewTabProps {
  app: App
  currentEnv: App["env"]
  projectIdToUse?: string
  isViewer: boolean
  isLoading: boolean
  instancesLoading: boolean
  instancesFetching: boolean
  onRefreshInstances: () => void
  viewMode: "table" | "card"
  onViewModeChange: (value: "table" | "card") => void
  rowSelection: RowSelectionState
  onRowSelectionChange: OnChangeFn<RowSelectionState>
  instancePagination: PaginationState
  onInstancePaginationChange: OnChangeFn<PaginationState>
  safeInstances: AppInstance[]
  selectedInstanceNamesFromRows: string[]
  selectedInstanceCount: number
  timeRange: TimeRange
  onTimeRangeChange: (value: TimeRange) => void
  rangeSeconds: number
  step: string
  deleteInstancePending: boolean
  metricsInstance: string | null
  onMetricsInstanceChange: (instanceName: string | null) => void
  onOpenLogs: (instance: AppInstance, containerName: string) => void
  onOpenTerminal: (instance: AppInstance, containerName: string) => void
  onOpenFiles: (instance: AppInstance, containerName: string) => void
  onOpenInstanceEvents: (instanceName: string) => void
  onRequestDeleteInstance: (instanceName: string) => void
  onRequestBulkDelete: (instanceNames: string[]) => void
  onEditImage: () => void
}

function getDefaultContainerName(app: App, instance: AppInstance): string {
  const appContainerName = app.slug ? `${app.slug}-app` : ""
  const containers = instance.containers || [appContainerName]
  return containers.includes(appContainerName) ? appContainerName : containers[0]
}

function InstanceActionButtons({
  isViewer,
  deleteInstancePending,
  onMetrics,
  onLogs,
  onTerminal,
  onFiles,
  onDelete,
}: {
  isViewer: boolean
  deleteInstancePending: boolean
  onMetrics: () => void
  onLogs: () => void
  onTerminal: () => void
  onFiles: () => void
  onDelete: () => void
}) {
  return (
    <div className="flex items-center justify-end gap-1">
      <Tooltip>
        <TooltipTrigger
          delay={200}
          render={<Button variant="ghost" size="icon-sm" onClick={onMetrics} />}
        >
          <ChartLine />
        </TooltipTrigger>
        <TooltipContent>View Metrics</TooltipContent>
      </Tooltip>
      {!isViewer && (
        <Tooltip>
          <TooltipTrigger
            delay={200}
            render={<Button variant="ghost" size="icon-sm" onClick={onLogs} />}
          >
            <FileClock />
          </TooltipTrigger>
          <TooltipContent>View Logs</TooltipContent>
        </Tooltip>
      )}
      {!isViewer && (
        <Tooltip>
          <TooltipTrigger
            delay={200}
            render={<Button variant="ghost" size="icon-sm" onClick={onTerminal} />}
          >
            <TerminalIcon />
          </TooltipTrigger>
          <TooltipContent>Open Terminal</TooltipContent>
        </Tooltip>
      )}
      {!isViewer && (
        <Tooltip>
          <TooltipTrigger
            delay={200}
            render={<Button variant="ghost" size="icon-sm" onClick={onFiles} />}
          >
            <FolderOpen />
          </TooltipTrigger>
          <TooltipContent>File Explorer</TooltipContent>
        </Tooltip>
      )}
      {!isViewer && (
        <Tooltip>
          <TooltipTrigger
            delay={200}
            render={(
              <Button
                variant="ghost"
                size="icon-sm"
                className="text-destructive hover:text-destructive hover:bg-destructive/10"
                onClick={onDelete}
                disabled={deleteInstancePending}
              />
            )}
          >
            <Trash2 />
          </TooltipTrigger>
          <TooltipContent>Delete Instance</TooltipContent>
        </Tooltip>
      )}
    </div>
  )
}

export function ApplicationOverviewTab({
  app,
  currentEnv,
  projectIdToUse,
  isViewer,
  isLoading,
  instancesLoading,
  instancesFetching,
  onRefreshInstances,
  viewMode,
  onViewModeChange,
  rowSelection,
  onRowSelectionChange,
  instancePagination,
  onInstancePaginationChange,
  safeInstances,
  selectedInstanceNamesFromRows,
  selectedInstanceCount,
  timeRange,
  onTimeRangeChange,
  rangeSeconds,
  step,
  deleteInstancePending,
  metricsInstance,
  onMetricsInstanceChange,
  onOpenLogs,
  onOpenTerminal,
  onOpenFiles,
  onOpenInstanceEvents,
  onRequestDeleteInstance,
  onRequestBulkDelete,
  onEditImage,
}: ApplicationOverviewTabProps) {
  const refreshAction = useRefreshAction({
    onRefresh: onRefreshInstances,
    isLoading: instancesLoading,
  })
  const hasCardSelection = selectedInstanceCount > 0
  const isAllInstancesSelected = safeInstances.length > 0 && selectedInstanceCount === safeInstances.length
  const isSomeInstancesSelected = selectedInstanceCount > 0 && selectedInstanceCount < safeInstances.length

  const handleToggleAllInstances = (checked: boolean) => {
    if (checked) {
      onRowSelectionChange(
        Object.fromEntries(safeInstances.map((_, index) => [index.toString(), true]))
      )
      return
    }

    onRowSelectionChange({})
  }

  const handleToggleInstance = (index: number, checked: boolean) => {
    onRowSelectionChange((previous) => {
      const nextSelection = { ...previous }

      if (checked) {
        nextSelection[index] = true
      } else {
        delete nextSelection[index]
      }

      return nextSelection
    })
  }

  const instanceColumns: ColumnDef<AppInstance>[] = [
    {
      id: "select",
      header: ({ table }) => (
        <Checkbox
          checked={table.getIsAllPageRowsSelected()}
          data-state={table.getIsSomePageRowsSelected() && !table.getIsAllPageRowsSelected() ? "indeterminate" : undefined}
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
      accessorKey: "instance_name",
      header: "Instance",
      cell: ({ row }) => (
        <div className="flex flex-col">
          <span className="font-mono text-xs font-medium">{row.original.instance_name}</span>
          {row.original.ip && (
            <span className="font-mono text-[10px] text-muted-foreground">{row.original.ip}</span>
          )}
        </div>
      ),
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => (
        <ColorBadge color={row.original.status === "Running" ? "green" : "gray"}>
          {row.original.status}
        </ColorBadge>
      ),
    },
    {
      id: "containers",
      header: "Containers",
      cell: ({ row }) => (
        <div className="flex items-center gap-3">
          {row.original.init_container_count > 0 && (
            <div className="flex items-center gap-1.5 text-muted-foreground" title="Init Containers">
              <Zap className="h-3.5 w-3.5" />
              <span>{row.original.init_container_count}</span>
            </div>
          )}
          <div className="flex items-center gap-1.5 text-muted-foreground" title="Containers">
            <Layers2 className="h-3.5 w-3.5" />
            <span>{row.original.container_count}</span>
          </div>
        </div>
      ),
    },
    {
      accessorKey: "restart_count",
      header: "Restarts",
      cell: ({ row }) => (
        <span className={`text-xs font-mono ${row.original.restart_count > 0 ? "text-destructive font-bold" : "text-muted-foreground"}`}>
          {row.original.restart_count || 0}
        </span>
      ),
    },
    {
      accessorKey: "eventCount",
      header: "Events",
      cell: ({ row }) => (
        <Button
          variant="link"
          size="icon-sm"
          className="text-primary"
          onClick={(event) => {
            event.stopPropagation()
            onOpenInstanceEvents(row.original.instance_name)
          }}
        >
          <ClockCheck />
        </Button>
      ),
    },
    {
      accessorKey: "node_name",
      header: "Node",
      cell: ({ row }) => (
        <div className="flex flex-col">
          <span className="text-xs text-muted-foreground">{row.original.node_name}</span>
          {row.original.node_ip && (
            <span className="font-mono text-[10px] text-muted-foreground/60">{row.original.node_ip}</span>
          )}
        </div>
      ),
    },
    {
      accessorKey: "running_duration",
      header: "Age",
      cell: ({ row }) => (
        <div className="flex items-center gap-1.5 text-muted-foreground">
          <Clock className="h-3.5 w-3.5" />
          <span className="text-xs">{row.original.running_duration}</span>
        </div>
      ),
    },
    {
      id: "actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => {
        const instance = row.original
        const defaultContainer = getDefaultContainerName(app, instance)

        return (
          <InstanceActionButtons
            isViewer={isViewer}
            deleteInstancePending={deleteInstancePending}
            onMetrics={() => onMetricsInstanceChange(instance.instance_name)}
            onLogs={() => onOpenLogs(instance, defaultContainer)}
            onTerminal={() => onOpenTerminal(instance, defaultContainer)}
            onFiles={() => onOpenFiles(instance, defaultContainer)}
            onDelete={() => onRequestDeleteInstance(instance.instance_name)}
          />
        )
      },
    },
  ]

  return (
    <>
      <Card className="bg-linear-to-b/increasing from-blue-500/5 to-transparent data-[active=true]:bg-transparent">
        <CardHeader>
          <CardTitle className="text-sm flex items-center gap-2">
            <Info className="h-4 w-4" />
            Application Information
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-1 lg:grid-cols-3">
            <div className="space-y-1">
              <p className="text-xs font-medium text-muted-foreground">Slug</p>
              <div className="flex items-center gap-2">
                <p className="text-sm font-mono">{app.slug}</p>
                <Button
                  variant="ghost"
                  size="icon-xs"
                  className="opacity-0 group-hover/card:opacity-100 transition-opacity"
                  onClick={() => {
                    navigator.clipboard.writeText(app.slug)
                    toast.success("Copied to clipboard")
                  }}
                >
                  <Copy />
                </Button>
              </div>
            </div>
            <div className="space-y-1">
              <p className="text-xs font-medium text-muted-foreground">Image</p>
              <div className="flex items-center gap-2">
                <p className="text-sm font-mono truncate">{app.container_image}</p>
                {app.registry_username && (
                  <Key className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                )}
                {!isViewer && (
                  <Tooltip>
                    <TooltipTrigger
                      delay={200}
                      render={<Button variant="ghost" size="icon-xs" onClick={onEditImage} />}
                    >
                      <Edit2 className="h-3.5 w-3.5" />
                    </TooltipTrigger>
                    <TooltipContent>Edit Image</TooltipContent>
                  </Tooltip>
                )}
                <Button
                  variant="ghost"
                  size="icon-xs"
                  className="opacity-0 group-hover/card:opacity-100 transition-opacity"
                  onClick={() => {
                    navigator.clipboard.writeText(app.container_image)
                    toast.success("Copied to clipboard")
                  }}
                >
                  <Copy />
                </Button>
              </div>
            </div>
            <div className="space-y-1">
              <p className="text-xs font-medium text-muted-foreground">Created At</p>
              <div className="flex items-center gap-1.5 text-sm">
                <Clock className="h-3.5 w-3.5 text-muted-foreground" />
                <span>{app.created_at ? formatDate(app.created_at) : "N/A"}</span>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          title="CPU"
          value={`${app.request_cpu} / ${app.limit_cpu} m`}
          icon={Cpu}
          color="amber"
          description="Request / Limit"
        />
        <StatCard
          title="Memory"
          value={`${app.request_memory} / ${app.limit_memory} Mi`}
          icon={MemoryStick}
          color="amber"
          description="Request / Limit"
        />
        <StatCard
          title="Deploy Type"
          value={app.app_type}
          icon={CloudCog}
        />
        <StatCard
          title="Replicas"
          value={app.replicas}
          icon={Layers}
          color="sky"
          description={<>Desired {app.auto_scaling ? `(AS: ${app.auto_scaling.min_replicas}-${app.auto_scaling.max_replicas})` : ""}</>}
        />
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-sm flex items-center gap-2">
            <Container className="h-4 w-4" /> Running Instances
          </CardTitle>
          <CardAction className="flex flex-wrap items-center justify-end gap-2">
            {viewMode === "table" && selectedInstanceCount > 0 && !isViewer && (
              <Button
                variant="destructive"
                onClick={() => onRequestBulkDelete(selectedInstanceNamesFromRows)}
              >
                <Trash2 />
                Delete ({selectedInstanceCount})
              </Button>
            )}
            {!isViewer && viewMode === "card" && (
              <label className={cn(
                "flex items-center gap-2 cursor-pointer transition-opacity",
                hasCardSelection ? "opacity-100" : "opacity-0 group-hover/card:opacity-100"
              )}>
                <Checkbox
                  checked={(isAllInstancesSelected || (isSomeInstancesSelected ? "mixed" : false)) as boolean | undefined}
                  onCheckedChange={(value) => handleToggleAllInstances(!!value)}
                />
                <p className="text-xs font-medium text-muted-foreground">Select all</p>
              </label>
            )}
            {viewMode === "card" && selectedInstanceCount > 0 && !isViewer && (
              <Button
                variant="destructive"
                onClick={() => onRequestBulkDelete(selectedInstanceNamesFromRows)}
              >
                <Trash2 />
                Delete ({selectedInstanceCount})
              </Button>
            )}
            <Button
              variant="secondary"
              size="icon"
              aria-label="Refresh instances"
              disabled={instancesLoading || refreshAction.isRefreshing}
              onClick={refreshAction.handleRefresh}
            >
              <RefreshButtonIcon spinning={refreshAction.isRefreshing || instancesFetching} />
            </Button>
            <Tabs
              value={viewMode}
              onValueChange={(value) => {
                if (value !== "table" && value !== "card") {
                  return
                }
                onViewModeChange(value)
              }}
              className="w-auto h-7"
              >
                <TabsList>
                  <TabsTrigger value="table" className="px-2">
                    <List />
                  </TabsTrigger>
                <TabsTrigger value="card">
                  <LayoutGrid />
                </TabsTrigger>
                </TabsList>
            </Tabs>
            {!isViewer && <ApplicationScalePopover app={app} />}
          </CardAction>
        </CardHeader>
        <CardContent>
          {safeInstances.length === 0 ? (
            <EmptyState
              title="No instances running"
              description="The application might be scaled to zero or not yet deployed."
              icon={Container}
            />
          ) : (
            <div className="relative">
              {viewMode === "table" ? (
                <DataTable
                  columns={instanceColumns}
                  data={safeInstances}
                  isLoading={isLoading || instancesLoading}
                  refreshState={refreshAction}
                  rowSelection={rowSelection}
                  onRowSelectionChange={onRowSelectionChange}
                  pagination={instancePagination}
                  onPaginationChange={onInstancePaginationChange}
                  hidePagination
                />
              ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  {safeInstances.map((instance, index) => {
                    const defaultContainer = getDefaultContainerName(app, instance)
                    const isSelected = !!rowSelection[index]

                    return (
                      <div key={instance.instance_name} className="relative group/instance-card">
                        {!isViewer && (
                          <div className={cn(
                            "absolute top-2 left-2 z-10 transition-opacity",
                            isSelected ? "opacity-100" : "opacity-0 group-hover/instance-card:opacity-100",
                            hasCardSelection ? "opacity-100" : ""
                          )}>
                            <Checkbox
                              checked={isSelected}
                              onCheckedChange={(value) => handleToggleInstance(index, !!value)}
                              aria-label="Select row"
                              className="bg-background"
                            />
                          </div>
                        )}
                        <Card className="group/card hover:shadow-md transition-shadow cursor-pointer">
                        <CardHeader>
                          <div className="flex items-start justify-between gap-4">
                            <div className="flex items-start gap-3 min-w-0">
                              <Avatar className="h-10 w-10 rounded-lg bg-primary/10 text-primary border-none">
                                <AvatarFallback className="rounded-lg text-lg font-bold">
                                  {instance.instance_name.charAt(0).toUpperCase()}
                                </AvatarFallback>
                              </Avatar>
                              <div className="flex flex-col min-w-0">
                                <div className="flex items-center gap-2 flex-wrap">
                                  <CardTitle className="font-mono text-xs font-semibold truncate" title={instance.instance_name}>
                                    {instance.instance_name}
                                  </CardTitle>
                                  <ColorBadge color={instance.status === "Running" ? "green" : "gray"} className="text-[10px] px-1.5 py-0 shrink-0">
                                    {instance.status.toUpperCase()}
                                  </ColorBadge>
                                </div>
                                <CardDescription className="font-mono text-[10px] truncate">
                                  {instance.ip || "No IP assigned"}
                                </CardDescription>
                              </div>
                            </div>
                            <InstanceActionButtons
                              isViewer={isViewer}
                              deleteInstancePending={deleteInstancePending}
                              onMetrics={() => onMetricsInstanceChange(instance.instance_name)}
                              onLogs={() => onOpenLogs(instance, defaultContainer)}
                              onTerminal={() => onOpenTerminal(instance, defaultContainer)}
                              onFiles={() => onOpenFiles(instance, defaultContainer)}
                              onDelete={() => onRequestDeleteInstance(instance.instance_name)}
                            />
                          </div>
                        </CardHeader>
                        <CardContent className="space-y-4 pt-2">
                          <div className="space-y-2">
                            <div className="flex items-center justify-between text-xs text-muted-foreground">
                              <div className="flex items-center gap-2">
                                <Server className="h-3.5 w-3.5" />
                                <span className="truncate" title={instance.node_name}>{instance.node_name}</span>
                              </div>
                            </div>
                            <div className="flex items-center justify-between text-xs text-muted-foreground">
                              <div className="flex items-center gap-3">
                                {instance.init_container_count > 0 && (
                                  <div className="flex items-center gap-1.5" title="Init Containers">
                                    <Zap className="h-3.5 w-3.5" />
                                    <span>{instance.init_container_count}</span>
                                  </div>
                                )}
                                <div className="flex items-center gap-1.5" title="Containers">
                                  <Layers2 className="h-3.5 w-3.5" />
                                  <span>{instance.container_count}</span>
                                </div>
                                <div className={`flex items-center gap-1.5 ${instance.restart_count > 0 ? "text-destructive font-bold" : ""}`} title="Restarts">
                                  <RotateCw className="h-3 w-3" />
                                  <span>{instance.restart_count || 0}</span>
                                </div>
                              </div>
                              <Button
                                variant="link"
                                className="p-0 h-auto text-xs"
                                onClick={(event) => {
                                  event.stopPropagation()
                                  onOpenInstanceEvents(instance.instance_name)
                                }}
                              >
                                <ClockCheck className="h-3 w-3" />Events
                              </Button>
                            </div>
                          </div>
                          <div className="flex items-center justify-between gap-2 text-[10px] text-muted-foreground/60 border-t pt-2">
                            <div className="flex items-center gap-1.5">
                              <Clock className="h-3 w-3" />
                              <span>{instance.running_duration}</span>
                            </div>
                          </div>
                        </CardContent>
                        </Card>
                      </div>
                    )
                  })}
                </div>
              )}
              {viewMode === "card" && refreshAction.showRefreshOverlay && (
                <div className="absolute inset-0 z-20 flex items-center justify-center rounded-md bg-background/55 backdrop-blur-[1px]">
                  <RefreshIndicator />
                </div>
              )}
            </div>
          )}
        </CardContent>
      </Card>

      {currentEnv && (
        <Card className="bg-linear-to-b/increasing from-blue-500/5 to-transparent data-[active=true]:bg-transparent">
          <CardHeader>
            <CardTitle className="text-sm flex items-center gap-2">
              <ChartLine className="h-4 w-4" />
              Metrics
            </CardTitle>
            <CardAction>
              <MetricsTimeRangeSelector value={timeRange} onChange={onTimeRangeChange} />
            </CardAction>
          </CardHeader>
          <CardContent>
            <ApplicationMetrics
              clusterId={currentEnv.cluster_id}
              projectId={projectIdToUse}
              prometheusAvailable={currentEnv.has_prometheus_integration}
              namespace={currentEnv.cluster_namespace}
              appSlug={app.slug}
              app={app}
              timeRange={timeRange}
              rangeSeconds={rangeSeconds}
              step={step}
            />
          </CardContent>
        </Card>
      )}

      <InstanceResourceMetrics
        open={!!metricsInstance}
        onOpenChange={(open) => {
          if (!open) {
            onMetricsInstanceChange(null)
          }
        }}
        clusterId={currentEnv?.cluster_id || ""}
        projectId={projectIdToUse}
        namespace={currentEnv?.cluster_namespace || ""}
        podName={metricsInstance || ""}
        app={app}
      />
    </>
  )
}
