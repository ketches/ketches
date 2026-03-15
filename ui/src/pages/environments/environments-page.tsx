import { formatDate } from "@/lib/utils"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import {
  BrickWall,
  Clock,
  Copy,
  GalleryVerticalEnd,
  LayoutGrid,
  List as ListIcon,
  Loader2,
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
import { EmptyEnvironmentState } from "@/components/shared/empty-state"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import type { BreadcrumbItem } from "@/contexts/breadcrumb-state"
import { useDebounce } from "@/hooks/use-debounce"
import { useProjectRole } from "@/hooks/useProjectRole"
import { useAuthStore } from "@/stores/auth"
import { useProjectStore } from "@/stores/project"

const ENVIRONMENTS_VIEW_MODE_KEY = "environments_view_mode"

export function EnvironmentsPage({ projectId: projectIdProp }: { projectId?: string } = {}) {
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

  const { activeProjectId: activeProjectIdFromStore, activeProjectName } = useProjectStore()
  const activeProjectId = projectIdProp ?? activeProjectIdFromStore
  const projectNameToUse = activeProjectName
  const projectRole = useProjectRole()
  const isViewer = projectRole === 'viewer'
  const isAdmin = useAuthStore((state) => state.user?.role === "admin")

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
      page_size: pagination.pageSize
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
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
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
        <div className="flex items-center gap-2">
          <div className="p-1.5 bg-green-500/10 rounded-md text-green-600 shrink-0">
            <Orbit className="h-4 w-4" />
          </div>
          <div className="min-w-0">
            <p className="font-medium text-xs truncate cursor-pointer hover:text-primary transition-colors" onClick={() => navigate(`/environments/${row.original.id}`)}>
              {row.original.name}
            </p>
            <p className="text-xs text-muted-foreground font-mono truncate">
              {row.original.slug}
            </p>
          </div>
        </div >
      ),
    },
    {
      accessorKey: "cluster",
      header: "Cluster",
      cell: ({ row }) => (
        <div className="flex items-center gap-1.5 text-muted-foreground">
          <ShipWheel className="h-3 w-3" />
          <span>{row.original.cluster_name}</span>
        </div>
      ),
    },
    {
      accessorKey: "cluster_namespace",
      header: "Namespace",
      cell: ({ row }) => (
        <div className="flex items-center gap-2">
          <span className="truncate block text-muted-foreground font-mono">
            {row.original.cluster_namespace}
          </span>
          <Button
            variant="ghost"
            size="icon-sm"
            className="opacity-0 group-hover/row:opacity-100 transition-opacity"
            onClick={(e) => {
              e.stopPropagation()
              navigator.clipboard.writeText(row.original.cluster_namespace)
              toast.success("Image address copied to clipboard")
            }}
          >
            <Copy />
          </Button>
        </div>
      ),
    },
    {
      accessorKey: "created_at",
      header: "Created At",
      cell: ({ row }) => (
        <div className="flex items-center gap-1.5 text-muted-foreground">
          <Clock className="h-3 w-3" />
          <span>{formatDate(row.original.created_at)}</span>
        </div>
      ),
    },
    {
      id: "actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => (
        <div className="flex items-center justify-end gap-2">
          {!isViewer && (
            <Tooltip>
              <TooltipTrigger
                delay={200}
                render={
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    onClick={(e) => {
                      e.stopPropagation()
                      setEditingEnv(row.original)
                      setEditDialogOpen(true)
                    }}
                  />
                }
              >
                <Pencil />
              </TooltipTrigger>
              <TooltipContent>
                <p>Edit environment</p>
              </TooltipContent>
            </Tooltip>
          )}
          {!isViewer && (
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
                      setDeletingEnv(row.original)
                      setDeleteDialogOpen(true)
                    }}
                  />
                }
              >
                <Trash2 />
              </TooltipTrigger>
              <TooltipContent>
                <p>Delete environment</p>
              </TooltipContent>
            </Tooltip>
          )}
        </div>
      ),
    },
  ]
  const breadcrumbs: BreadcrumbItem[] = isAdmin ? [
    { label: "Projects", icon: GalleryVerticalEnd, href: "/projects" },
    { label: projectNameToUse || "Projects", icon: GalleryVerticalEnd, href: `/projects/${activeProjectId}` },
  ] : []
  breadcrumbs.push({ label: "Environments", icon: Orbit })


  const leftToolbar = (
    <Input className="flex flex-1 max-w-sm min-w-75"
      placeholder="Search environments..."
      value={searchQuery}
      onChange={(e) => setSearchQuery(e.target.value)}
    />
  )

  const rightToolbar = (
    <div className="flex items-center gap-2">
      <Tabs value={viewMode} onValueChange={(v) => {
        const newMode = v as "list" | "card"
        setViewMode(newMode)
        setPagination((prev) => ({
          ...prev,
          pageIndex: 0,
          pageSize: newMode === "card" ? 9 : 10,
        }))
      }} className="w-auto h-7">
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
      {!projectIdProp && <PageHeader items={breadcrumbs} />}
      {!projectIdProp && (
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">Environments</h1>
            <p className="text-sm text-muted-foreground mt-1">
              Manage your deployment environments
            </p>
          </div>
        </div>
      )}

      {isLoading && safeEnvs.length === 0 ? (
        <div className="flex flex-col flex-1 items-center justify-center min-h-100">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      ) : !isLoading && safeEnvs.length === 0 && !searchQuery.trim() ? (
        <EmptyEnvironmentState onAction={!isViewer ? () => setCreateDialogOpen(true) : undefined} />
      ) : (
        <DataTable
          columns={columns}
          data={safeEnvs}
          isLoading={isLoading}
          viewMode={viewMode}
          onRefresh={refetch}
          manualPagination
          totalCount={paginationInfo?.total || 0}
          pagination={pagination}
          onPaginationChange={setPagination}
          leftToolbar={() => leftToolbar}
          rightToolbar={() => rightToolbar}
          renderCard={(env) => (
            <Card
              key={env.id}
              className="group/card hover:shadow-md transition-shadow h-full bg-secondary/10"
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
                        <CardTitle className="text-base font-semibold truncate cursor-pointer hover:text-primary transition-colors"
                          onClick={() => navigate(`/environments/${env.id}`)}>{env.name}</CardTitle>
                      </div>
                      <div className="flex items-center gap-2 text-[10px] text-muted-foreground truncate font-mono">
                        <span>{env.slug}</span>
                        <span>•</span>
                        {env.description ? (
                          <span className="truncate">{env.description}</span>
                        ) : (
                          <span className="italic">No description</span>
                        )}
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center gap-1 shrink-0" onClick={(e) => e.stopPropagation()}>
                    {!isViewer && (
                      <Tooltip>
                        <TooltipTrigger
                          delay={200}
                          render={
                            <Button
                              variant="ghost"
                              size="icon-sm"
                              onClick={(e) => {
                                e.stopPropagation()
                                setEditingEnv(env)
                                setEditDialogOpen(true)
                              }}
                            />
                          }
                        >
                          <Pencil />
                        </TooltipTrigger>
                        <TooltipContent>Edit environment</TooltipContent>
                      </Tooltip>
                    )}
                    {!isViewer && (
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
                                setDeletingEnv(env)
                                setDeleteDialogOpen(true)
                              }}
                            />
                          }
                        >
                          <Trash2 />
                        </TooltipTrigger>
                        <TooltipContent>Delete environment</TooltipContent>
                      </Tooltip>
                    )}
                  </div>
                </div>
              </CardHeader>
              <CardContent className="space-y-4 pt-2">
                <div className="space-y-2">
                  <div className="flex items-center gap-2 text-xs text-muted-foreground">
                    <ShipWheel className="h-3.5 w-3.5" />
                    <span className="font-mono">
                      {env.cluster_name}
                    </span>
                  </div>
                  <div className="flex items-center gap-2 text-xs text-muted-foreground">
                    <BrickWall className="h-3.5 w-3.5" />
                    <span className="font-mono">{env.cluster_namespace}</span>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      className="opacity-0 group-hover/card:opacity-100 transition-opacity"
                      onClick={(e) => {
                        e.stopPropagation()
                        navigator.clipboard.writeText(env.cluster_namespace)
                        toast.success("Namespace copied to clipboard")
                      }}
                    >
                      <Copy />
                    </Button>
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
      )
      }

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

export default EnvironmentsPage
