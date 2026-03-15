import { formatDate } from "@/lib/utils"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  Bookmark,
  ChartLine,
  ChevronsUpDown,
  CircleSlash,
  Cpu,
  HardDrive,
  Info,
  Layers,
  Loader2,
  MemoryStick,
  PaintBucket,
  PcCase,
  ShipWheel,
  Tag,
  Telescope,
  Terminal as TerminalIcon,
  Wrench
} from "lucide-react"
import * as React from "react"
import { useNavigate, useParams, useSearchParams } from "react-router-dom"
import { toast } from "sonner"

import { clustersApi } from "@/api/clusters"
import { NodeAnnotationsEditor } from "@/components/cluster/node-annotations-editor"
import { NodeLabelsEditor } from "@/components/cluster/node-labels-editor"
import { NodeTaintsEditor } from "@/components/cluster/node-taints-editor"
import { NotFoundPage } from "@/components/layout/not-found-page"
import { PageHeader } from "@/components/layout/page-header"
import { ClusterNodeResourceMetrics } from "@/components/monitoring/cluster-node-resource-metrics"
import { MetricsTimeRangeSelector } from "@/components/monitoring/metrics-time-range-selector"
import { useTimeRange } from "@/components/monitoring/use-time-range"
import { ColorBadge } from "@/components/shared/color-badge"
import { EmptyState } from "@/components/shared/empty-state"
import { StatCard } from "@/components/shared/stat-card"
import { Button } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useBottomPanel } from "@/contexts/bottom-panel-context"
import type { AxiosError } from "axios"

export function ClusterNodeDetailPage() {
  const { clusterId, nodeName } = useParams<{ clusterId: string; nodeName: string }>()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const queryClient = useQueryClient()
  const { openPanel } = useBottomPanel()

  const activeTab = searchParams.get("tab") || "overview"
  const [labelsOpen, setLabelsOpen] = React.useState(false)
  const [annotationsOpen, setAnnotationsOpen] = React.useState(false)
  const [taintsOpen, setTaintsOpen] = React.useState(false)
  const { timeRange, setTimeRange, rangeSeconds, step } = useTimeRange()

  const { data: clusters = [] } = useQuery({
    queryKey: ["clusters-simple"],
    queryFn: () => clustersApi.listSimple(),
  })

  const { data: nodes = [] } = useQuery({
    queryKey: ["cluster-nodes", clusterId],
    queryFn: () => clustersApi.listNodes(clusterId!),
    enabled: !!clusterId,
  })

  const safeClusters = Array.isArray(clusters) ? clusters : []
  const safeNodes = Array.isArray(nodes) ? nodes : []

  const { data: cluster } = useQuery({
    queryKey: ["cluster", clusterId],
    queryFn: () => clustersApi.get(clusterId!),
    enabled: !!clusterId,
  })

  const { data: node, isLoading, error } = useQuery({
    queryKey: ["cluster-node", clusterId, nodeName],
    queryFn: () => clustersApi.getNode(clusterId!, nodeName!),
    enabled: !!clusterId && !!nodeName,
    retry: false,
  })

  const cordonMutation = useMutation({
    mutationFn: (cordon: boolean) => clustersApi.cordonNode(clusterId!, nodeName!, cordon),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["cluster-node", clusterId, nodeName] })
      toast.success(node?.spec.unschedulable ? "Node uncordoned" : "Node cordoned")
    },
    onError: (error: AxiosError<{ error: string }>) => {
      toast.error("Action failed", {
        description: error.response?.data?.error || error.message,
      })
    },
  })

  if (isLoading) {
    return (
      <div className="flex flex-col flex-1 items-center justify-center gap-2">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        <p className="text-sm text-muted-foreground">Loading node details...</p>
      </div>
    )
  }

  if (error || !node || !cluster) {
    return (
      <NotFoundPage
        resourceType="Node"
        backHref={`/clusters/${clusterId}`}
        backLabel="Back to Cluster"
      />
    )
  }

  const breadcrumbs = [
    { label: "Clusters", icon: ShipWheel, href: "/clusters" },
    {
      label: cluster.name,
      href: `/clusters/${clusterId}`,
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
    {
      label: nodeName!,
      icon: PcCase,
      dropdown: safeNodes.length > 1 ? (
        <DropdownMenu>
          <DropdownMenuTrigger render={<Button variant="ghost" size="icon-sm"><ChevronsUpDown /></Button>} />
          <DropdownMenuContent align="start" className="w-fit">
            <DropdownMenuGroup>
              {safeNodes.map(n => (
                <DropdownMenuItem
                  key={n.metadata.name}
                  onClick={() => navigate(`/clusters/${clusterId}/nodes/${n.metadata.name}`)}
                >
                  <PcCase className="h-4 w-4" />
                  {n.metadata.name}
                </DropdownMenuItem>
              ))}
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      ) : undefined
    },
  ]

  const isReady = node.status.conditions?.find(c => c.type === "Ready")?.status === "True"
  const isUnschedulable = node.spec.unschedulable

  const getRole = () => {
    const labels = node.metadata.labels || {}
    if ("node-role.kubernetes.io/control-plane" in labels || "node-role.kubernetes.io/master" in labels) {
      return "Control Plane"
    }
    return "Worker"
  }

  const internalIP = node.status.addresses?.find(a => a.type === "InternalIP")?.address
  const hostIP = node.status.addresses?.find(a => a.type === "Hostname")?.address || nodeName

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

  const memoryAllocatable = parseMemory(node.status.allocatable.memory || "0")
  const memoryCapacity = parseMemory(node.status.capacity.memory || "0")
  const storageAllocatable = parseMemory(node.status.allocatable["ephemeral-storage"] || "0")
  const storageCapacity = parseMemory(node.status.capacity["ephemeral-storage"] || "0")

  return (
    <div className="flex flex-col flex-1 gap-6">
      <PageHeader items={breadcrumbs} />

      <div className="flex justify-between items-start">
        <div className="flex items-center gap-4">
          <div className="p-3 bg-primary/10 rounded-lg text-primary">
            <PcCase className="h-8 w-8" />
          </div>
          <div>
            <h1 className="text-2xl font-bold tracking-tight">{nodeName}</h1>
            <p className="text-sm font-mono text-muted-foreground">{getRole()}</p>
          </div>
        </div>
        <div className="flex flex-col items-end gap-2">
          <div className="flex gap-2">
            {isUnschedulable && (
              <ColorBadge color="yellow">SchedulingDisabled</ColorBadge>
            )}
            <ColorBadge color={isReady ? "green" : "red"}>
              {isReady ? "Ready" : "NotReady"}
            </ColorBadge>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              onClick={() => cordonMutation.mutate(!isUnschedulable)}
              disabled={cordonMutation.isPending}
            >
              <CircleSlash />
              {isUnschedulable ? "Uncordon" : "Cordon"}
            </Button>
            <Button
              variant="outline"
              onClick={() => {
                openPanel({
                  type: "terminal",
                  targetType: "node",
                  appId: clusterId!,
                  appName: cluster.name,
                  instanceName: nodeName!,
                  containerName: "shell",
                  containers: ["shell"],
                })
              }}
            >
              <TerminalIcon />
              Terminal
            </Button>
          </div>
        </div>
      </div>

      <Tabs value={activeTab} onValueChange={(v) => setSearchParams({ tab: v }, { replace: true })}>
        <TabsList>
          <TabsTrigger value="overview">
            <Telescope />
            Overview
          </TabsTrigger>
          <TabsTrigger value="labels">
            <Tag />
            Labels
          </TabsTrigger>
          <TabsTrigger value="annotations">
            <Bookmark />
            Annotations
          </TabsTrigger>
          <TabsTrigger value="taints">
            <PaintBucket />
            Taints
          </TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-4 mt-2">
          <Card className="bg-linear-to-b/increasing from-blue-500/5 to-transparent data-[active=true]:bg-transparent">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Info className="h-4 w-4" />Node Information</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-1 lg:grid-cols-3">
                <div>
                  <p className="text-xs font-medium text-muted-foreground">Internal IP</p>
                  <p className="text-sm font-mono">{internalIP || "N/A"}</p>
                </div>
                <div>
                  <p className="text-xs font-medium text-muted-foreground">Host Name</p>
                  <p className="text-sm font-mono">{hostIP}</p>
                </div>
                <div>
                  <p className="text-xs font-medium text-muted-foreground">Kubelet Version</p>
                  <p className="text-sm font-mono">{node.status.nodeInfo.kubeletVersion}</p>
                </div>
                <div>
                  <p className="text-xs font-medium text-muted-foreground">OS Image</p>
                  <p className="text-sm">{node.status.nodeInfo.osImage}</p>
                </div>
                <div>
                  <p className="text-xs font-medium text-muted-foreground">Operating System</p>
                  <p className="text-sm capitalize">{node.status.nodeInfo.operatingSystem} ({node.status.nodeInfo.architecture})</p>
                </div>
                <div>
                  <p className="text-xs font-medium text-muted-foreground">Kernel Version</p>
                  <p className="text-sm font-mono">{node.status.nodeInfo.kernelVersion}</p>
                </div>
                <div>
                  <p className="text-xs font-medium text-muted-foreground">Container Runtime</p>
                  <p className="text-sm font-mono">{node.status.nodeInfo.containerRuntimeVersion}</p>
                </div>
                <div>
                  <p className="text-xs font-medium text-muted-foreground">Created At</p>
                  <p className="text-sm">
                    {formatDate(node.metadata.creationTimestamp)}
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            <StatCard
              title="CPU"
              value={`${node.status.allocatable.cpu} / ${node.status.capacity.cpu}`}
              icon={Cpu}
              description="Allocatable / Capacity"
              color="amber"
            />
            <StatCard
              title="Memory"
              value={`${memoryAllocatable.toFixed(1)}Gi / ${memoryCapacity.toFixed(1)}Gi`}
              icon={MemoryStick}
              description="Allocatable / Capacity"
              color="amber"
            />
            <StatCard
              title="Storage"
              value={`${storageAllocatable.toFixed(1)}Gi / ${storageCapacity.toFixed(1)}Gi`}
              icon={HardDrive}
              description="Allocatable / Capacity"
              color="amber"
            />
            <StatCard
              title="Pods"
              value={node.status.allocatable.pods}
              icon={Layers}
              description="Pod Capacity"
              color="sky"
            />
          </div>

          <Card className="bg-linear-to-b/increasing from-blue-500/5 to-transparent data-[active=true]:bg-transparent">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <ChartLine className="h-4 w-4" />Node Resource Usage</CardTitle>
              <CardAction>
                <MetricsTimeRangeSelector value={timeRange} onChange={setTimeRange} />
              </CardAction>
            </CardHeader>
            <CardContent>
              <ClusterNodeResourceMetrics
                clusterId={clusterId!}
                nodeName={nodeName!}
                nodeIp={internalIP}
                timeRange={timeRange}
                rangeSeconds={rangeSeconds}
                step={step}
              />
            </CardContent>
          </Card>

        </TabsContent>

        <TabsContent value="labels" className="space-y-4 mt-2">
          <Card>
            <CardHeader className="flex flex-row justify-between pb-2">
              <CardTitle className="text-sm font-medium flex items-center gap-2">
                <Tag className="h-4 w-4" />Labels</CardTitle>
              <Button variant="outline" onClick={() => setLabelsOpen(true)}>
                <Wrench />
                Manage Labels
              </Button>
            </CardHeader>
            <CardContent>
              {Object.entries(node.metadata.labels).length === 0 ? (
                <EmptyState
                  title="No labels applied to this node."
                  description="Labels are used to add metadata to a node."
                  icon={Tag}
                  actionIcon={Wrench}
                  actionText="Manage Labels"
                  onAction={() => setLabelsOpen(true)}
                />
              ) : (
                <div className="flex flex-wrap gap-2">
                  {Object.entries(node.metadata.labels).map(([key, value]) => (
                    <div key={key} className="bg-muted px-2 py-1 rounded border text-xs flex items-center gap-2 max-w-full">
                      <span className="text-muted-foreground shrink-0">{key}:</span>
                      <span className="font-medium truncate">{value}</span>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="annotations" className="space-y-4 mt-2">
          <Card>
            <CardHeader className="flex flex-row justify-between pb-2">
              <CardTitle className="text-sm font-medium flex items-center gap-2">
                <Bookmark className="h-4 w-4" />Annotations</CardTitle>
              <Button variant="outline" onClick={() => setAnnotationsOpen(true)}>
                <Wrench />
                Manage Annotations
              </Button>
            </CardHeader>
            <CardContent>
              {Object.entries(node.metadata.annotations).length === 0 ? (
                <EmptyState
                  title="No annotations applied to this node."
                  description="Annotations are used to add metadata to a node."
                  icon={Bookmark}
                  actionIcon={Wrench}
                  actionText="Manage Annotations"
                  onAction={() => setAnnotationsOpen(true)}
                />
              ) : (
                <div className="flex flex-wrap gap-2">
                  {Object.entries(node.metadata.annotations).map(([key, value]) => (
                    <div key={key} className="bg-muted px-2 py-1 rounded border text-xs flex items-center gap-2 max-w-full">
                      <span className="text-muted-foreground shrink-0">{key}:</span>
                      <span className="font-medium truncate">{value}</span>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="taints" className="space-y-4 mt-2">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm font-medium flex items-center gap-2">
                <PaintBucket className="h-4 w-4" />Taints</CardTitle>
              {node.spec.taints && node.spec.taints.length > 0 && (
                <Button variant="outline" onClick={() => setTaintsOpen(true)}>
                  <Wrench />
                  Manage Taints
                </Button>
              )}
            </CardHeader>
            <CardContent>
              {!node.spec.taints || node.spec.taints.length === 0 ? (
                <EmptyState
                  title="No taints applied to this node."
                  description="Taints are used to mark nodes as unschedulable or to restrict the scheduling of pods to certain nodes."
                  icon={PaintBucket}
                  actionText="Manage Taints"
                  actionIcon={Wrench}
                  onAction={() => setTaintsOpen(true)}
                />
              ) : (
                <div className="space-y-2">
                  {node.spec.taints.map((taint, i) => (
                    <div key={i} className="flex items-center gap-4 p-3 border rounded">
                      <div className="flex-1">
                        <div className="font-medium text-sm">{taint.key}</div>
                        <div className="text-xs text-muted-foreground">{taint.value || "No value"}</div>
                      </div>
                      <ColorBadge color="orange">{taint.effect}</ColorBadge>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      <NodeLabelsEditor
        open={labelsOpen}
        onOpenChange={setLabelsOpen}
        clusterId={clusterId!}
        nodeName={nodeName!}
        labels={node.metadata.labels}
      />

      <NodeAnnotationsEditor
        open={annotationsOpen}
        onOpenChange={setAnnotationsOpen}
        clusterId={clusterId!}
        nodeName={nodeName!}
        annotations={node.metadata.annotations}
      />

      <NodeTaintsEditor
        open={taintsOpen}
        onOpenChange={setTaintsOpen}
        clusterId={clusterId!}
        nodeName={nodeName!}
        taints={node.spec.taints}
      />
    </div>
  )
}
