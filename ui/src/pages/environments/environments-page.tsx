import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import {
  Clock,
  LayoutGrid,
  List as ListIcon,
  Orbit,
  Pencil,
  Plus,
  ShipWheel,
  Trash2
} from "lucide-react"
import * as React from "react"
import { useNavigate } from "react-router-dom"
import { toast } from "sonner"

import { envsApi, type Env } from "@/api/envs"
import { DataTable } from "@/components/data-table/data-table"
import { CreateEnvironmentDialog } from "@/components/environment/create-environment-dialog"
import { EditEnvironmentDialog } from "@/components/environment/edit-environment-dialog"
import { PageHeader } from "@/components/layout/page-header"
import { ColorBadge } from "@/components/shared/color-badge"
import { EmptyEnvironmentState } from "@/components/shared/empty-state"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useDebounce } from "@/hooks/use-debounce"
import { useProjectRole } from "@/hooks/useProjectRole"
import { useProjectStore } from "@/stores/project"

const formatDate = (dateString: string) => {
  if (!dateString) return "-"
  const date = new Date(dateString)
  return date.toLocaleString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

const ENVIRONMENTS_VIEW_MODE_KEY = "environments_view_mode"

export function EnvironmentsPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [createDialogOpen, setCreateDialogOpen] = React.useState(false)
  const [editDialogOpen, setEditDialogOpen] = React.useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [editingEnv, setEditingEnv] = React.useState<Env | null>(null)
  const [deletingEnv, setDeletingEnv] = React.useState<Env | null>(null)
  const [viewMode, setViewMode] = React.useState<"list" | "card">(() => {
    const saved = localStorage.getItem(ENVIRONMENTS_VIEW_MODE_KEY)
    return (saved === "list" || saved === "card") ? saved : "list"
  })
  const [searchQuery, setSearchQuery] = React.useState("")
  const debouncedSearch = useDebounce(searchQuery, 300)

  const { activeProjectId } = useProjectStore()
  const projectRole = useProjectRole()
  const isViewer = projectRole === 'viewer'

  React.useEffect(() => {
    localStorage.setItem(ENVIRONMENTS_VIEW_MODE_KEY, viewMode)
  }, [viewMode])

  const [pagination, setPagination] = React.useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })

  const { data: envsResponse, isLoading, refetch } = useQuery({
    queryKey: ['envs', activeProjectId, debouncedSearch, pagination.pageIndex, pagination.pageSize],
    queryFn: () => envsApi.list(activeProjectId!, {
      search: debouncedSearch,
      page: pagination.pageIndex + 1,
      pageSize: pagination.pageSize
    }),
    enabled: !!activeProjectId,
  })

  const envs = envsResponse?.items ?? []
  const paginationInfo = envsResponse?.pagination

  const deleteMutation = useMutation({
    mutationFn: (envId: string) => envsApi.delete(envId),
    onSuccess: () => {
      toast.success("Environment deleted", {
        description: "The environment has been successfully deleted",
      })
      queryClient.invalidateQueries({ queryKey: ['envs', activeProjectId] })
      setDeleteDialogOpen(false)
      setDeletingEnv(null)
    },
    onError: (error: any) => {
      toast.error("Failed to delete environment", {
        description: error.response?.data?.error || "An unknown error occurred",
      })
      setDeleteDialogOpen(false)
      setDeletingEnv(null)
    },
  })

  const safeEnvs = Array.isArray(envs) ? envs : []

  const columns: ColumnDef<Env>[] = [
    {
      accessorKey: "name",
      header: "Environment",
      cell: ({ row }) => (
        <div
          className="flex flex-col cursor-pointer group/name"
          onClick={() => navigate(`/environments/${row.original.id}`)}
        >
          <span className="font-medium text-foreground group-hover/name:text-primary transition-colors">{row.original.name}</span>
          <span className="text-xs text-muted-foreground font-mono">{row.original.slug}</span>
        </div>
      ),
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => (
        <ColorBadge color="green">
          {row.original.status || "Active"}
        </ColorBadge>
      ),
    },
    {
      accessorKey: "cluster_namespace",
      header: "Namespace",
      cell: ({ row }) => (
        <span className="font-mono text-xs text-muted-foreground px-1.5 py-0.5 rounded">
          {row.original.cluster_namespace}
        </span>
      ),
    },
    {
      accessorKey: "created_at",
      header: "Created At",
      cell: ({ row }) => (
        <span className="text-muted-foreground">
          {formatDate(row.original.created_at)}
        </span>
      ),
    },
    {
      id: "actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => (
        <div className="flex items-center justify-end gap-2">
          {!isViewer && (
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={(e) => {
                e.stopPropagation()
                setEditingEnv(row.original)
                setEditDialogOpen(true)
              }}
            >
              <Pencil />
            </Button>
          )}
          {!isViewer && (
            <Button
              variant="ghost"
              size="icon-sm"
              className="text-destructive hover:text-destructive hover:bg-destructive/10"
              onClick={(e) => {
                e.stopPropagation()
                setDeletingEnv(row.original)
                setDeleteDialogOpen(true)
              }}
            >
              <Trash2 />
            </Button>
          )}
        </div>
      ),
    },
  ]

  const breadcrumbs = [
    { label: "Environments", icon: Orbit }
  ]

  const toolbarLeft = (
    <Input className="flex flex-1 max-w-sm min-w-75"
      placeholder="Search environments..."
      value={searchQuery}
      onChange={(e) => setSearchQuery(e.target.value)}
    />
  )

  const toolbarRight = (
    <div className="flex items-center gap-2">
      <Tabs value={viewMode} onValueChange={(v) => setViewMode(v as any)} className="w-auto h-7">
        <TabsList>
          <TabsTrigger value="list">
            <ListIcon />
          </TabsTrigger>
          <TabsTrigger value="card">
            <LayoutGrid />
          </TabsTrigger>
        </TabsList>
      </Tabs>
      {!isViewer && (
        <Button onClick={() => setCreateDialogOpen(true)}>
          <Plus />
          Create Environment
        </Button>
      )}
    </div>
  )

  return (
    <div className="flex flex-col flex-1 gap-6">
      <PageHeader items={breadcrumbs} />

      {!isLoading && safeEnvs.length === 0 && !searchQuery ? (
        <EmptyEnvironmentState onAction={!isViewer ? () => setCreateDialogOpen(true) : undefined} />
      ) : (
        <>
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-2xl font-bold">Environments</h1>
              <p className="text-sm text-muted-foreground mt-1">
                Manage your deployment environments
              </p>
            </div>
          </div>

          <DataTable
            columns={columns}
            data={safeEnvs}
            viewMode={viewMode}
            onRefresh={refetch}
            manualPagination
            totalCount={paginationInfo?.total || 0}
            pagination={pagination}
            onPaginationChange={setPagination}
            leftActions={() => toolbarLeft}
            toolbarActions={() => toolbarRight}
            renderCard={(env) => (
              <Card
                key={env.id}
                className="group/card hover:shadow-md transition-shadow cursor-pointer h-full"
                onClick={() => navigate(`/environments/${env.id}`)}
              >
                <CardHeader className="pb-2">
                  <div className="flex items-start justify-between gap-4">
                    <div className="flex items-start gap-3 min-w-0">
                      <Avatar className="h-10 w-10 rounded-lg bg-primary/10 text-primary border-none">
                        <AvatarFallback className="rounded-lg text-lg font-bold">
                          {env.name.charAt(0).toUpperCase()}
                        </AvatarFallback>
                      </Avatar>
                      <div className="flex flex-col flex-1 min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          <CardTitle className="text-base font-semibold truncate">{env.name}</CardTitle>
                          <ColorBadge color="green">
                            {env.status || "Active"}
                          </ColorBadge>
                        </div>
                        <div className="flex items-center gap-2 text-[10px] text-muted-foreground truncate font-mono">
                          <span>{env.slug}</span>
                          {env.description && (
                            <>
                              <span>•</span>
                              <span className="truncate">{env.description}</span>
                            </>
                          )}
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center gap-1 shrink-0" onClick={(e) => e.stopPropagation()}>
                      {!isViewer && (
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          onClick={(e) => {
                            e.stopPropagation()
                            setEditingEnv(env)
                            setEditDialogOpen(true)
                          }}
                        >
                          <Pencil />
                        </Button>
                      )}
                      {!isViewer && (
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          className="text-destructive hover:text-destructive hover:bg-destructive/10"
                          onClick={(e) => {
                            e.stopPropagation()
                            setDeletingEnv(env)
                            setDeleteDialogOpen(true)
                          }}
                        >
                          <Trash2 />
                        </Button>
                      )}
                    </div>
                  </div>
                </CardHeader>
                <CardContent className="space-y-4 pt-2">
                  <div className="space-y-2">
                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                      <ShipWheel className="h-3.5 w-3.5" />
                      <span className="font-mono">
                        {env.cluster_id}
                      </span>
                    </div>
                    <div className="text-xs text-muted-foreground">
                      <span>Namespace: <span className="font-mono bg-muted px-1.5 py-0.5 rounded text-[10px]">{env.cluster_namespace}</span></span>
                    </div>
                  </div>
                  <div className="flex items-center justify-between gap-2 text-[10px] text-muted-foreground/60 border-t pt-2">
                    <div className="flex items-center gap-1.5">
                      <Clock className="h-3 w-3" />
                      <span>Created at {formatDate(env.created_at)}</span>
                    </div>
                  </div>
                </CardContent>
              </Card>
            )}
          />
        </>
      )}

      <CreateEnvironmentDialog open={createDialogOpen} onOpenChange={setCreateDialogOpen} />
      <EditEnvironmentDialog
        open={editDialogOpen}
        onOpenChange={setEditDialogOpen}
        env={editingEnv}
        onSuccess={() => setEditingEnv(null)}
      />
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Environment?</AlertDialogTitle>
            <AlertDialogDescription>
              This will permanently delete the environment "{deletingEnv?.name}" and all its applications.
              This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setDeletingEnv(null)}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deletingEnv && deleteMutation.mutate(deletingEnv.id)}
              disabled={deleteMutation.isPending}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {deleteMutation.isPending ? "Deleting..." : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

export default EnvironmentsPage
