import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef, type PaginationState, type RowSelectionState } from "@tanstack/react-table"
import {
  Box,
  Clock,
  CloudCog,
  Copy,
  Cpu,
  Download,
  Image,
  Layers,
  LayoutGrid,
  List as ListIcon,
  Pencil,
  Plus,
  RefreshCw,
  Trash2,
  Upload
} from "lucide-react"
import * as React from "react"
import { useNavigate } from "react-router-dom"
import { toast } from "sonner"

import { appsApi, type App } from "@/api/apps"
import { AppActionIconsWrapper } from "@/components/applications/app-action-icons-wrapper"
import { CreateAppDialog } from "@/components/applications/create-app-dialog"
import { EditAppDialog } from "@/components/applications/edit-app-dialog"
import { ExportAppsDialog } from "@/components/applications/export-apps-dialog"
import { ImportAppsDialog } from "@/components/applications/import-apps-dialog"
import { DataTable } from "@/components/data-table/data-table"
import { ColorBadge } from "@/components/shared/color-badge"
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
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useDebounce } from "@/hooks/use-debounce"
import { useProjectRole } from "@/hooks/useProjectRole"
import { getAppStatusColor } from "@/lib/app-status"

import { appFavoritesApi } from "@/api/app-favorite"
import { appGroupsApi } from "@/api/app-groups"
import { Star } from 'lucide-react'
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

const APPLICATIONS_VIEW_MODE_KEY = "applications_view_mode"

interface ApplicationListProps {
  envId: string
  envName?: string
  favoritesOnly?: boolean
  hideToolbarActions?: boolean
  // When set, only show apps whose IDs are in this set (used by group view)
  allowedAppIds?: Set<string>
  // When set, shows 'Remove from Group' action instead of 'Add to Group' for this group
  currentGroupId?: string
}

export function ApplicationList({ envId, envName: _envName, favoritesOnly = false, hideToolbarActions, allowedAppIds, currentGroupId }: ApplicationListProps) {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const projectRole = useProjectRole()
  // Hide action buttons for viewers
  const isViewer = projectRole === 'viewer'
  const [createDialogOpen, setCreateDialogOpen] = React.useState(false)
  const [editDialogOpen, setEditDialogOpen] = React.useState(false)
  const [editingApp, setEditingApp] = React.useState<App | null>(null)
  const [importDialogOpen, setImportDialogOpen] = React.useState(false)
  const [exportDialogOpen, setExportDialogOpen] = React.useState(false)
  const [exportAppIds, setExportAppIds] = React.useState<string[]>([])
  const [exportAppId, setExportAppId] = React.useState<string | undefined>(undefined)
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [deleteAppIds, setDeleteAppIds] = React.useState<string[]>([])
  const [rowSelection, setRowSelection] = React.useState<RowSelectionState>({})

  const [viewMode, setViewMode] = React.useState<"list" | "card">(() => {
    const saved = localStorage.getItem(APPLICATIONS_VIEW_MODE_KEY)
    return (saved === "list" || saved === "card") ? saved : "list"
  })
  const [searchQuery, setSearchQuery] = React.useState("")
  const debouncedSearch = useDebounce(searchQuery, 300)

  React.useEffect(() => {
    localStorage.setItem(APPLICATIONS_VIEW_MODE_KEY, viewMode)
  }, [viewMode])

  const [pagination, setPagination] = React.useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })
  const { data: favorites = [], refetch: refetchFavorites } = useQuery({
    queryKey: ['app-favorites', envId],
    queryFn: () => appFavoritesApi.listFavorites(envId),
    enabled: !!envId,
  })
  const favoriteIds = new Set(favorites.map((f: any) => f.app_id))

  const { data: appGroups = [] } = useQuery({
    queryKey: ['app-groups', envId],
    queryFn: () => appGroupsApi.list(envId),
    enabled: !!envId,
  })

  const toggleFavMutation = useMutation({
    mutationFn: (app: App): Promise<void> =>
      favoriteIds.has(app.id)
        ? appFavoritesApi.removeFavorite(app.id)
        : appFavoritesApi.addFavorite(app.id).then(() => undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['app-favorites', envId] })
      queryClient.invalidateQueries({ queryKey: ['apps', envId] })
      toast.success('Favorite updated')
    },
    onError: (error: any) => {
      toast.error('Failed to update favorite', {
        description: error.response?.data?.error || 'An error occurred'
      })
    }
  })

  const removeFromGroupMutation = useMutation({
    mutationFn: ({ groupId, appId }: { groupId: string; appId: string }) =>
      appGroupsApi.removeApp(groupId, appId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['app-groups', envId] })
      queryClient.invalidateQueries({ queryKey: ['apps', envId] })
      toast.success('Removed from group')
    },
    onError: (error: any) => {
      toast.error('Failed to remove from group', {
        description: error.response?.data?.error || 'An error occurred'
      })
    }
  })

  const addToGroupMutation = useMutation({
    mutationFn: async ({ groupId, appId }: { groupId: string; appId: string }) => {
      // Find the current group this app belongs to
      const currentGroup = appGroups.find((g: any) => g.apps?.some((a: any) => a.id === appId))
      if (currentGroup && currentGroup.id !== groupId) {
        await appGroupsApi.removeApp(currentGroup.id, appId)
      }
      return appGroupsApi.addApp(groupId, appId)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['app-groups', envId] })
      queryClient.invalidateQueries({ queryKey: ['apps', envId] })
      toast.success('Moved to group')
    },
    onError: (error: any) => {
      toast.error('Failed to move to group', {
        description: error.response?.data?.error || 'An error occurred'
      })
    }
  })

  const { data: appsResponse, refetch } = useQuery({
    queryKey: ['apps', envId, debouncedSearch, pagination.pageIndex, pagination.pageSize],
    queryFn: () => appsApi.list(envId, {
      search: debouncedSearch,
      page: pagination.pageIndex + 1,
      page_size: pagination.pageSize,
    }),
    enabled: !!envId,
    refetchInterval: 5000,
    placeholderData: (previousData) => previousData,
  })

  const batchDeleteMutation = useMutation({
    mutationFn: async (ids: string[]) => {
      return await appsApi.batchDelete(ids)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['apps'] })
      toast.success("Applications deleted successfully")
      setDeleteDialogOpen(false)
      setRowSelection({})
    },
    onError: (error: any) => {
      toast.error("Failed to delete applications", {
        description: error.response?.data?.error || "An error occurred while deleting applications",
      })
    },
  })

  const apps = appsResponse?.items ?? []
  const paginationInfo = appsResponse?.pagination
  const safeAppsRaw = Array.isArray(apps) ? apps : []
  const safeApps = (() => {
    let result = safeAppsRaw
    if (favoritesOnly) result = result.filter(a => favoriteIds.has(a.id))
    if (allowedAppIds) result = result.filter(a => allowedAppIds.has(a.id))
    return result
  })()

  const handleRefresh = React.useCallback(async () => {
    if (currentGroupId) {
      await appGroupsApi.listSpecificApps(currentGroupId)
      await queryClient.invalidateQueries({ queryKey: ['app-groups', envId] })
      return
    }

    if (favoritesOnly) {
      await refetchFavorites()
      return
    }

    await refetch()
  }, [currentGroupId, envId, favoritesOnly, queryClient, refetch, refetchFavorites])

  const columns: ColumnDef<App>[] = [
    {
      id: "select",
      size: 40,
      header: ({ table }) => (
        <div className="flex items-center px-1">
          <Checkbox
            checked={
              (table.getIsAllPageRowsSelected() ||
                (table.getIsSomePageRowsSelected() ? "mixed" : false)) as any
            }
            onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
            aria-label="Select all"
          />
        </div>
      ),
      cell: ({ row }) => (
        <div className="flex items-center px-1">
          <Checkbox
            checked={row.getIsSelected()}
            onCheckedChange={(value) => row.toggleSelected(!!value)}
            aria-label="Select row"
          />
        </div>
      ),
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: "name",
      header: "Application",
      cell: ({ row }) => (
        <div className="flex items-center gap-2">
          <div className="p-1.5 bg-blue-500/10 rounded-md text-blue-600 shrink-0">
            <Box className="h-4 w-4" />
          </div>
          <div className="min-w-0">
            <p className="font-medium text-xs truncate cursor-pointer hover:text-primary transition-colors" onClick={() => navigate(`/applications/${row.original.id}`)}
            >
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
      cell: ({ row }) => {
        const status = row.original.status
        return (
          <ColorBadge color={getAppStatusColor(status)}>
            {status.toUpperCase()}
          </ColorBadge>
        )
      },
    },
    {
      accessorKey: "app_type",
      header: "Type",
      cell: ({ row }) => (
        <span className="text-muted-foreground">{row.original.app_type || "Deployment"}</span>
      ),
    },
    {
      accessorKey: "container_image",
      header: "Image",
      cell: ({ row }) => (
        <span className="max-w-50 truncate block text-muted-foreground font-mono">
          {row.original.container_image}
        </span>
      ),
    },
    {
      accessorKey: "replicas",
      header: "Replicas",
      cell: ({ row }) => (
        <div className="flex items-center gap-1.5">
          <Layers className="h-3.5 w-3.5 text-muted-foreground" />
          <span>{row.original.replicas}</span>
        </div>
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
            <>
              <Button
                variant="ghost"
                size="icon-sm"
                onClick={(e) => {
                  e.stopPropagation()
                  setEditingApp(row.original)
                  setEditDialogOpen(true)
                }}
              >
                <Pencil />
              </Button>
              <Button
                variant="ghost"
                size="icon-sm"
                onClick={() => { toggleFavMutation.mutate(row.original) }}
              >
                <Star className={`${favoriteIds.has(row.original.id) ? "fill-yellow-400 text-yellow-400" : ""}`} />
              </Button>
              <AppActionIconsWrapper
                appId={row.original.id}
                envId={envId}
                appGroups={appGroups}
                currentGroupId={currentGroupId}
                onMoveToGroup={(groupId) => addToGroupMutation.mutate({ groupId, appId: row.original.id })}
                onRemoveFromGroup={currentGroupId ? () => removeFromGroupMutation.mutate({ groupId: currentGroupId, appId: row.original.id }) : undefined}
              />
            </>
          )}
        </div>
      ),
    },
  ]

  const toolbarLeft = (
    <Input
      className="flex flex-1 max-w-sm min-w-75"
      placeholder="Search applications..."
      value={searchQuery}
      onChange={(e) => setSearchQuery(e.target.value)}
    />
  )

  const toolbarRight = (
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
      {!isViewer && !hideToolbarActions && (
        <>
          <Button onClick={() => setCreateDialogOpen(true)}>
            <Plus />
            Create Application
          </Button>
          <Button variant="outline" onClick={() => setImportDialogOpen(true)}>
            <Upload />
            Import
          </Button>
        </>
      )}
    </div>
  )

  return (
    <>
      <DataTable
        columns={columns}
        data={safeApps}
        viewMode={viewMode}
        onRefresh={handleRefresh}
        manualPagination={true}
        totalCount={paginationInfo?.total || 0}
        pagination={pagination}
        onPaginationChange={setPagination}
        rowSelection={rowSelection}
        onRowSelectionChange={setRowSelection}
        leftActions={() => toolbarLeft}
        toolbarActions={() => toolbarRight}
        renderCard={(app) => (
          <Card
            key={app.id}
            className="group/card hover:shadow-md transition-shadow h-full"
          >
            <CardHeader className="pb-2">
              <div className="flex items-start justify-between gap-4">
                <div className="flex items-start gap-3 min-w-0">
                  <Avatar className="h-10 w-10 rounded-lg bg-primary/10 text-primary border-none">
                    <AvatarFallback className="rounded-lg text-lg font-bold">
                      {app.name.charAt(0).toUpperCase()}
                    </AvatarFallback>
                  </Avatar>
                  <div className="flex flex-col min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <CardTitle className="text-base font-semibold truncate cursor-pointer hover:text-primary transition-colors"
                        onClick={() => navigate(`/applications/${app.id}`)}>{app.name}</CardTitle>
                      <ColorBadge color={getAppStatusColor(app.status)} className="text-[10px] px-1.5 py-0 shrink-0">
                        {app.status.toUpperCase() || "RUNNING"}
                      </ColorBadge>
                    </div>
                    <div className="flex items-center gap-2 text-[10px] text-muted-foreground truncate font-mono">
                      <span>{app.slug}</span>
                      <span>•</span>
                      {app.description ? (
                        <span className="truncate">{app.description}</span>
                      ) : (
                        <span className="italic">No description</span>
                      )}
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-1 shrink-0" onClick={(e) => e.stopPropagation()}>
                  {!isViewer && (
                    <>
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        onClick={(e) => {
                          e.stopPropagation()
                          setEditingApp(app)
                          setEditDialogOpen(true)
                        }}
                      >
                        <Pencil />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        onClick={() => { toggleFavMutation.mutate(app) }}
                      >
                        <Star className={`${favoriteIds.has(app.id) ? "fill-yellow-400 text-yellow-400" : ""}`} />
                      </Button>
                      <AppActionIconsWrapper
                        appId={app.id}
                        envId={envId}
                        appGroups={appGroups}
                        currentGroupId={currentGroupId}
                        onMoveToGroup={(groupId) => addToGroupMutation.mutate({ groupId, appId: app.id })}
                        onRemoveFromGroup={currentGroupId ? () => removeFromGroupMutation.mutate({ groupId: currentGroupId, appId: app.id }) : undefined}
                      />
                    </>
                  )}
                </div>
              </div>
            </CardHeader>
            <CardContent className="space-y-4 pt-2">
              <div className="space-y-2">
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  <Image className="h-3.5 w-3.5" />
                  <span className="font-mono">
                    {app.container_image}
                  </span>
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    className="opacity-0 group-hover/card:opacity-100 transition-opacity"
                    onClick={(e) => {
                      e.stopPropagation()
                      navigator.clipboard.writeText(app.container_image)
                      toast.success("Image address copied to clipboard")
                    }}
                  >
                    <Copy />
                  </Button>
                </div>
                <div className="flex items-center justify-between text-xs text-muted-foreground">
                  <div className="flex items-center gap-2">
                    <CloudCog className="h-3.5 w-3.5" />
                    <span>{app.app_type || "Deployment"}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <Cpu className="h-3.5 w-3.5" />
                    <span>{app.replicas} Replicas</span>
                  </div>
                </div>
              </div>
              <div className="flex items-center justify-between gap-2 text-[10px] text-muted-foreground/60 border-t pt-2">
                <div className="flex items-center gap-1.5">
                  <Clock className="h-3 w-3" />
                  <span>Created at {formatDate(app.created_at)}</span>
                </div>
              </div>
            </CardContent>
          </Card>
        )}
        batchActions={(table) => {
          if (isViewer) return null
          const selectedRows = table.getFilteredSelectedRowModel().rows
          if (selectedRows.length === 0) return null
          return (
            <div className="flex items-center gap-2">
              <Button variant="outline" onClick={() => toast.info("Restarting applications...")}>
                <RefreshCw />
                Restart
              </Button>
              <Button variant="outline" className="text-destructive hover:text-destructive hover:bg-destructive/10" onClick={() => {
                const selectedIds = table.getFilteredSelectedRowModel().rows.map(row => row.original.id)
                setDeleteAppIds(selectedIds)
                setDeleteDialogOpen(true)
              }}>
                <Trash2 />
                Delete
              </Button>
              <Button variant="outline" onClick={() => {
                const selectedIds = table.getFilteredSelectedRowModel().rows.map(row => row.original.id)
                setExportAppIds(selectedIds)
                setExportAppId(undefined)
                setExportDialogOpen(true)
              }}>
                <Download />
                Export
              </Button>
            </div>
          )
        }}
      />

      <CreateAppDialog open={createDialogOpen} onOpenChange={setCreateDialogOpen} />
      <EditAppDialog
        open={editDialogOpen}
        onOpenChange={setEditDialogOpen}
        app={editingApp}
        onSuccess={() => setEditingApp(null)}
      />
      <ImportAppsDialog
        open={importDialogOpen}
        onOpenChange={setImportDialogOpen}
        envId={envId}
        onSuccess={() => refetch()}
      />
      <ExportAppsDialog
        open={exportDialogOpen}
        onOpenChange={setExportDialogOpen}
        envId={envId}
        appIds={exportAppIds}
        appId={exportAppId}
        onSuccess={() => {
          setExportDialogOpen(false)
        }}
      />

      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Applications?</AlertDialogTitle>
            <AlertDialogDescription>
              This action cannot be undone. This will permanently delete the selected {deleteAppIds.length} application(s) and remove all associated resources from the cluster.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => batchDeleteMutation.mutate(deleteAppIds)}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {batchDeleteMutation.isPending ? "Deleting..." : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
