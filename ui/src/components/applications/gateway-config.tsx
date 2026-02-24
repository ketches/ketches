import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef } from "@tanstack/react-table"
import { Edit2, Globe, Lock, Network, Plus, Trash2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import type { App, GatewaySpec } from "@/api/apps"
import { appsApi } from "@/api/apps"
import { GatewayDialog } from "@/components/applications/gateway-dialog"
import { DataTable } from "@/components/data-table/data-table"
import { EmptyState } from "@/components/shared/empty-state"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"

interface GatewayConfigProps {
  app: App
}

export function NetworkConfig({ app }: GatewayConfigProps) {
  const queryClient = useQueryClient()
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
        id: gw.ID || gw.id,
        port: gw.Port || gw.port,
        protocol: gw.Protocol || gw.protocol,
        domain: gw.Domain || gw.domain,
        path: gw.Path || gw.path,
        gateway_port: gw.GatewayPort || gw.gateway_port,
        exposed: gw.Exposed ?? gw.exposed ?? false,
        cert_id: gw.CertID || gw.cert_id,
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
        <span className="font-mono text-sm font-medium">{row.original.port}</span>
      ),
    },
    {
      accessorKey: "protocol",
      header: "Protocol",
      cell: ({ row }) => (
        <Badge variant="outline" className="uppercase font-medium">
          {row.original.protocol === 'https' && <Lock className="h-3 w-3 mr-1" />}
          {row.original.protocol?.toUpperCase()}
        </Badge>
      ),
    },
    {
      accessorKey: "exposed",
      header: "Access",
      cell: ({ row }) => (
        row.original.exposed ? (
          <Badge variant="default" className="gap-1">
            <Globe className="h-3 w-3" />
            Public
          </Badge>
        ) : (
          <Badge variant="secondary">Internal</Badge>
        )
      ),
    },
    {
      id: "domain_or_gateway_port",
      header: "Domain / Gateway Port",
      cell: ({ row }) => (
        <span className="font-mono text-sm">
          {isHttpProtocol(row.original.protocol) ? (
            row.original.domain || <span className="text-muted-foreground">-</span>
          ) : (
            row.original.gateway_port || <span className="text-muted-foreground">-</span>
          )}
        </span>
      ),
    },
    {
      accessorKey: "path",
      header: "Path",
      cell: ({ row }) => (
        <span className="font-mono text-sm">
          {isHttpProtocol(row.original.protocol) ? (
            row.original.path || <span className="text-muted-foreground">-</span>
          ) : (
            <span className="text-muted-foreground">-</span>
          )}
        </span>
      ),
    },
    {
      id: "actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => (
        <div className="flex items-center justify-end gap-1">
          <Button
            variant="ghost"
            size="icon-sm"
            onClick={() => handleEdit(row.original)}
          >
            <Edit2 />
          </Button>
          <Button
            variant="ghost"
            size="icon-sm"
            className="text-destructive hover:text-destructive hover:bg-destructive/10"
            onClick={() => handleDelete(row.original)}
            disabled={deleteMutation.isPending}
          >
            <Trash2 />
          </Button>
        </div>
      ),
    },
  ]

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-sm flex items-center gap-2">
            <Network className="h-4 w-4" /> Port Gateways
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8 text-muted-foreground text-sm">
            Loading gateways...
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm flex items-center gap-2">
          <Network className="h-4 w-4" /> Port Gateways
        </CardTitle>
        <CardDescription>Expose your application to the network</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">

        {gateways.length === 0 ? (
          <EmptyState
            title="No gateways configured"
            description="Add a gateway to expose your application to the network."
            icon={Network}
            actionText="Add Gateway"
            onAction={handleAdd}
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
                {Object.keys(rowSelection).length > 0 && (
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
                <Button onClick={handleAdd}>
                  <Plus />
                  Add Gateway
                </Button>
              </div>
            </div>
            <DataTable
              borderless
              columns={gatewayColumns}
              data={filteredGateways}
              rowSelection={rowSelection}
              onRowSelectionChange={setRowSelection}
              hidePagination
            />
          </>
        )}
      </CardContent>

      <GatewayDialog
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
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (deletingGateway) {
                  deleteMutation.mutate(deletingGateway.id)
                }
                setDeleteDialogOpen(false)
                setDeletingGateway(null)
              }}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
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
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (selectedGatewayIds.length > 0) {
                  bulkDeleteMutation.mutate(selectedGatewayIds)
                  setRowSelection({})
                }
                setBulkDeleteDialogOpen(false)
                setSelectedGatewayIds([])
              }}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Card>
  )
}
