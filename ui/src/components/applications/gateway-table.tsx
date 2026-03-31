import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef } from "@tanstack/react-table"
import { Copy, Edit2, ExternalLink, Globe, GlobeLock, Lock, Network, Plus, Trash2 } from "lucide-react"
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
import { useAuthStore } from "@/stores/auth"
import { ColorBadge } from "../shared/color-badge"

interface GatewayConfigProps {
  app: App
}

export function NetworkConfig({ app }: GatewayConfigProps) {
  const queryClient = useQueryClient()
  const projectRole = useProjectRole()
  const isViewer = projectRole === 'viewer'
  const accessToken = useAuthStore((state) => state.accessToken)
  const [isDialogOpen, setIsDialogOpen] = React.useState(false)
  const [editingGateway, setEditingGateway] = React.useState<GatewaySpec | null>(null)
  const [searchQuery, setSearchQuery] = React.useState("")
  const [rowSelection, setRowSelection] = React.useState({})
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [deletingGateway, setDeletingGateway] = React.useState<GatewaySpec | null>(null)
  const [bulkDeleteDialogOpen, setBulkDeleteDialogOpen] = React.useState(false)
  const [selectedGatewayIds, setSelectedGatewayIds] = React.useState<string[]>([])

  const { data: gateways = [], isLoading } = useQuery({
    queryKey: ['app-gateways', app.id],
    queryFn: async () => {
      const response = await appsApi.listGateways(app.id)
      // Transform backend response to match GatewaySpec
      return response.map((gw: any) => ({
        id: gw.id,
        port: gw.port,
        protocol: gw.protocol,
        domain: gw.domain,
        path: gw.path,
        gateway_port: gw.gateway_port,
        service_type: gw.service_type,
        node_port: gw.node_port,
        gateway_ip: gw.gateway_ip,
        internal_address: gw.internal_address,
        exposed: gw.exposed ?? false,
        cert_id: gw.cert_id,
      }))
    }
  })

  // Filter gateways based on search query
  const filteredGateways = React.useMemo(() => {
    if (!searchQuery) return gateways
    const lowQuery = searchQuery.toLowerCase()
    return gateways.filter(g =>
      g.port?.toString().includes(lowQuery) ||
      g.protocol?.toLowerCase().includes(lowQuery) ||
      g.domain?.toLowerCase().includes(lowQuery) ||
      g.path?.toLowerCase().includes(lowQuery) ||
      g.gateway_port?.toString().includes(lowQuery)
    )
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
    return protocol === 'http' || protocol === 'https'
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
          {row.original.protocol === 'https' && <Lock className="h-3 w-3 mr-1" />}
          {row.original.protocol?.toUpperCase()}
        </ColorBadge>
      ),
    },
    {
      accessorKey: "exposed",
      header: "Access",
      cell: ({ row }) => (
        row.original.exposed ? (
          <ColorBadge color="blue" className="gap-1">
            <Globe className="h-3 w-3" />
            Public
          </ColorBadge>
        ) : (
          <ColorBadge color="gray" className="gap-1">
            <GlobeLock className="h-3 w-3" />
            Internal
          </ColorBadge>
        )
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
      id: "external_access",
      header: "External Access",
      cell: ({ row }) => {
        const gw = row.original
        const isHttp = isHttpProtocol(gw.protocol)
        if (isHttp && gw.exposed && gw.domain) {
          return (
            <Button variant="link" className="p-0 h-auto font-mono text-xs" onClick={() => window.open(`${gw.protocol}://${gw.domain}`, '_blank')}>
              {gw.protocol}://{gw.domain}
            </Button>
          )
        }
        if (gw.gateway_port) {
          return <span className="font-mono text-xs">{gw.gateway_port}</span>
        }
        return <span className="text-muted-foreground text-xs">-</span>
      },
    },
    {
      id: "node_port_address",
      header: "NodePort",
      cell: ({ row }) => {
        const gw = row.original
        if (!gw.gateway_ip || !gw.node_port) {
          return <span className="text-muted-foreground text-xs">-</span>
        }
        const addr = `${gw.gateway_ip}:${gw.node_port}`
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
      accessorKey: "path",
      header: "Path",
      cell: ({ row }) => (
        <>
          {isHttpProtocol(row.original.protocol) ? (
            row.original.path || <span className="font-mono text-xs text-muted-foreground">-</span>
          ) : (
            <span className="text-muted-foreground text-xs">-</span>
          )}
        </>
      ),
    },
    {
      id: "actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => (
        <div className="flex items-center justify-end gap-1">
          {/* Quick Access: visible when app is running/updating and protocol is http/https */}
          {(app.status === 'running' || app.status === 'updating') &&
            (row.original.protocol === 'http' || row.original.protocol === 'https') && (
              <Tooltip>
                <TooltipTrigger
                  delay={200}
                  render={
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      onClick={() => {
                        // Set a short-lived cookie so the backend can authenticate the
                        // proxy request without exposing the JWT in the browser address bar.
                        document.cookie = `X-Ketches-Token=${accessToken}; path=/forward; SameSite=Strict; max-age=3600`
                        window.open(
                          `/forward/${row.original.id}/`,
                          '_blank'
                        )
                      }}
                    />
                  }
                >
                  <ExternalLink className="h-4 w-4" />
                </TooltipTrigger>
                <TooltipContent>Quick Access</TooltipContent>
              </Tooltip>
            )}
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
        {!isLoading && gateways.length === 0 ? (
          <EmptyState
            title="No gateways configured"
            description="Add a gateway to expose your application to the network."
            icon={Network}
            actionText={!isViewer ? "Add Gateway" : undefined}
            onAction={!isViewer ? handleAdd : undefined}
            actionIcon={Plus}
          />
        ) : (
          <>
            <div className="flex flex-wrap items-center justify-between gap-4">
              <Input
                className="flex flex-1 max-w-sm min-w-75"
                placeholder="Filter by port, protocol, domain, path..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
              />

              <div className="flex items-center gap-2">
                {Object.keys(rowSelection).length > 0 && !isViewer && (
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
                )}
                {!isViewer && (
                  <Button onClick={handleAdd}>
                    <Plus />
                    Add Gateway
                  </Button>
                )}
              </div>
            </div>
            <DataTable
              columns={gatewayColumns}
              data={filteredGateways}
              isLoading={isLoading}
              rowSelection={rowSelection}
              onRowSelectionChange={setRowSelection}
              hidePagination
            />
          </>
        )}
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
