import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef } from "@tanstack/react-table"
import { Copy, Edit2, Globe, GlobeLock, Lock, Network, Plus, Trash2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import type { App, GatewaySpec } from "@/api/apps"
import { appsApi } from "@/api/apps"
import { GatewayEditor } from "@/components/applications/gateway-editor"
import { DataTable } from "@/components/data-table/data-table"
import { EmptyState } from "@/components/shared/empty-state"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { useProjectRole } from "@/hooks/useProjectRole"
import { ColorBadge } from "../shared/color-badge"

interface GatewayConfigProps {
  app: App
}

function getRouteDisplayPath(path?: string) {
  if (!path || path === "/") {
    return ""
  }

  return path.startsWith("/") ? path : `/${path}`
}

function getRouteURL(route: NonNullable<GatewaySpec["routes"]>[number]) {
  const displayPath = getRouteDisplayPath(route.path)
  return `${route.listener_protocol}://${route.host}${displayPath}`
}

function getGatewaySearchText(gateway: GatewaySpec): string {
  return [
    gateway.port,
    gateway.protocol,
    gateway.service_type,
    gateway.internal_address,
    gateway.gateway_host,
    gateway.gateway_port,
    gateway.node_port,
    ...(gateway.routes ?? []).flatMap((route) => [
      route.host,
      route.path,
      route.listener_protocol,
      route.path_match_type,
    ]),
  ].filter(Boolean).join(" ").toLowerCase()
}

export function NetworkConfig({ app }: GatewayConfigProps) {
  const queryClient = useQueryClient()
  const projectRole = useProjectRole()
  const isViewer = projectRole === 'viewer'
  const [isDialogOpen, setIsDialogOpen] = React.useState(false)
  const [editingGateway, setEditingGateway] = React.useState<GatewaySpec | null>(null)
  const [searchQuery, setSearchQuery] = React.useState("")
  const [rowSelection, setRowSelection] = React.useState({})
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [deletingGateway, setDeletingGateway] = React.useState<GatewaySpec | null>(null)
  const [bulkDeleteDialogOpen, setBulkDeleteDialogOpen] = React.useState(false)
  const [selectedGatewayIds, setSelectedGatewayIds] = React.useState<string[]>([])

  const { data: gateways = [], isLoading, refetch } = useQuery({
    queryKey: ['app-gateways', app.id],
    queryFn: async () => {
      return appsApi.listGateways(app.id)
    }
  })

  // Filter gateways based on search query
  const filteredGateways = React.useMemo(() => {
    if (!searchQuery) return gateways
    const lowQuery = searchQuery.toLowerCase()
    return gateways.filter((gateway) => getGatewaySearchText(gateway).includes(lowQuery))
  }, [gateways, searchQuery])

  const deleteMutation = useMutation({
    mutationFn: (gatewayId: string) => {
      return appsApi.deleteGateway(gatewayId)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['app-gateways', app.id] })
      toast.success("Gateway deleted successfully")
    },
    onError: (err: unknown) => {
      const error = err as { response?: { data?: { error?: string } } }
      toast.error("Failed to delete gateway", {
        description: error.response?.data?.error || "Unknown error"
      })
    }
  })

  const bulkDeleteMutation = useMutation({
    mutationFn: async (gatewayIds: string[]) => {
      return Promise.all(gatewayIds.map(id => appsApi.deleteGateway(id)))
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['app-gateways', app.id] })
      toast.success("Gateways deleted successfully")
      setRowSelection({})
    },
    onError: (err: unknown) => {
      const error = err as { response?: { data?: { error?: string } } }
      toast.error("Failed to delete gateways", {
        description: error.response?.data?.error || "Unknown error"
      })
    }
  })

  const handleEdit = (gateway: GatewaySpec) => {
    setEditingGateway(gateway)
    setIsDialogOpen(true)
  }

  const handleAdd = () => {
    setEditingGateway(null)
    setIsDialogOpen(true)
  }

  const handleDelete = (gateway: GatewaySpec) => {
    if (!gateway.id) return
    setDeletingGateway(gateway)
    setDeleteDialogOpen(true)
  }

  const handleDialogSuccess = () => {
    setIsDialogOpen(false)
    setEditingGateway(null)
  }

  const isHttpProtocol = (protocol: string) => {
    return protocol === 'http'
  }

  const renderRouteSummary = (gateway: GatewaySpec) => {
    const routes = gateway.routes ?? []
    if (routes.length === 0) {
      return (
        <ColorBadge color="gray" className="gap-1">
          <GlobeLock className="h-3 w-3" />
          No public routes
        </ColorBadge>
      )
    }

    const visibleRoutes = routes.slice(0, 3)
    const overflowCount = Math.max(routes.length - visibleRoutes.length, 0)

    return (
      <div className="flex max-w-120 flex-wrap items-center gap-1.5">
        {visibleRoutes.map((route) => {
          const label = `${route.host}${getRouteDisplayPath(route.path)}`
          const protocolLabel = route.listener_protocol?.toUpperCase() || "HTTP"
          const icon = route.listener_protocol === "https"
            ? <Lock className="h-3 w-3" />
            : <Globe className="h-3 w-3" />
          const content = (
            <span className="inline-flex min-w-0 max-w-56 items-center gap-1 truncate">
              {icon}
              <span className="shrink-0">{protocolLabel}</span>
              <span className="truncate font-mono">{label}</span>
            </span>
          )

          if (!route.enabled) {
            return (
              <ColorBadge key={route.id ?? `${route.host}-${route.path}`} color="gray" className="gap-1 opacity-55">
                {content}
              </ColorBadge>
            )
          }

          return (
            <Button
              key={route.id ?? `${route.host}-${route.path}`}
              variant="link"
              className="h-6 min-w-0 p-0 text-xs"
              onClick={() => window.open(getRouteURL(route), "_blank")}
            >
              <ColorBadge color={route.listener_protocol === "https" ? "blue" : "green"} className="gap-1">
                {content}
              </ColorBadge>
            </Button>
          )
        })}
        {overflowCount > 0 && (
          <ColorBadge color="gray">+{overflowCount}</ColorBadge>
        )}
      </div>
    )
  }

  const gatewayColumns: ColumnDef<GatewaySpec>[] = [
    {
      id: "select",
      header: ({ table }) => (
        <Checkbox
          checked={table.getIsAllPageRowsSelected()}
          onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
          aria-label="Select all"
        />
      ),
      cell: ({ row }) => (
        <Checkbox
          checked={row.getIsSelected()}
          onCheckedChange={(value) => row.toggleSelected(!!value)}
          aria-label="Select row"
        />
      ),
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: "port",
      header: "Port",
      cell: ({ row }) => (
        <span className="font-mono text-xs font-medium">{row.original.port}</span>
      ),
    },
    {
      accessorKey: "protocol",
      header: "Protocol",
      cell: ({ row }) => (
        <ColorBadge color="gray" >
          {row.original.protocol?.toUpperCase()}
        </ColorBadge>
      ),
    },
    {
      accessorKey: "service_type",
      header: "Service",
      cell: ({ row }) => (
        <ColorBadge color={row.original.service_type === "NodePort" ? "blue" : "gray"}>
          {row.original.service_type || "ClusterIP"}
        </ColorBadge>
      ),
    },
    {
      id: "internal_address",
      header: "Internal Address",
      cell: ({ row }) => {
        const gw = row.original
        const addr = gw.internal_address
        if (!addr) return <span className="text-muted-foreground text-xs">-</span>
        const copyToClipboard = (text: string) => {
          navigator.clipboard.writeText(text).then(() => toast.success('Copied to clipboard'))
        }
        return (
          <div className="flex items-center gap-1 font-mono text-xs">
            <span>{addr}</span>
            <Tooltip>
              <TooltipTrigger
                delay={200}
                render={
                  <Button variant="ghost" className="opacity-0 group-hover/row:opacity-100 transition-opacity" size="icon-sm" onClick={() => copyToClipboard(addr)} />
                }
              >
                <Copy className="h-3 w-3" />
              </TooltipTrigger>
              <TooltipContent>Copy address</TooltipContent>
            </Tooltip>
          </div>
        )
      },
    },
    {
      id: "routes",
      header: "Public Routes",
      cell: ({ row }) => renderRouteSummary(row.original),
    },
    {
      id: "node_port_address",
      header: "NodePort",
      cell: ({ row }) => {
        const gw = row.original
        if (!gw.gateway_host || !gw.node_port) {
          return <span className="text-muted-foreground text-xs">-</span>
        }
        const addr = `${gw.gateway_host}:${gw.node_port}`
        const isHttp = isHttpProtocol(gw.protocol)
        const copyToClipboard = (text: string) => {
          navigator.clipboard.writeText(text).then(() => toast.success('Copied to clipboard'))
        }
        return (
          <div className="flex items-center gap-1 font-mono text-xs">
            {isHttp ? (
              <Button variant="link" className="p-0 h-auto text-xs" onClick={() => window.open(`${gw.protocol}://${addr}`, '_blank')}>{addr}</Button>
            ) : (
              <span>{addr}</span>
            )}
            <Tooltip>
              <TooltipTrigger
                delay={200}
                render={
                  <Button variant="ghost" className="opacity-0 group-hover/row:opacity-100 transition-opacity" size="icon-sm" onClick={() => copyToClipboard(addr)} />
                }
              >
                <Copy className="h-3 w-3" />
              </TooltipTrigger>
              <TooltipContent>Copy address</TooltipContent>
            </Tooltip>
          </div>
        )
      },
    },
    {
      id: "actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => (
        <div className="flex items-center justify-end gap-1">
          {!isViewer && (
            <>
              <Tooltip>
                <TooltipTrigger
                  delay={200}
                  render={
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      onClick={() => handleEdit(row.original)}
                    />
                  }
                >
                  <Edit2 />
                </TooltipTrigger>
                <TooltipContent>Edit Gateway</TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger
                  delay={200}
                  render={
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      className="text-destructive hover:text-destructive hover:bg-destructive/10"
                      onClick={() => handleDelete(row.original)}
                      disabled={deleteMutation.isPending}
                    />
                  }
                >
                  <Trash2 />
                </TooltipTrigger>
                <TooltipContent>Delete Gateway</TooltipContent>
              </Tooltip>
            </>
          )}
        </div>
      ),
    },
  ]

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm flex items-center gap-2">
          <Network className="h-4 w-4" /> Port Gateways
        </CardTitle>
        <CardDescription>Expose your application to the network</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <DataTable
          columns={gatewayColumns}
          data={filteredGateways}
          sourceDataCount={gateways.length}
          isLoading={isLoading}
          sourceEmptyContent={(
            <EmptyState
              title="No gateways configured"
              description="Add a gateway port and optional HTTP routes."
              icon={Network}
              actionText={!isViewer ? "Add Gateway" : undefined}
              onAction={!isViewer ? handleAdd : undefined}
              actionIcon={Plus}
            />
          )}
          useStandaloneEmptyState
          leftToolbar={() => (
            <Input
              className="flex flex-1 max-w-sm min-w-75"
              placeholder="Filter by port, protocol, service, route..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
          )}
          batchActions={() => (
            Object.keys(rowSelection).length > 0 && !isViewer ? (
              <Button
                variant="destructive"
                onClick={() => {
                  const selectedIndices = Object.keys(rowSelection).filter(key => rowSelection[key as keyof typeof rowSelection])
                  const selectedIds = selectedIndices.map(idx => filteredGateways[parseInt(idx)]?.id).filter(Boolean) as string[]

                  setSelectedGatewayIds(selectedIds)
                  setBulkDeleteDialogOpen(true)
                }}
                disabled={bulkDeleteMutation.isPending}
              >
                <Trash2 />
                Delete ({Object.keys(rowSelection).filter(key => rowSelection[key as keyof typeof rowSelection]).length})
              </Button>
            ) : null
          )}
          rightToolbar={() => (
            !isViewer ? (
              <Button onClick={handleAdd}>
                <Plus />
                Add Gateway
              </Button>
            ) : null
          )}
          rowSelection={rowSelection}
          onRowSelectionChange={setRowSelection}
          onRefresh={() => refetch()}
          hidePagination
        />
      </CardContent>

      <GatewayEditor
        app={app}
        gateway={editingGateway}
        open={isDialogOpen}
        onOpenChange={setIsDialogOpen}
        onSuccess={handleDialogSuccess}
      />
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Gateway</AlertDialogTitle>
            <AlertDialogDescription>
              {deletingGateway
                ? `Are you sure you want to delete this gateway? This action cannot be undone.`
                : "Are you sure you want to delete this gateway?"}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="secondary">Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (deletingGateway) {
                  deleteMutation.mutate(deletingGateway.id!)
                }
                setDeleteDialogOpen(false)
                setDeletingGateway(null)
              }}
              variant="destructive"
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      <AlertDialog open={bulkDeleteDialogOpen} onOpenChange={setBulkDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Gateways</AlertDialogTitle>
            <AlertDialogDescription>
              {selectedGatewayIds.length > 0
                ? `Are you sure you want to delete ${selectedGatewayIds.length} gateway(s)? This action cannot be undone.`
                : "Are you sure you want to delete these gateways?"}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="secondary">Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (selectedGatewayIds.length > 0) {
                  bulkDeleteMutation.mutate(selectedGatewayIds)
                  setRowSelection({})
                }
                setBulkDeleteDialogOpen(false)
                setSelectedGatewayIds([])
              }}
              variant="destructive"
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Card>
  )
}
