import { formatDate } from "@/lib/utils"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { AxiosError } from "axios"
import {
  AlertCircle,
  Blocks,
  ChartLine,
  ChevronsUpDown,
  CircleSlash,
  Clock,
  Copy,
  Cpu,
  GamepadDirectional,
  Globe,
  Info,
  Key,
  Link2,
  Loader2,
  MemoryStick,
  MoreVertical,
  Pencil,
  Server,
  ShieldCheck,
  ShipWheel,
  Telescope,
  Terminal,
  Trash2,
  Warehouse
} from "lucide-react"
import * as React from "react"
import { useNavigate, useParams, useSearchParams } from "react-router-dom"
import { toast } from "sonner"

import { clustersApi } from "@/api/clusters"
import { ClusterCertificates } from "@/components/cluster/cluster-certificates"
import { ClusterDomains } from "@/components/cluster/cluster-domains"
import { ClusterExtensions } from "@/components/cluster/cluster-extensions"
import { ClusterIntegrationsConfig } from "@/components/cluster/cluster-integrations-config"
import { EditClusterDialog } from "@/components/cluster/edit-cluster-dialog"
import { EditClusterKubeConfigDialog } from "@/components/cluster/edit-cluster-kube-config-dialog"
import { ContainerRegistryList } from "@/components/container-registries/container-registry-list"
import { DataTable } from "@/components/data-table/data-table"
import { NotFoundPage } from "@/components/layout/not-found-page"
import { PageHeader } from "@/components/layout/page-header"
import { ClusterNodeResourceMetrics } from "@/components/monitoring/cluster-node-resource-metrics"
import { MetricsTimeRangeSelector } from "@/components/monitoring/metrics-time-range-selector"
import { useTimeRange } from "@/components/monitoring/use-time-range"
import { ColorBadge } from "@/components/shared/color-badge"
import { EmptyState } from "@/components/shared/empty-state"
import { DetailHeroSkeleton, InfoCardSkeleton, PanelCardSkeleton, StatCardsSkeleton, TabsSkeleton } from "@/components/shared/page-skeletons"
import { StatCard } from "@/components/shared/stat-card"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Empty, EmptyContent, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { useBottomPanel } from "@/contexts/bottom-panel-context"
import type { ColumnDef } from "@tanstack/react-table"

interface Node {
  metadata: {
    name: string
    labels: Record<string, string>
  }
  spec: {
    unschedulable?: boolean
  }
  status: {
    nodeInfo: {
      kubeletVersion: string
    }
    addresses: Array<{ type: string; address: string }>
    capacity: {
      cpu: string
      memory: string
    }
    allocatable: {
      cpu: string
      memory: string
    }
    conditions: Array<{ type: string; status: string }>
  }
}

function NodeMetricsPanel({ clusterId, prometheusAvailable, node }: { clusterId: string; prometheusAvailable?: boolean; node: Node }) {
  const { timeRange, setTimeRange, rangeSeconds, step } = useTimeRange()
  const isReady = node.status.conditions?.find((c) => c.type === "Ready")?.status === "True"
  const internalIP = node.status.addresses?.find((a) => a.type === "InternalIP")?.address
  return (
    <div
      className="p-4 border rounded-lg"
    >
      <div className="flex items-center justify-between mb-4">
        <div>
          <div className="flex items-center gap-2">
            <h4 className="font-medium">{node.metadata.name}</h4>
            <ColorBadge color={isReady ? "green" : "red"}>
              {isReady ? "Ready" : "NotReady"}
            </ColorBadge>
          </div>
          <p className="text-xs text-muted-foreground font-mono">{internalIP || "N/A"}</p>
        </div>
        <div onClick={(e) => e.stopPropagation()}>
          <MetricsTimeRangeSelector value={timeRange} onChange={setTimeRange} />
        </div>
      </div>
      <ClusterNodeResourceMetrics
        clusterId={clusterId}
        prometheusAvailable={prometheusAvailable}
        nodeName={node.metadata.name}
        nodeIp={internalIP}
        timeRange={timeRange}
        rangeSeconds={rangeSeconds}
        step={step}
      />
    </div>
  )
}

export function ClusterDetailPage() {
  const { clusterId } = useParams<{ clusterId: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { openPanel } = useBottomPanel()
  const [editOpen, setEditOpen] = React.useState(false)
  const [editKubeConfigOpen, setEditKubeConfigOpen] = React.useState(false)
  const [deleteOpen, setDeleteOpen] = React.useState(false)
  const [searchParams, setSearchParams] = useSearchParams()
  const activeTab = searchParams.get("tab") || "overview"

  const { data: cluster, isLoading: clusterLoading, error: clusterError } = useQuery({
    queryKey: ["cluster", clusterId],
    queryFn: () => clustersApi.get(clusterId!),
    enabled: !!clusterId,
    retry: false,
  })

  const { data: nodes = [] } = useQuery({
    queryKey: ["cluster-nodes", clusterId],
    queryFn: () => clustersApi.listNodes(clusterId!),
    enabled: !!clusterId,
  })

  const checkConnectivityMutation = useMutation({
    mutationFn: () => clustersApi.checkConnectivity(clusterId!),
    onSuccess: () => {
      toast.success("Connectivity check started", {
        description: "Status will update shortly",
      })
      setTimeout(() => {
        queryClient.invalidateQueries({ queryKey: ["cluster", clusterId] })
      }, 2000)
    },
    onError: (error: AxiosError<{ error: string }>) => {
      toast.error("Failed to check connectivity", {
        description: error.response?.data?.error || error.message,
      })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: () => clustersApi.delete(clusterId!),
    onSuccess: () => {
      toast.success("Cluster deleted successfully")
      queryClient.invalidateQueries({ queryKey: ["clusters"] })
      navigate("/clusters")
    },
    onError: (error: AxiosError<{ error: string }>) => {
      toast.error("Failed to delete cluster", {
        description: error.response?.data?.error || error.message,
      })
    },
  })

  const safeNodes: Node[] = Array.isArray(nodes) ? nodes : []

  const columns: ColumnDef<Node>[] = [
    {
      accessorKey: "name",
      header: "Node Name",
      cell: ({ row }) => {
        const internalIP = row.original.status.addresses?.find(
          (addr) => addr.type === "InternalIP"
        )
        return (
          <div className="flex items-center gap-2">
            <div className="p-1.5 bg-sky-500/10 rounded-md text-sky-600 shrink-0">
              <Blocks className="h-4 w-4" />
            </div>
            <div className="min-w-0">
              <p className="font-medium text-sm truncate cursor-pointer hover:text-primary transition-colors" onClick={() => navigate(`/clusters/${clusterId}/nodes/${row.original.metadata.name}`)}>
                {row.original.metadata.name}
              </p>
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <p className="text-xs text-muted-foreground font-mono truncate">
                  {internalIP?.address}
                </p>
                {internalIP?.address && <Button
                  variant="ghost"
                  size="icon-sm"
                  className="opacity-0 group-hover/row:opacity-100 transition-opacity"
                  onClick={(e) => {
                    e.stopPropagation()
                    navigator.clipboard.writeText(internalIP?.address || "")
                    toast.success("Internal IP address copied to clipboard")
                  }}
                >
                  <Copy />
                </Button>}
              </div>
            </div>
          </div>
        )
      },
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => {
        const isReady = row.original.status.conditions?.find(
          (c) => c.type === "Ready"
        )?.status === "True"
        const isUnschedulable = row.original.spec.unschedulable
        return (
          <div className="flex items-center">
            <ColorBadge color={isReady ? "green" : "red"}>
              {isReady ? (
                <>
                  Ready
                </>
              ) : (
                <>
                  NotReady
                </>
              )}
            </ColorBadge>
            {isUnschedulable && (
              <ColorBadge color="yellow">
                Unschedulable
              </ColorBadge>
            )}
          </div>
        )
      },
    },
    {
      accessorKey: "role",
      header: "Role",
      cell: ({ row }) => {
        const labels = row.original.metadata.labels || {}
        const role = "node-role.kubernetes.io/control-plane" in labels ||
          "node-role.kubernetes.io/master" in labels ? "control-plane" : "worker"
        return <span className="text-xs">{role}</span>
      },
    },
    {
      accessorKey: "version",
      header: "Version",
      cell: ({ row }) => (
        <span className="font-mono text-xs">
          {row.original.status.nodeInfo.kubeletVersion}
        </span>
      ),
    },
    {
      accessorKey: "cpu",
      header: "CPU",
      cell: ({ row }) => {
        const capacity = row.original.status.capacity?.cpu || "0"
        const allocatable = row.original.status.allocatable?.cpu || "0"
        return (
          <div className="text-xs">
            <div className="font-medium">{allocatable} / {capacity}</div>
            <div className="text-muted-foreground">Allocatable / Capacity</div>
          </div>
        )
      },
    },
    {
      accessorKey: "memory",
      header: "Memory",
      cell: ({ row }) => {
        const parseMemory = (mem: string) => {
          if (!mem) return 0
          const match = mem.match(/^(\d+)(.*)$/)
          if (!match) return 0
          const value = parseInt(match[1])
          const unit = match[2]
          if (unit.includes("Ki")) return value / 1024 / 1024
          if (unit.includes("Mi")) return value / 1024
          if (unit.includes("Gi")) return value
          return value / 1024 / 1024 / 1024
        }
        const capacity = parseMemory(row.original.status.capacity?.memory || "0")
        const allocatable = parseMemory(row.original.status.allocatable?.memory || "0")
        return (
          <div className="text-xs">
            <div className="font-medium">
              {allocatable.toFixed(1)}Gi / {capacity.toFixed(1)}Gi
            </div>
            <div className="text-muted-foreground">Allocatable / Capacity</div>
          </div>
        )
      },
    },
    {
      id: "actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => {
        const nodeName = row.original.metadata.name
        const isUnschedulable = row.original.spec.unschedulable

        return (
          <div className="flex items-center justify-end gap-2">
            <Tooltip>
              <TooltipTrigger
                delay={200}
                render={
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    onClick={() => {
                      openPanel({
                        type: "terminal",
                        targetType: "node",
                        appId: clusterId!,
                        appName: cluster?.name || "Cluster",
                        instanceName: nodeName,
                        containerName: "shell",
                        containers: ["shell"],
                      })
                    }}
                  />
                }
              >
                <Terminal />
              </TooltipTrigger>
              <TooltipContent>Terminal</TooltipContent>
            </Tooltip>
            <DropdownMenu>
              <Tooltip>
                <TooltipTrigger
                  delay={200}
                  render={<DropdownMenuTrigger render={<Button variant="ghost" size="icon-sm" />} />}
                >
                  <MoreVertical />
                </TooltipTrigger>
                <TooltipContent>More actions</TooltipContent>
              </Tooltip>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => cordonMutation.mutate({ nodeName, cordon: !isUnschedulable })}>
                  <CircleSlash />
                  {isUnschedulable ? "Uncordon" : "Cordon"}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        )
      }
    },
  ]

  const cordonMutation = useMutation({
    mutationFn: ({ nodeName, cordon }: { nodeName: string; cordon: boolean }) =>
      clustersApi.cordonNode(clusterId!, nodeName, cordon),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["cluster-nodes", clusterId] })
      toast.success("Node status updated")
    },
    onError: (error: AxiosError<{ error: string }>) => {
      toast.error("Action failed", {
        description: error.response?.data?.error || error.message,
      })
    },
  })

  const { data: clusters = [] } = useQuery({
    queryKey: ["clusters-simple"],
    queryFn: () => clustersApi.listSimple(),
  })

  const safeClusters = Array.isArray(clusters) ? clusters : []

  if (clusterLoading) {
    return (
      <div className="flex flex-col flex-1 gap-6">
        <DetailHeroSkeleton showBadge actions={3} />
        <TabsSkeleton count={6} />
        <InfoCardSkeleton fields={6} />
        <StatCardsSkeleton count={4} columnsClassName="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4" />
        <PanelCardSkeleton
          titleWidth="w-36"
          descriptionWidth="w-64"
          contentHeight="h-72"
        />
        <PanelCardSkeleton
          titleWidth="w-28"
          descriptionWidth="w-72"
          contentHeight="h-64"
        />
      </div>
    )
  }

  if (clusterError || !cluster) {
    return (
      <NotFoundPage
        resourceType="Cluster"
        backHref="/clusters"
        backLabel="Back to Clusters"
      />
    )
  }

  const breadcrumbs = [
    { label: "Clusters", icon: ShipWheel, href: "/clusters" },
    {
      label: cluster.name,
      icon: ShipWheel,
      dropdown: safeClusters.length > 1 ? (
        <DropdownMenu>
          <DropdownMenuTrigger render={<Button variant="ghost" size="icon-sm"><ChevronsUpDown /></Button>} />
          <DropdownMenuContent align="start" className="w-fit">
            <DropdownMenuGroup>
              {safeClusters.map(c => (
                <DropdownMenuItem
                  key={c.id}
                  onClick={() => navigate(`/clusters/${c.id}`)}
                >
                  <ShipWheel className="h-4 w-4" />
                  {c.name}
                </DropdownMenuItem>
              ))}
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      ) : undefined
    },
  ]

  const getConnectionStatusBadge = () => {
    const status = cluster.connection_status || "unknown"
    switch (status) {
      case "connected":
        return (
          <ColorBadge color="green">
            Connected
          </ColorBadge>
        )
      case "disconnected":
        return (
          <ColorBadge color="red">
            Disconnected
          </ColorBadge>
        )
      default:
        return (
          <ColorBadge color="gray">
            Unknown
          </ColorBadge>
        )
    }
  }

  const totalCpu = safeNodes.reduce((acc, node) => {
    const cpu = parseInt(node.status.capacity?.cpu || "0")
    return acc + cpu
  }, 0)

  const totalMemory = safeNodes.reduce((acc, node) => {
    const parseMemory = (mem: string) => {
      if (!mem) return 0
      const match = mem.match(/^(\d+)(.*)$/)
      if (!match) return 0
      const value = parseInt(match[1])
      const unit = match[2]
      if (unit.includes("Ki")) return value / 1024 / 1024
      if (unit.includes("Mi")) return value / 1024
      if (unit.includes("Gi")) return value
      return value / 1024 / 1024 / 1024
    }
    return acc + parseMemory(node.status.capacity?.memory || "0")
  }, 0)

  return (
    <div className="flex flex-col flex-1 gap-6">
      <PageHeader items={breadcrumbs} />

      <div className="flex flex-col gap-4">
        <div className="flex justify-between items-start">
          <div className="flex items-center gap-4">
            <div className="p-3 bg-primary/10 rounded-lg text-primary">
              <ShipWheel className="h-8 w-8" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-2xl font-bold tracking-tight">{cluster.name}</h1>
                {getConnectionStatusBadge()}
              </div>
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <span className="font-mono">{cluster.slug}</span>
                <span>•</span>
                {cluster.description ? (
                  <span className="truncate">{cluster.description}</span>
                ) : (
                  <span className="italic">No description</span>
                )}
              </div>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="icon"
              onClick={() => setEditOpen(true)}
            >
              <Pencil />
            </Button>
            <Button
              variant="outline"
              onClick={() => checkConnectivityMutation.mutate()}
              disabled={checkConnectivityMutation.isPending}
            >
              {checkConnectivityMutation.isPending ? (
                <>
                  <Loader2 className="animate-spin" />
                  Checking...
                </>
              ) : (
                <>
                  <Link2 />
                  Check Connection
                </>
              )}
            </Button>
            <Button
              variant="outline"
              className="text-destructive hover:text-destructive hover:bg-destructive/10"
              onClick={() => setDeleteOpen(true)}
            >
              <Trash2 />
              Delete
            </Button>
          </div>
        </div>
      </div>

      {cluster.connection_status === "disconnected" && cluster.connection_status_reason && (
        <Card className="border-destructive/10 bg-destructive/5">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-destructive">
              <AlertCircle className="h-4 w-4" />
              Connection Error
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-xs text-muted-foreground">{cluster.connection_status_reason}</p>
          </CardContent>
        </Card>
      )}

      <Tabs value={activeTab} onValueChange={(v) => setSearchParams({ tab: v }, { replace: true })}>
        <TabsList>
          <TabsTrigger value="overview">
            <Telescope />
            Overview
          </TabsTrigger>
          <TabsTrigger value="nodes">
            <Server />
            Nodes
          </TabsTrigger>
          <TabsTrigger value="extensions">
            <Blocks />
            Extensions
          </TabsTrigger>
          <TabsTrigger value="integrations">
            <GamepadDirectional />
            Integrations
          </TabsTrigger>
          <TabsTrigger value="registries">
            <Warehouse />
            Registries
          </TabsTrigger>
          <TabsTrigger value="certificates">
            <ShieldCheck />
            Certificates
          </TabsTrigger>
          <TabsTrigger value="domains">
            <Globe />
            Domains
          </TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-4 mt-2">
          <Card className="bg-linear-to-b/increasing from-blue-500/5 to-transparent data-[active=true]:bg-transparent">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Info className="h-4 w-4" />
                Cluster Information
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                <div className="space-y-1">
                  <p className="text-xs font-medium text-muted-foreground">Slug</p>
                  <p className="text-sm font-mono">{cluster.slug}</p>
                </div>
                <div className="space-y-1">
                  <p className="text-xs font-medium text-muted-foreground">Status</p>
                  <div className="flex items-center">{getConnectionStatusBadge()}</div>
                </div>
                <div className="space-y-1">
                  <p className="text-xs font-medium text-muted-foreground">Last Checked</p>
                  <div className="flex items-center gap-1.5 text-sm">
                    <Clock className="h-3.5 w-3.5 text-muted-foreground" />
                    <span>{cluster.last_checked_at ? formatDate(cluster.last_checked_at) : "Never"}</span>
                  </div>
                </div>
              </div>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                <div className="space-y-1">
                  <p className="text-xs font-medium text-muted-foreground">API Server</p>
                  <div className="flex items-center gap-2">
                    <p className="text-sm font-mono break-all">{cluster.api_server || "Unavailable"}</p>
                    {cluster.api_server && (
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        className="opacity-0 transition-opacity group-hover/card:opacity-100"
                        onClick={() => {
                          const apiServer = cluster.api_server
                          if (!apiServer) {
                            return
                          }
                          navigator.clipboard.writeText(apiServer)
                          toast.success("API server copied to clipboard")
                        }}
                      >
                        <Copy />
                      </Button>
                    )}
                  </div>
                </div>
                <div className="space-y-1">
                  <p className="text-xs font-medium text-muted-foreground">KubeConfig</p>
                  <div className="flex items-center gap-2">
                    <ColorBadge color={cluster.has_kube_config ? "green" : "red"}>
                      {cluster.has_kube_config ? "Configured" : "Not Configured"}
                    </ColorBadge>
                    {/* <p className="text-sm font-medium">{cluster.has_kube_config ? "Configured" : "Not configured"}</p> */}
                    <Button
                      type="button"
                      variant="outline"
                      onClick={() => setEditKubeConfigOpen(true)}
                    >
                      <Key className="h-4 w-4" />
                      {cluster.has_kube_config ? "Edit KubeConfig" : "Configure KubeConfig"}
                    </Button>
                  </div>
                </div>
                <div className="space-y-1">
                  <p className="text-xs font-medium text-muted-foreground">Gateway Host</p>
                  <div className="flex items-center gap-2">
                    <p className="text-sm font-mono">{cluster.gateway_host || "N/A"}</p>
                    {cluster.gateway_host && (
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        className="opacity-0 transition-opacity group-hover/card:opacity-100"
                        onClick={() => {
                          const gatewayHost = cluster.gateway_host
                          if (!gatewayHost) {
                            return
                          }
                          navigator.clipboard.writeText(gatewayHost)
                          toast.success("Gateway host copied to clipboard")
                        }}
                      >
                        <Copy />
                      </Button>
                    )}
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>

          <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
            <StatCard
              title="Nodes"
              value={safeNodes.length}
              icon={Server}
              description={`${safeNodes.filter(n => n.status.conditions?.find(c => c.type === "Ready")?.status === "True").length} Ready`}
              color="sky"
            />
            <StatCard
              title="Total CPU"
              value={`${totalCpu} Cores`}
              icon={Cpu}
              description="Capacity"
              color="amber"
            />
            <StatCard
              title="Total Memory"
              value={`${totalMemory.toFixed(1)} Gi`}
              icon={MemoryStick}
              description="Capacity"
              color="amber"
            />
          </div>

          <Card className="bg-linear-to-b/increasing from-blue-500/5 to-transparent data-[active=true]:bg-transparent">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <ChartLine className="h-4 w-4" />
                Node Metrics
              </CardTitle>
              <CardDescription>
                Real-time node metrics for all nodes in the cluster. Use the time range selector to adjust the displayed time range.
              </CardDescription>
            </CardHeader>
            <CardContent>
              {safeNodes.length === 0 ? (
                <Empty>
                  <EmptyHeader>
                    <EmptyMedia variant="icon">
                      <Server />
                    </EmptyMedia>
                    <EmptyTitle>No Nodes</EmptyTitle>
                  </EmptyHeader>
                  <EmptyContent>
                    <p className="text-sm text-muted-foreground">No nodes found in this cluster.</p>
                  </EmptyContent>
                </Empty>
              ) : (
                <div className="space-y-6">
                  {safeNodes.map((node) => (
                    <NodeMetricsPanel
                      key={node.metadata.name}
                      clusterId={clusterId!}
                      prometheusAvailable={cluster?.has_prometheus_integration}
                      node={node}
                    />
                  ))}
                </div>
              )}
            </CardContent>
          </Card>

        </TabsContent>

        <TabsContent value="nodes" className="space-y-4 mt-2">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Server className="h-4 w-4" />Cluster Nodes</CardTitle>
              <CardDescription>
                Manage and monitor the nodes in your Kubernetes cluster
              </CardDescription>
            </CardHeader>
            <CardContent>
              {safeNodes && safeNodes.length > 0 ? (<DataTable
                columns={columns}
                data={safeNodes}
                isLoading={clusterLoading}
                searchKey="name"
                searchPlaceholder="Filter nodes..."
              />) : (
                <EmptyState icon={Server} title="No Nodes Available" description="No nodes found in this cluster. This could indicate a connectivity issue or that the cluster is still provisioning." />
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="extensions" className="space-y-4 mt-2">
          <ClusterExtensions clusterId={clusterId!} />
        </TabsContent>

        <TabsContent value="integrations" className="space-y-4 mt-2">
          <ClusterIntegrationsConfig clusterId={clusterId!} />
        </TabsContent>

        <TabsContent value="registries" className="space-y-4 mt-2">
          <ContainerRegistryList scope="cluster" scopeId={clusterId!} />
        </TabsContent>

        <TabsContent value="certificates" className="space-y-4 mt-2">
          <ClusterCertificates clusterId={clusterId!} />
        </TabsContent>

        <TabsContent value="domains" className="space-y-4 mt-2">
          <ClusterDomains clusterId={clusterId!} />
        </TabsContent>
      </Tabs>

      <EditClusterDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        cluster={cluster}
        onSuccess={() => {
          queryClient.invalidateQueries({ queryKey: ["cluster", clusterId] })
        }}
      />

      <EditClusterKubeConfigDialog
        open={editKubeConfigOpen}
        onOpenChange={setEditKubeConfigOpen}
        cluster={cluster}
      />


      <AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Are you sure?</AlertDialogTitle>
            <AlertDialogDescription>
              This will permanently delete the cluster "{cluster.name}" from the platform.
              This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="secondary">Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deleteMutation.mutate()}
              variant="destructive"
            >
              {deleteMutation.isPending ? "Deleting..." : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

export default ClusterDetailPage
