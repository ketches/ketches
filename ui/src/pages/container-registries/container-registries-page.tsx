import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import {
  Clock,
  Copy,
  GalleryVerticalEnd,
  Handbag,
  LayoutGrid,
  Link,
  List as ListIcon,
  Loader2,
  Pencil,
  Plus,
  Star,
  Trash2,
  Warehouse
} from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import {
  containerRegistriesApi,
  registryProviderLabels,
  type ContainerRegistry,
} from "@/api/container-registries"
import { ContainerRegistryDialog } from "@/components/container-registries/container-registry-dialog"
import { DataTable } from "@/components/data-table/data-table"
import { PageHeader } from "@/components/layout/page-header"
import { ColorBadge } from "@/components/shared/color-badge"
import { EmptyState } from "@/components/shared/empty-state"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import type { BreadcrumbItem } from "@/contexts/breadcrumb-state"
import { useDebounce } from "@/hooks/use-debounce"
import { useProjectRole } from "@/hooks/useProjectRole"
import { formatDate } from "@/lib/utils"
import { useAuthStore } from "@/stores/auth"
import { useProjectStore } from "@/stores/project"

const REGISTRIES_VIEW_MODE_KEY = "registries_view_mode"

export function ContainerRegistriesPage({ projectId: projectIdProp }: { projectId?: string } = {}) {
  const queryClient = useQueryClient()
  const { activeProjectId: activeProjectIdFromStore, activeProjectName } = useProjectStore()
  const activeProjectId = projectIdProp ?? activeProjectIdFromStore
  const projectNameToUse = activeProjectName
  const projectRole = useProjectRole()
  const isViewer = projectRole === 'viewer'
  const isAdmin = useAuthStore((state) => state.user?.role === "admin")
  const [createOpen, setCreateOpen] = React.useState(false)
  const [editDialogOpen, setEditDialogOpen] = React.useState(false)
  const [editingRegistry, setEditingRegistry] = React.useState<ContainerRegistry | null>(null)
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [deletingRegistry, setDeletingRegistry] = React.useState<ContainerRegistry | null>(null)
  const [viewMode, setViewMode] = React.useState<"list" | "card">(() => {
    const saved = localStorage.getItem(REGISTRIES_VIEW_MODE_KEY)
    return saved === "list" || saved === "card" ? saved : "list"
  })
  const [searchQuery, setSearchQuery] = React.useState("")
  const debouncedSearch = useDebounce(searchQuery, 300)

  React.useEffect(() => {
    localStorage.setItem(REGISTRIES_VIEW_MODE_KEY, viewMode)
  }, [viewMode])

  const [pagination, setPagination] = React.useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })

  const { data: registriesResponse, isLoading, refetch } = useQuery({
    queryKey: ["registries", "project", activeProjectId, debouncedSearch, pagination.pageIndex, pagination.pageSize],
    queryFn: () => containerRegistriesApi.listByProject(activeProjectId!, {
      search: debouncedSearch,
      page: pagination.pageIndex + 1,
      page_size: pagination.pageSize
    }),
    enabled: !!activeProjectId,
    placeholderData: (previousData) => previousData,
  })

  const registries = registriesResponse?.items ?? []
  const paginationInfo = registriesResponse?.pagination
  const safeRegistries = Array.isArray(registries) ? registries : []

  const deleteMutation = useMutation({
    mutationFn: (id: string) => containerRegistriesApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["registries", "project", activeProjectId],
      })
      toast.success("Registry deleted")
    },
    onError: (err: unknown) => {
      const msg =
        err && typeof err === "object" && "response" in err
          ? (err as { response?: { data?: { error?: string } } }).response?.data
            ?.error
          : null
      toast.error(msg || "Failed to delete registry")
    },
  })

  const columns: ColumnDef<ContainerRegistry>[] = [
    {
      accessorKey: "name",
      header: "Registry",
      cell: ({ row }) => (
        <div className="flex items-center gap-2">
          <div className="p-1.5 bg-indigo-500/10 rounded-md text-indigo-600 shrink-0">
            <Warehouse className="h-4 w-4" />
          </div>
          <div className="min-w-0 gap-2 flex flex-col">
            <span className="font-medium text-foreground flex items-center">{row.original.name}{row.original.is_default && (
              <Star className="ml-2 h-3.5 w-3.5 text-primary fill-primary shrink-0" />
            )}</span>
            <p className="text-xs text-muted-foreground font-mono truncate">
              {row.original.description}
            </p>
          </div>
        </div>
      ),
    },
    {
      accessorKey: "endpoint",
      header: "Endpoint",
      cell: ({ row }) => (
        <div className="flex items-center gap-2">
          <span className="text-xs text-muted-foreground font-mono truncate max-w-100">
            {row.original.endpoint}
          </span>
          <Button
            variant="ghost"
            size="icon-sm"
            className="opacity-0 group-hover/row:opacity-100 transition-opacity"
            onClick={(e) => {
              e.stopPropagation()
              navigator.clipboard.writeText(row.original.endpoint)
              toast.success("Endpoint copied to clipboard")
            }}
          >
            <Copy />
          </Button>
        </div>
      ),
    },
    {
      accessorKey: "provider",
      header: "Provider",
      cell: ({ row }) => (
        <span className="text-muted-foreground">
          {registryProviderLabels[row.original.provider]}
        </span>
      ),
    },
    {
      accessorKey: "enabled",
      header: "Status",
      cell: ({ row }) => (
        <ColorBadge color={row.original.enabled ? "green" : "gray"} className="ml-2" >
          {row.original.enabled ? "Enabled" : "Disabled"}
        </ColorBadge>
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
    ...(isViewer ? [] : [{
      id: "actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }: { row: import('@tanstack/react-table').Row<ContainerRegistry> }) => (
        <div className="flex items-center justify-end gap-2">
          <Tooltip>
            <TooltipTrigger
              delay={200}
              render={(
                <Button
                  variant="ghost"
                  size="icon-sm"
                  onClick={() => {
                    setEditingRegistry(row.original)
                    setEditDialogOpen(true)
                  }}
                />
              )}
            >
              <Pencil />
            </TooltipTrigger>
            <TooltipContent>
              <p>Edit</p>
            </TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger
              delay={200}
              render={(
                <Button
                  variant="ghost"
                  size="icon-sm"
                  className="text-destructive hover:text-destructive hover:bg-destructive/10"
                  onClick={() => {
                    setDeletingRegistry(row.original)
                    setDeleteDialogOpen(true)
                  }}
                />
              )}
            >
              <Trash2 />
            </TooltipTrigger>
            <TooltipContent>
              <p>Delete</p>
            </TooltipContent>
          </Tooltip>
        </div>
      ),
    }]),
  ]

  const breadcrumbs: BreadcrumbItem[] = isAdmin ? [
    { label: "Projects", icon: GalleryVerticalEnd, href: "/projects" },
    { label: projectNameToUse || "Projects", icon: GalleryVerticalEnd, href: `/projects/${activeProjectId}` },
  ] : []
  breadcrumbs.push({ label: "Container Registries", icon: Warehouse })

  const leftToolbar = (
    <Input
      className="flex flex-1 max-w-sm min-w-75"
      placeholder="Search registries..."
      value={searchQuery}
      onChange={(e) => setSearchQuery(e.target.value)}
    />
  )

  const rightToolbar = (
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
      {!isViewer && (
        <Button onClick={() => setCreateOpen(true)}>
          <Plus />
          Add Registry
        </Button>
      )}
    </div>
  )

  if (!activeProjectId) {
    return (
      <div className="flex flex-col flex-1 gap-6">
        {!projectIdProp && <PageHeader items={breadcrumbs} />}
        <EmptyState
          title="Select a project"
          description="Select a project to view and manage container registries."
          icon={Warehouse}
        />
      </div>
    )
  }

  const isEmptyRegistries = !isLoading && safeRegistries.length === 0 && !searchQuery.trim()

  const renderRegistriesTable = (loading: boolean) => (
    <DataTable
      columns={columns}
      data={safeRegistries}
      isLoading={loading}
      viewMode={viewMode}
      onRefresh={refetch}
      manualPagination
      totalCount={paginationInfo?.total || 0}
      pagination={pagination}
      onPaginationChange={setPagination}
      leftToolbar={() => leftToolbar}
      rightToolbar={() => rightToolbar}
      renderCard={(reg) => (
        <Card
          key={reg.id}
          className="group/card hover:shadow-md transition-shadow h-full bg-secondary/10"
        >
          <CardHeader>
            <div className="flex items-start justify-between gap-4">
              <div className="flex items-start gap-3 min-w-0">
                <Avatar className="h-10 w-10 rounded-lg bg-primary/10 text-primary border-none shrink-0">
                  <AvatarFallback className="rounded-lg text-lg font-bold">
                    {reg.name.charAt(0).toUpperCase()}
                  </AvatarFallback>
                </Avatar>
                <div className="flex flex-col min-w-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    <CardTitle className="text-base font-semibold truncate">
                      {reg.name}
                    </CardTitle>
                    <div className="flex items-center gap-2">
                      {reg.is_default && (
                        <Star className="h-3.5 w-3.5 text-primary fill-primary shrink-0" />
                      )}
                      <ColorBadge color={reg.enabled ? "green" : "gray"} >{reg.enabled ? "Enabled" : "Disabled"}</ColorBadge>
                    </div>
                  </div>
                  <div className="flex items-center gap-2 text-[10px] text-muted-foreground truncate font-mono">
                    {reg.description ? (
                      <span className="truncate">{reg.description}</span>
                    ) : (
                      <span className="italic">No description</span>
                    )}
                  </div>
                </div>
              </div>
              <div className="flex items-center gap-2 shrink-0" onClick={(e) => e.stopPropagation()}>
                {!isViewer && (
                  <Tooltip>
                    <TooltipTrigger
                      delay={200}
                      render={
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          onClick={() => {
                            setEditingRegistry(reg)
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
                    <TooltipContent>Edit registry</TooltipContent>
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
                          onClick={() => {
                            setDeletingRegistry(reg)
                            setDeleteDialogOpen(true)
                          }}
                        />
                      }
                    >
                      <div className="flex items-center">
                        <Trash2 />
                        <span className="sr-only">Delete</span>
                      </div>
                    </TooltipTrigger>
                    <TooltipContent>Delete registry</TooltipContent>
                  </Tooltip>
                )}
              </div>
            </div>
          </CardHeader>
          <CardContent className="space-y-4 pt-2">
            <div className="space-y-2">
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <Link className="h-3.5 w-3.5" />
                <span className="font-mono">
                  {reg.endpoint}
                </span>
                <Button variant="ghost" size="icon-sm" className="opacity-0 group-hover/card:opacity-100 transition-opacity" onClick={() => {
                  navigator.clipboard.writeText(reg.endpoint)
                  toast.success("Endpoint copied to clipboard")
                }}
                >
                  <Copy />
                </Button>
              </div>
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <Handbag className="h-3.5 w-3.5" />
                {registryProviderLabels[reg.provider]}
              </div>
            </div>

            <div className="flex items-center justify-between gap-2 text-[10px] text-muted-foreground/60 border-t pt-2">
              <div className="flex items-center gap-1.5">
                <Clock className="h-3 w-3" />
                <span>Created at {formatDate(reg.created_at)}</span>
              </div>
            </div>
          </CardContent>
        </Card>
      )}
    />
  )

  return (
    <div className="flex flex-col flex-1 gap-6">
      {!projectIdProp && <PageHeader items={breadcrumbs} />}

      {!projectIdProp && (
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">Container Registries</h1>
            <p className="text-sm text-muted-foreground mt-1">
              Manage container registries for builds and deployments
            </p>
          </div>
        </div>
      )}

      {isLoading && safeRegistries.length === 0 ? (
        <div className="flex flex-col flex-1 items-center justify-center min-h-100">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      ) : isEmptyRegistries ? (
        <EmptyState
          title="No registries yet"
          description="Add a container registry to provide images for builds and deployments."
          icon={Warehouse}
          actionText={!isViewer ? "Add Registry" : undefined}
          onAction={!isViewer ? () => setCreateOpen(true) : undefined}
          actionIcon={!isViewer ? Plus : undefined}
        />
      ) : renderRegistriesTable(false)}

      <ContainerRegistryDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        scope="project"
        scopeId={activeProjectId}
      />
      <ContainerRegistryDialog
        open={editDialogOpen}
        onOpenChange={(open) => {
          setEditDialogOpen(open)
          if (!open) setEditingRegistry(null)
        }}
        scope="project"
        scopeId={activeProjectId}
        registry={editingRegistry}
      />
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Registry</AlertDialogTitle>
            <AlertDialogDescription>
              {deletingRegistry
                ? `Delete registry "${deletingRegistry.name}"? This may affect build settings that use it.`
                : "Are you sure you want to delete this registry?"}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="secondary">Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (deletingRegistry) {
                  deleteMutation.mutate(deletingRegistry.id)
                }
                setDeleteDialogOpen(false)
                setDeletingRegistry(null)
              }}
              variant="destructive"
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

export default ContainerRegistriesPage
