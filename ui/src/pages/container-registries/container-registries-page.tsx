import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef } from "@tanstack/react-table"
import {
  LayoutGrid,
  List as ListIcon,
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
import { EmptyRegistryState, EmptyState } from "@/components/shared/empty-state"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { useDebounce } from "@/hooks/use-debounce"
import { cn } from "@/lib/utils"
import { useProjectStore } from "@/stores/project"

const formatDate = (dateString: string) => {
  if (!dateString) return "-"
  const date = new Date(dateString)
  return date.toLocaleString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  })
}

const REGISTRIES_VIEW_MODE_KEY = "registries_view_mode"

export function ContainerRegistriesPage() {
  const queryClient = useQueryClient()
  const { activeProjectId } = useProjectStore()
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

  const { data: registries = [], isLoading, refetch } = useQuery({
    queryKey: ["registries", "project", activeProjectId],
    queryFn: () => containerRegistriesApi.listByProject(activeProjectId!),
    enabled: !!activeProjectId,
  })

  const safeRegistries = Array.isArray(registries) ? registries : []
  const filteredRegistries = React.useMemo(() => {
    if (!debouncedSearch.trim()) return safeRegistries
    const q = debouncedSearch.toLowerCase()
    return safeRegistries.filter(
      (r) =>
        r.name.toLowerCase().includes(q) ||
        r.endpoint.toLowerCase().includes(q) ||
        registryProviderLabels[r.provider].toLowerCase().includes(q)
    )
  }, [safeRegistries, debouncedSearch])

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
        <div className="flex flex-col cursor-default">
          <div className="flex items-center gap-2">
            <span className="font-medium text-foreground">{row.original.name}</span>
            {row.original.is_default && (
              <Star className="h-3.5 w-3.5 text-primary fill-primary shrink-0" />
            )}
          </div>
          <span className="text-xs text-muted-foreground font-mono truncate max-w-[280px]">
            {row.original.endpoint}
          </span>
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
      accessorKey: "scope",
      header: "Scope",
      cell: ({ row }) => (
        <span className="text-muted-foreground capitalize">{row.original.scope}</span>
      ),
    },
    {
      accessorKey: "enabled",
      header: "Status",
      cell: ({ row }) => (
        <span
          className={
            row.original.enabled
              ? "text-muted-foreground"
              : "text-muted-foreground/70 italic"
          }
        >
          {row.original.enabled ? "Enabled" : "Disabled"}
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
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon-sm"
                onClick={() => {
                  setEditingRegistry(row.original)
                  setEditDialogOpen(true)
                }}
              >
                <Pencil />
              </Button>
            </TooltipTrigger>
            <TooltipContent>
              <p>Edit</p>
            </TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon-sm"
                className="text-destructive hover:text-destructive hover:bg-destructive/10"
                onClick={() => {
                  setDeletingRegistry(row.original)
                  setDeleteDialogOpen(true)
                }}
              >
                <Trash2 />
              </Button>
            </TooltipTrigger>
            <TooltipContent>
              <p>Delete</p>
            </TooltipContent>
          </Tooltip>
        </div>
      ),
    },
  ]

  const breadcrumbs = [{ label: "Container Registries", icon: Warehouse }]

  const toolbarLeft = (
    <Input
      className="flex flex-1 max-w-sm min-w-75"
      placeholder="Search registries..."
      value={searchQuery}
      onChange={(e) => setSearchQuery(e.target.value)}
    />
  )

  const toolbarRight = (
    <div className="flex items-center gap-2">
      <Tabs
        value={viewMode}
        onValueChange={(v) => setViewMode(v as "list" | "card")}
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
      <Button onClick={() => setCreateOpen(true)}>
        <Plus />
        Add Registry
      </Button>
    </div>
  )

  if (!activeProjectId) {
    return (
      <div className="flex flex-col flex-1 gap-6">
        <PageHeader items={breadcrumbs} />
        <EmptyState
          title="Select a project"
          description="Select a project to view and manage container registries."
          icon={Warehouse}
        />
      </div>
    )
  }

  return (
    <div className="flex flex-col flex-1 gap-6">
      <PageHeader items={breadcrumbs} />

      {!isLoading && safeRegistries.length === 0 ? (
        <EmptyRegistryState onAction={() => setCreateOpen(true)} />
      ) : (
        <>
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-2xl font-bold">Container Registries</h1>
              <p className="text-sm text-muted-foreground mt-1">
                Manage image registries for builds and deployments
              </p>
            </div>
          </div>

          <DataTable
            columns={columns}
            data={filteredRegistries}
            viewMode={viewMode}
            onRefresh={refetch}
            leftActions={() => toolbarLeft}
            toolbarActions={() => toolbarRight}
            renderCard={(reg) => (
              <Card
                key={reg.id}
                className="group/card hover:shadow-md transition-shadow cursor-pointer h-full"
                onClick={() => {
                  setEditingRegistry(reg)
                  setEditDialogOpen(true)
                }}
              >
                <CardHeader className="pb-2">
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
                            <span
                              className={cn(
                                "text-[10px] px-1.5 py-0 rounded-full border shrink-0",
                                reg.enabled ? "bg-green-50 text-green-700 border-green-200" : "bg-muted text-muted-foreground"
                              )}
                            >
                              {reg.enabled ? "Enabled" : "Disabled"}
                            </span>
                          </div>
                        </div>
                        <div className="flex items-center gap-2 text-[10px] text-muted-foreground truncate font-mono">
                          <span>{reg.endpoint}</span>
                          {reg.description && (
                            <>
                              <span>•</span>
                              <span className="truncate">{reg.description}</span>
                            </>
                          )}
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center gap-2 shrink-0" onClick={(e) => e.stopPropagation()}>
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        className="opacity-0 group-hover/card:opacity-100 transition-opacity shrink-0"
                        onClick={() => {
                          setEditingRegistry(reg)
                          setEditDialogOpen(true)
                        }}
                      >
                        <Pencil />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        className="text-destructive hover:text-destructive hover:bg-destructive/10"
                        onClick={() => {
                          setDeletingRegistry(reg)
                          setDeleteDialogOpen(true)
                        }}
                      >
                        <Trash2 />
                      </Button>
                    </div>
                  </div>
                </CardHeader>
                <CardContent className="pt-2">
                  <div className="flex items-center justify-between gap-2 text-xs text-muted-foreground">
                    <span>{registryProviderLabels[reg.provider]}</span>
                  </div>
                  <div className="flex items-center justify-between gap-2 text-[10px] text-muted-foreground/60 border-t pt-2 mt-2">
                    <span>Created at {formatDate(reg.created_at)}</span>
                  </div>
                </CardContent>
              </Card>
            )}
          />
        </>
      )}

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
                ? `Delete registry "${deletingRegistry.name}"? This may affect build configs that use it.`
                : "Are you sure you want to delete this registry?"}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (deletingRegistry) {
                  deleteMutation.mutate(deletingRegistry.id)
                }
                setDeleteDialogOpen(false)
                setDeletingRegistry(null)
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

export default ContainerRegistriesPage
