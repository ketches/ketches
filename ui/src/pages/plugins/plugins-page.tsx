import { useQuery } from "@tanstack/react-query"
import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import {
  Clock,
  Copy,
  Download,
  Image,
  LayoutGrid,
  List as ListIcon,
  Pencil,
  Plus,
  Puzzle,
  Trash2
} from "lucide-react"
import * as React from "react"

import { pluginsApi, type Plugin } from "@/api/plugins"
import { DataTable } from "@/components/data-table/data-table"
import { PageHeader } from "@/components/layout/page-header"
import { CreatePluginDialog } from "@/components/plugins/create-plugin-dialog"
import { DeletePluginDialog } from "@/components/plugins/delete-plugin-dialog"
import { EditPluginDialog } from "@/components/plugins/edit-plugin-dialog"
import { InstalledAppsDialog } from "@/components/plugins/installed-apps-dialog"
import { ColorBadge } from "@/components/shared/color-badge"
import { EmptyState } from "@/components/shared/empty-state"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useDebounce } from "@/hooks/use-debounce"
import { useProjectRole } from "@/hooks/useProjectRole"
import { formatDate } from "@/lib/utils"
import { useProjectStore } from "@/stores/project"
import { toast } from "sonner"

const PLUGINS_VIEW_MODE_KEY = "plugins_view_mode"

export function PluginsPage({ projectId: projectIdProp }: { projectId?: string } = {}) {
  const { activeProjectId: activeProjectIdFromStore } = useProjectStore()
  const activeProjectId = projectIdProp ?? activeProjectIdFromStore
  const projectRole = useProjectRole()
  const isViewer = projectRole === 'viewer'
  const [createDialogOpen, setCreateDialogOpen] = React.useState(false)
  const [editDialogOpen, setEditDialogOpen] = React.useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [editingPlugin, setEditingPlugin] = React.useState<Plugin | null>(null)
  const [deletingPlugin, setDeletingPlugin] = React.useState<Plugin | null>(null)
  const [selectedPlugin, setSelectedPlugin] = React.useState<Plugin | null>(null)

  const [viewMode, setViewMode] = React.useState<"list" | "card">(() => {
    const saved = localStorage.getItem(PLUGINS_VIEW_MODE_KEY)
    return (saved === "list" || saved === "card") ? saved : "list"
  })
  const [searchQuery, setSearchQuery] = React.useState("")
  const debouncedSearch = useDebounce(searchQuery, 300)

  React.useEffect(() => {
    localStorage.setItem(PLUGINS_VIEW_MODE_KEY, viewMode)
  }, [viewMode])

  const [pagination, setPagination] = React.useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })

  const { data: pluginsResponse, isLoading, refetch } = useQuery({
    queryKey: ['plugins', activeProjectId, debouncedSearch, pagination.pageIndex, pagination.pageSize],
    queryFn: () => pluginsApi.listPlugins(activeProjectId!, {
      search: debouncedSearch,
      page: pagination.pageIndex + 1,
      pageSize: pagination.pageSize
    }),
    enabled: !!activeProjectId,
    placeholderData: (previousData) => previousData,
  })

  const plugins = pluginsResponse?.items ?? []
  const paginationInfo = pluginsResponse?.pagination
  const safePlugins = Array.isArray(plugins) ? plugins : []
  const columns: ColumnDef<Plugin>[] = [
    {
      accessorKey: "name",
      header: "Plugin",
      cell: ({ row }) => (
        <div className="flex items-center gap-2">
          <div className="p-1.5 bg-amber-500/10 rounded-md text-amber-600 shrink-0">
            <Puzzle className="h-4 w-4" />
          </div>
          <div className="min-w-0">
            <p className="font-medium text-xs truncate">
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
      accessorKey: "plugin_type",
      header: "Type",
      cell: ({ row }) => {
        const type = row.original.plugin_type
        return (
          <ColorBadge color={type === "init" ? "blue" : "purple"}>
            {type.toUpperCase()}
          </ColorBadge>
        )
      },
    },
    {
      accessorKey: "image",
      header: "Image",
      cell: ({ row }) => (
        <span className="max-w-50 truncate block text-muted-foreground font-mono">
          {row.original.image}
        </span>
      ),
    },
    {
      accessorKey: "install_count",
      header: "Installs",
      cell: ({ row }) => (
        row.original.install_count > 0 ? (
          <Button
            variant="link"
            className="p-0 h-auto font-normal"
            onClick={() => setSelectedPlugin(row.original)}
          >
            {row.original.install_count}
          </Button>
        ) : (
          <span className="text-muted-foreground">0</span>
        )
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
  ]

  if (!isViewer) {
    columns.push({
      id: "actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => (
        <div className="flex items-center justify-end gap-2">
          <Button
            variant="ghost"
            size="icon-sm"
            onClick={(e) => {
              e.stopPropagation()
              setEditingPlugin(row.original)
              setEditDialogOpen(true)
            }}
          >
            <Pencil />
          </Button>
          <Button
            variant="ghost"
            size="icon-sm"
            className="text-destructive hover:text-destructive hover:bg-destructive/10"
            onClick={(e) => {
              e.stopPropagation()
              setDeletingPlugin(row.original)
              setDeleteDialogOpen(true)
            }}
          >
            <Trash2 />
          </Button>
        </div>
      ),
    })
  }

  const breadcrumbs = [
    {
      label: "Plugins",
      icon: Puzzle,
    }
  ]

  const toolbarLeft = (
    <Input
      className="flex flex-1 max-w-sm min-w-75"
      placeholder="Search plugins..."
      value={searchQuery}
      onChange={(e) => setSearchQuery(e.target.value)}
    />
  )

  const toolbarRight = (
    <div className="flex items-center gap-2">
      <Tabs value={viewMode} onValueChange={(v) => setViewMode(v as "list" | "card")} className="w-auto h-7">
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
          Create Plugin
        </Button>
      )}
    </div>
  )

  return (
    <div className="flex flex-col flex-1 gap-6">
      {!projectIdProp && <PageHeader items={breadcrumbs} />}

      {!isLoading && safePlugins.length === 0 && !searchQuery ? (
        <EmptyState
          title="No plugins found"
          description="Get started by creating your first plugin."
          icon={Puzzle}
          actionText={!isViewer ? "Create Plugin" : undefined}
          onAction={!isViewer ? () => setCreateDialogOpen(true) : undefined}
          actionIcon={!isViewer ? Plus : undefined}
        />
      ) : (
        <>
          {!projectIdProp && (
            <div className="flex items-center justify-between">
              <div>
                <h1 className="text-2xl font-bold">Plugins</h1>
                <p className="text-sm text-muted-foreground mt-1">
                  Manage system plugins and sidecars
                </p>
              </div>
            </div>
          )}

          <DataTable
            columns={columns}
            data={safePlugins}
            viewMode={viewMode}
            onRefresh={refetch}
            manualPagination
            totalCount={paginationInfo?.total || 0}
            pagination={pagination}
            onPaginationChange={setPagination}
            leftActions={() => toolbarLeft}
            toolbarActions={() => toolbarRight}
            renderCard={(plugin) => (
              <Card
                key={plugin.id}
                className="group/card hover:shadow-md transition-shadow h-full"
              >
                <CardHeader className="pb-2">
                  <div className="flex items-start justify-between gap-4">
                    <div className="flex items-start gap-3 min-w-0">
                      <Avatar className="h-10 w-10 rounded-lg bg-primary/10 text-primary border-none">
                        <AvatarFallback className="rounded-lg text-lg font-bold">
                          {plugin.name.charAt(0).toUpperCase()}
                        </AvatarFallback>
                      </Avatar>
                      <div className="flex flex-col min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          <CardTitle className="text-base font-semibold truncate">{plugin.name}</CardTitle>
                          <ColorBadge color={plugin.plugin_type === "init" ? "blue" : "purple"} className="text-[10px] px-1.5 py-0 shrink-0">
                            {plugin.plugin_type.toUpperCase()}
                          </ColorBadge>
                        </div>
                        <div className="flex items-center gap-2 text-[10px] text-muted-foreground truncate font-mono">
                          <span>{plugin.slug}</span>
                          <span>•</span>
                          {plugin.description ? (
                            <span className="truncate">{plugin.description}</span>
                          ) : (
                            <span className="italic">No description</span>
                          )}
                        </div>
                      </div>
                    </div>
                    {!isViewer && (
                      <div className="flex items-center gap-1 shrink-0" onClick={(e) => e.stopPropagation()}>
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          onClick={(e) => {
                            e.stopPropagation()
                            setEditingPlugin(plugin)
                            setEditDialogOpen(true)
                          }}
                        >
                          <Pencil />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          className="text-destructive hover:text-destructive hover:bg-destructive/10"

                          onClick={(e) => {
                            e.stopPropagation()
                            setDeletingPlugin(plugin)
                            setDeleteDialogOpen(true)
                          }}
                        >
                          <Trash2 />
                        </Button>
                      </div>
                    )}
                  </div>
                </CardHeader>
                <CardContent className="space-y-4 pt-2">
                  <div className="space-y-2">
                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                      <Image className="h-3.5 w-3.5" />
                      <span className="font-mono">
                        {plugin.image}
                      </span>
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        className="opacity-0 group-hover/card:opacity-100 transition-opacity"
                        onClick={(e) => {
                          e.stopPropagation()
                          navigator.clipboard.writeText(plugin.image)
                          toast.success("Image address copied to clipboard")
                        }}
                      >
                        <Copy />
                      </Button>
                    </div>
                    <div className="flex items-center justify-between text-xs text-muted-foreground">
                      {plugin.install_count > 0 ? (
                        <Button
                          variant="link"
                          className="p-0 h-auto font-normal"
                          onClick={() => setSelectedPlugin(plugin)}
                        >
                          <Download className="h-3.5 w-3.5" />
                          {plugin.install_count} installs
                        </Button>
                      ) : (
                        <Button
                          variant="ghost"
                          disabled
                          className="p-0 h-auto font-normal"
                        >
                          <Download className="h-3.5 w-3.5" />
                          {plugin.install_count} installs
                        </Button>
                      )}
                    </div>
                  </div>
                  <div className="flex items-center justify-between gap-2 text-[10px] text-muted-foreground/60 border-t pt-2">
                    <div className="flex items-center gap-1.5">
                      <Clock className="h-3 w-3" />
                      <span>Created at {formatDate(plugin.created_at)}</span>
                    </div>
                  </div>
                </CardContent>
              </Card>
            )}
          />
        </>
      )}

      <CreatePluginDialog open={createDialogOpen} onOpenChange={setCreateDialogOpen} projectId={activeProjectId!} />

      {editingPlugin && (
        <EditPluginDialog
          open={editDialogOpen}
          projectId={activeProjectId!}
          onOpenChange={(open) => {
            setEditDialogOpen(open)
            if (!open) setTimeout(() => setEditingPlugin(null), 300)
          }}
          plugin={editingPlugin}
        />
      )}

      {deletingPlugin && (
        <DeletePluginDialog
          open={deleteDialogOpen}
          projectId={activeProjectId!}
          onOpenChange={(open) => {
            setDeleteDialogOpen(open)
            if (!open) setTimeout(() => setDeletingPlugin(null), 300)
          }}
          plugin={deletingPlugin}
        />
      )}

      {selectedPlugin && (
        <InstalledAppsDialog
          open={!!selectedPlugin}
          projectId={activeProjectId!}
          onOpenChange={(open) => !open && setSelectedPlugin(null)}
          plugin={selectedPlugin}
        />
      )}
    </div>
  )
}

export default PluginsPage
