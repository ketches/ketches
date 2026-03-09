import { clustersApi, type Cluster } from "@/api/clusters"
import { CreateClusterDialog } from "@/components/cluster/create-cluster-dialog"
import { EditClusterDialog } from "@/components/cluster/edit-cluster-dialog"
import { DataTable } from "@/components/data-table/data-table"
import { PageHeader } from "@/components/layout/page-header"
import { ColorBadge } from "@/components/shared/color-badge"
import { EmptyState } from "@/components/shared/empty-state"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import { Clock, LayoutGrid, Link2, List as ListIcon, Loader2, Network, Pencil, Plus, ShipWheel, Trash2 } from "lucide-react"
import * as React from "react"
import { useNavigate } from "react-router-dom"
import { toast } from "sonner"

import { Input } from "@/components/ui/input"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useDebounce } from "@/hooks/use-debounce"
import { formatDate } from "@/lib/utils"
import type { AxiosError } from "axios"

const CLUSTERS_VIEW_MODE_KEY = "clusters_view_mode"

export function ClustersPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [createDialogOpen, setCreateDialogOpen] = React.useState(false)
  const [editDialogOpen, setEditDialogOpen] = React.useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [editingCluster, setEditingCluster] = React.useState<Cluster | null>(null)
  const [deletingCluster, setDeletingCluster] = React.useState<Cluster | null>(null)
  const [viewMode, setViewMode] = React.useState<"list" | "card">(() => {
    const saved = localStorage.getItem(CLUSTERS_VIEW_MODE_KEY)
    return (saved === "list" || saved === "card") ? saved : "card"
  })
  const [searchQuery, setSearchQuery] = React.useState("")
  const debouncedSearch = useDebounce(searchQuery, 300)

  React.useEffect(() => {
    localStorage.setItem(CLUSTERS_VIEW_MODE_KEY, viewMode)
  }, [viewMode])

  const [pagination, setPagination] = React.useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })

  const { data: clustersResponse, refetch, isLoading } = useQuery({
    queryKey: ['clusters', debouncedSearch, pagination.pageIndex, pagination.pageSize],
    queryFn: () => clustersApi.list({
      search: debouncedSearch,
      page: pagination.pageIndex + 1,
      page_size: pagination.pageSize
    }),
    placeholderData: (previousData) => previousData,
  })

  const clusters = clustersResponse?.items ?? []
  const paginationInfo = clustersResponse?.pagination

  const deleteMutation = useMutation({
    mutationFn: (id: string) => clustersApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['clusters'] })
      toast.success("Cluster deleted successfully")
      setDeleteDialogOpen(false)
      setDeletingCluster(null)
    },
    onError: (error: AxiosError<{ error: string }>) => {
      toast.error("Error", {
        description: error.response?.data?.error || "Failed to delete cluster",
      })
    },
  })

  const [testingClusterId, setTestingClusterId] = React.useState<string | null>(null)

  const testConnectionMutation = useMutation({
    mutationFn: (id: string) => clustersApi.checkConnectivity(id),
    onSuccess: () => {
      toast.success("Connection test started", {
        description: "Status will update shortly",
      })
      setTimeout(() => {
        queryClient.invalidateQueries({ queryKey: ['clusters'] })
      }, 2000)
    },
    onError: (error: AxiosError<{ error: string }>) => {
      toast.error("Failed to test connection", {
        description: error.response?.data?.error || error.message,
      })
    },
    onSettled: () => {
      setTestingClusterId(null)
    },
  })

  const safeClusters = Array.isArray(clusters) ? clusters : []

  const columns: ColumnDef<Cluster>[] = [
    {
      accessorKey: "name",
      header: "Cluster",
      cell: ({ row }) => (
        <div className="flex items-center gap-2">
          <div className="p-1.5 bg-sky-500/10 rounded-md text-sky-600 shrink-0">
            <ShipWheel className="h-4 w-4" />
          </div>
          <div className="min-w-0">
            <p className="font-medium text-xs truncate cursor-pointer hover:text-primary transition-colors" onClick={() => navigate(`/clusters/${row.original.id}`)}>
              {row.original.name}
            </p>
            <p className="text-xs text-muted-foreground font-mono truncate">
              {row.original.slug}
            </p>
          </div>
        </div>
      ),
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => (
        <ColorBadge color={row.original.enabled ? "green" : "gray"}>
          {row.original.enabled ? (
            <>
              Active
            </>
          ) : (
            <>
              Disabled
            </>
          )}
        </ColorBadge>
      ),
    },
    {
      accessorKey: "gateway_ip",
      header: "Gateway IP",
      cell: ({ row }) => (
        <span className="font-mono text-xs">
          {row.original.gateway_ip || "-"}
        </span>
      ),
    },
    {
      accessorKey: "created_at",
      header: "Added At",
      cell: ({ row }) => (
        <div className="flex items-center gap-1.5 text-muted-foreground">
          <Clock className="h-3 w-3" />
          <span>{formatDate(row.original.created_at || "")}</span>
        </div>
      ),
    },
    {
      id: "actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => (
        <div className="flex items-center justify-end gap-1">
          <Tooltip>
            <TooltipTrigger
              delay={200}
              render={
                <Button
                  variant="ghost"
                  size="icon-sm"
                  onClick={(e) => {
                    e.stopPropagation()
                    setTestingClusterId(row.original.id)
                    testConnectionMutation.mutate(row.original.id)
                  }}
                  disabled={testingClusterId === row.original.id}
                />
              }
            >
              <div className="flex items-center">
                {testingClusterId === row.original.id ? (
                  <Loader2 className="animate-spin" />
                ) : (
                  <Link2 />
                )}
                <span className="sr-only">Test Connection</span>
              </div>
            </TooltipTrigger>
            <TooltipContent>Test connection</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger
              delay={200}
              render={
                <Button
                  variant="ghost"
                  size="icon-sm"
                  onClick={(e) => {
                    e.stopPropagation()
                    setEditingCluster(row.original)
                    setEditDialogOpen(true)
                  }}
                />
              }
            >
              <div className="flex items-center">
                <Pencil />
                <span className="sr-only">Edit</span>
              </div>
            </TooltipTrigger>
            <TooltipContent>Edit cluster</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger
              delay={200}
              render={
                <Button
                  variant="ghost"
                  size="icon-sm"
                  className="text-destructive hover:text-destructive hover:bg-destructive/10"
                  onClick={(e) => {
                    e.stopPropagation()
                    setDeletingCluster(row.original)
                    setDeleteDialogOpen(true)
                  }}
                  disabled={deleteMutation.isPending}
                />
              }
            >
              <div className="flex items-center">
                <Trash2 />
                <span className="sr-only">Delete</span>
              </div>
            </TooltipTrigger>
            <TooltipContent>Delete cluster</TooltipContent>
          </Tooltip>
        </div>
      ),
    },
  ]

  const breadcrumbs = [{ label: "Clusters", icon: ShipWheel }]

  const toolbarLeft = (
    <Input
      className="flex flex-1 max-w-sm min-w-75"
      placeholder="Search clusters..."
      value={searchQuery}
      onChange={(e) => setSearchQuery(e.target.value)}
    />
  )

  const toolbarRight = (
    <div className="flex items-center gap-2">
      <Tabs
        value={viewMode}
        onValueChange={(v) => {
          const newMode = v as "list" | "card"
          setViewMode(newMode)
          setPagination((prev) => ({
            ...prev,
            pageIndex: 0,
            pageSize: newMode === "card" ? 9 : 10,
          }))
        }}
        className="w-auto h-7"
      >
        <TabsList>
          <TabsTrigger value="list">
            <ListIcon />
          </TabsTrigger>
          <TabsTrigger value="card">
            <LayoutGrid />
          </TabsTrigger>
        </TabsList>
      </Tabs>
      <Button onClick={() => setCreateDialogOpen(true)}>
        <Plus />
        Add Cluster
      </Button>
    </div>
  )

  const isEmptyClusters = !isLoading && safeClusters.length === 0 && !searchQuery.trim()

  const renderClustersTable = (loading: boolean) => (
    <DataTable
      columns={columns}
      data={safeClusters}
      isLoading={loading}
      viewMode={viewMode}
      onRefresh={refetch}
      manualPagination
      totalCount={paginationInfo?.total || 0}
      pagination={pagination}
      onPaginationChange={setPagination}
      leftActions={() => toolbarLeft}
      toolbarActions={() => toolbarRight}
      renderCard={(cluster) => (
        <Card
          key={cluster.id}
          className="group/card hover:shadow-md transition-shadow h-full"
        >
          <CardHeader className="pb-2">
            <div className="flex items-start justify-between gap-4">
              <div className="flex items-start gap-3 min-w-0">
                <Avatar className="h-10 w-10 rounded-lg bg-primary/10 text-primary border-none">
                  <AvatarFallback className="rounded-lg text-lg font-bold">
                    {cluster.name.charAt(0).toUpperCase()}
                  </AvatarFallback>
                </Avatar>
                <div className="flex flex-col min-w-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    <CardTitle className="text-base font-semibold truncate cursor-pointer hover:text-primary transition-colors"
                      onClick={() => navigate(`/clusters/${cluster.id}`)}>{cluster.name}</CardTitle>
                    <ColorBadge
                      color={cluster.enabled ? "green" : "gray"}
                    >
                      {cluster.enabled ? (
                        <>
                          Active
                        </>
                      ) : (
                        <>
                          Disabled
                        </>
                      )}
                    </ColorBadge>
                  </div>
                  <div className="flex items-center gap-2 text-[10px] text-muted-foreground truncate font-mono">
                    <span>{cluster.slug}</span>
                    <span>•</span>
                    {cluster.description ? (
                      <span className="truncate">{cluster.description}</span>
                    ) : (
                      <span className="italic">No description</span>
                    )}
                  </div>
                </div>
              </div>
              <div className="flex items-center gap-1 shrink-0" onClick={(e) => e.stopPropagation()}>
                <Tooltip>
                  <TooltipTrigger
                    delay={200}
                    render={
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        onClick={(e) => {
                          e.stopPropagation()
                          setEditingCluster(cluster)
                          setEditDialogOpen(true)
                        }}
                      />
                    }
                  >
                    <div className="flex items-center">
                      <Pencil />
                      <span className="sr-only">Edit</span>
                    </div>
                  </TooltipTrigger>
                  <TooltipContent>Edit cluster</TooltipContent>
                </Tooltip>
                <Tooltip>
                  <TooltipTrigger
                    delay={200}
                    render={
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        onClick={(e) => {
                          e.stopPropagation()
                          setTestingClusterId(cluster.id)
                          testConnectionMutation.mutate(cluster.id)
                        }}
                        disabled={testingClusterId === cluster.id}
                      />
                    }
                  >
                    <div className="flex items-center">
                      {testingClusterId === cluster.id ? (
                        <Loader2 className="animate-spin" />
                      ) : (
                        <Link2 />
                      )}
                      <span className="sr-only">Test Connection</span>
                    </div>
                  </TooltipTrigger>
                  <TooltipContent>Test connection</TooltipContent>
                </Tooltip>
                <Tooltip>
                  <TooltipTrigger
                    delay={200}
                    render={
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        className="text-destructive hover:text-destructive hover:bg-destructive/10"
                        onClick={(e) => {
                          e.stopPropagation()
                          setDeletingCluster(cluster)
                          setDeleteDialogOpen(true)
                        }}
                        disabled={deleteMutation.isPending}
                      />
                    }
                  >
                    <div className="flex items-center">
                      <Trash2 />
                      <span className="sr-only">Delete</span>
                    </div>
                  </TooltipTrigger>
                  <TooltipContent>Delete cluster</TooltipContent>
                </Tooltip>
              </div>
            </div>
          </CardHeader>
          <CardContent className="space-y-4 pt-2">
            <div className="space-y-2">
              {cluster.gateway_ip && (
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  <Network className="h-3.5 w-3.5" />
                  <span className="font-mono bg-muted px-1.5 py-0.5 rounded text-[10px]">{cluster.gateway_ip}</span>
                </div>
              )}
            </div>

            <div className="flex items-center justify-between gap-2 text-[10px] text-muted-foreground/60 border-t pt-2">
              <div className="flex items-center gap-1.5">
                <Clock className="h-3 w-3" />
                <span>Added at {formatDate(cluster.created_at || "")}</span>
              </div>
            </div>
          </CardContent>
        </Card >
      )}
    />
  )



  return (
    <div className="flex flex-col flex-1 gap-6">
      <PageHeader items={breadcrumbs} />

      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Clusters</h1>
          <p className="text-sm text-muted-foreground mt-1">
            Manage your Kubernetes clusters
          </p>
        </div>
      </div>

      {isLoading && safeClusters.length === 0 ? (
        <div className="flex flex-col flex-1 items-center justify-center min-h-100">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      ) : isEmptyClusters ? (
        <EmptyState
          title="No clusters yet"
          description="Add your first cluster to start deploying workloads."
          icon={ShipWheel}
          actionText="Add Cluster"
          onAction={() => setCreateDialogOpen(true)}
          actionIcon={Plus}
        />
      ) : renderClustersTable(false)}

      <CreateClusterDialog open={createDialogOpen} onOpenChange={setCreateDialogOpen} />
      <EditClusterDialog
        open={editDialogOpen}
        onOpenChange={setEditDialogOpen}
        cluster={editingCluster}
        onSuccess={() => setEditingCluster(null)}
      />
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Cluster?</AlertDialogTitle>
            <AlertDialogDescription>
              This will permanently delete the cluster "{deletingCluster?.name}".
              This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setDeletingCluster(null)}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deletingCluster && deleteMutation.mutate(deletingCluster.id)}
              disabled={deleteMutation.isPending}
              variant="destructive"
            >
              {deleteMutation.isPending ? "Deleting..." : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div >
  )
}

export default ClustersPage
