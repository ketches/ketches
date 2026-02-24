import { useQuery } from "@tanstack/react-query"
import { type ColumnDef } from "@tanstack/react-table"
import {
  Clock,
  CloudCog,
  Copy,
  Cpu,
  Image,
  Layers,
  LayoutGrid,
  List as ListIcon,
  Pencil,
  Plus,
  RefreshCw,
  Trash2
} from "lucide-react"
import * as React from "react"
import { useNavigate } from "react-router-dom"
import { toast } from "sonner"

import { appsApi, type App } from "@/api/apps"
import { AppActionIconsWrapper } from "@/components/applications/app-action-icons-wrapper"
import { CreateAppDialog } from "@/components/applications/create-app-dialog"
import { EditAppDialog } from "@/components/applications/edit-app-dialog"
import { DataTable } from "@/components/data-table/data-table"
import { ColorBadge } from "@/components/shared/color-badge"
import { EmptyApplicationState } from "@/components/shared/empty-state"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { useDebounce } from "@/hooks/use-debounce"
import { getAppStatusColor } from "@/lib/app-status"

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
}

export function ApplicationList({ envId, envName }: ApplicationListProps) {
  const navigate = useNavigate()
  const [createDialogOpen, setCreateDialogOpen] = React.useState(false)
  const [editDialogOpen, setEditDialogOpen] = React.useState(false)
  const [editingApp, setEditingApp] = React.useState<App | null>(null)
  const [viewMode, setViewMode] = React.useState<"list" | "card">(() => {
    const saved = localStorage.getItem(APPLICATIONS_VIEW_MODE_KEY)
    return (saved === "list" || saved === "card") ? saved : "list"
  })
  const [searchQuery, setSearchQuery] = React.useState("")
  const debouncedSearch = useDebounce(searchQuery, 300)

  React.useEffect(() => {
    localStorage.setItem(APPLICATIONS_VIEW_MODE_KEY, viewMode)
  }, [viewMode])

  const { data: apps = [], isLoading, refetch } = useQuery<App[]>({
    queryKey: ['apps', envId, debouncedSearch],
    queryFn: () => appsApi.list(envId, debouncedSearch),
    enabled: !!envId,
    refetchInterval: 5000,
  })

  const safeApps = Array.isArray(apps) ? apps : []

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
        <div
          className="flex flex-col cursor-pointer group/name"
          onClick={() => navigate(`/applications/${row.original.id}`)}
        >
          <span className="font-medium text-foreground group-hover/name:text-primary transition-colors">{row.original.name}</span>
          <span className="text-xs text-muted-foreground font-mono">{row.original.slug}</span>
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
          <Tooltip>
            <TooltipTrigger>
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
            </TooltipTrigger>
            <TooltipContent>
              <p>Edit</p>
            </TooltipContent>
          </Tooltip>

          <AppActionIconsWrapper appId={row.original.id} />
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
      <Button onClick={() => setCreateDialogOpen(true)}>
        <Plus />
        Create Application
      </Button>
    </div>
  )

  if (isLoading) {
    return (
      <div className="flex flex-col items-center justify-center py-12 gap-4">
        <RefreshCw className="h-8 w-8 animate-spin text-muted-foreground" />
        <p className="text-sm text-muted-foreground">Loading applications...</p>
      </div>
    )
  }

  if (safeApps.length === 0 && !searchQuery) {
    return (
      <EmptyApplicationState
        onAction={() => setCreateDialogOpen(true)}
        environmentName={envName}
        actionDisabled={!envId}
      />
    )
  }

  return (
    <>
      <DataTable
        columns={columns}
        data={safeApps}
        viewMode={viewMode}
        onRefresh={refetch}
        leftActions={() => toolbarLeft}
        toolbarActions={() => toolbarRight}
        renderCard={(app) => (
          <Card
            key={app.id}
            className="group/card hover:shadow-md transition-shadow cursor-pointer h-full"
            onClick={() => navigate(`/applications/${app.id}`)}
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
                      <CardTitle className="text-base font-semibold truncate">{app.name}</CardTitle>
                      <ColorBadge color={getAppStatusColor(app.status)} className="text-[10px] px-1.5 py-0 shrink-0">
                        {app.status.toUpperCase() || "RUNNING"}
                      </ColorBadge>
                    </div>
                    <div className="flex items-center gap-2 text-[10px] text-muted-foreground truncate font-mono">
                      <span>{app.slug}</span>
                      {app.description && (
                        <>
                          <span>•</span>
                          <span className="truncate">{app.description}</span>
                        </>
                      )}
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-1 shrink-0" onClick={(e) => e.stopPropagation()}>
                  <Button
                    variant="ghost"
                    size="icon-xs"
                    className="h-6 w-6 opacity-0 group-hover/card:opacity-100 transition-opacity"
                    onClick={(e) => {
                      e.stopPropagation()
                      setEditingApp(app)
                      setEditDialogOpen(true)
                    }}
                  >
                    <Pencil className="h-3.5 w-3.5" />
                  </Button>
                  <AppActionIconsWrapper appId={app.id} />
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
                    size="icon-xs"
                    className="h-6 w-6 opacity-0 group-hover/card:opacity-100 transition-opacity"
                    onClick={(e) => {
                      e.stopPropagation()
                      navigator.clipboard.writeText(app.container_image)
                      toast.success("Image address copied to clipboard")
                    }}
                  >
                    <Copy className="h-3.5 w-3.5" />
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
          const selectedRows = table.getFilteredSelectedRowModel().rows
          if (selectedRows.length === 0) return null
          return (
            <div className="flex items-center gap-2">
              <Button variant="outline" onClick={() => toast.info("Restarting applications...")}>
                <RefreshCw />
                Restart
              </Button>
              <Button variant="outline" className="text-destructive hover:text-destructive hover:bg-destructive/10" onClick={() => toast.error("Batch delete not implemented")}>
                <Trash2 />
                Delete
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
    </>
  )
}
