import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Database, Pencil, Plus, Star, Trash2, Warehouse } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import {
  containerRegistriesApi,
  registryProviderLabels,
  type ContainerRegistry,
  type RegistryScope as ContainerRegistryScope,
} from "@/api/container-registries"
import { ContainerRegistryDialog } from "@/components/container-registries/container-registry-dialog"
import { DataTable } from "@/components/data-table/data-table"
import { EmptyState } from "@/components/shared/empty-state"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import type { AxiosError } from "axios"
import type { ColumnDef } from "@tanstack/react-table"

interface ContainerRegistryListProps {
  scope: ContainerRegistryScope
  scopeId: string
}

export function ContainerRegistryList({ scope, scopeId }: ContainerRegistryListProps) {
  const queryClient = useQueryClient()
  const [showDialog, setShowDialog] = React.useState(false)
  const [editRegistry, setEditRegistry] = React.useState<ContainerRegistry | null>(null)

  const { data: containerRegistries = [] } = useQuery({
    queryKey: ["registries", scope, scopeId],
    queryFn: async () => {
      const res = await (scope === "cluster"
        ? containerRegistriesApi.listByCluster(scopeId)
        : containerRegistriesApi.listByProject(scopeId))
      return res.items
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => containerRegistriesApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["registries", scope, scopeId] })
      toast.success("Container registry deleted")
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error(err?.response?.data?.error || 'Failed to delete registry')
    },
  })

  const columns: ColumnDef<ContainerRegistry>[] = [
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => (
        <div className="flex items-center gap-2">
          {row.original.name}
          {row.original.is_default && (
            <Badge variant="secondary" className="gap-1 text-xs">
              <Star className="h-3 w-3" />Default
            </Badge>
          )}
        </div>
      ),
    },
    {
      accessorKey: "provider",
      header: "Provider",
      cell: ({ row }) => registryProviderLabels[row.original.provider],
    },
    {
      accessorKey: "endpoint",
      header: "Server",
      cell: ({ row }) => (
        <span className="font-mono text-xs max-w-48 truncate block">{row.original.endpoint}</span>
      ),
    },
    {
      accessorKey: "namespace",
      header: "Namespace",
      cell: ({ row }) => row.original.namespace || '-',
    },
    {
      accessorKey: "enabled",
      header: "Status",
      cell: ({ row }) => (
        <Badge variant={row.original.enabled ? 'default' : 'outline'}>
          {row.original.enabled ? 'Enabled' : 'Disabled'}
        </Badge>
      ),
    },
    {
      id: "actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => (
        <div className="flex items-center gap-1 justify-end">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => { setEditRegistry(row.original); setShowDialog(true) }}
          >
            <Pencil />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="text-destructive hover:text-destructive hover:bg-destructive/10"
            onClick={() => deleteMutation.mutate(row.original.id)}
            disabled={deleteMutation.isPending}
          >
            <Trash2 />
          </Button>
        </div>
      ),
    },
  ]

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Database className="h-4 w-4" />
            Image Registries
          </CardTitle>
          <CardDescription>
            {scope === 'cluster'
              ? 'Cluster-level registries available to all projects'
              : 'Project-level registries for this project'}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {!containerRegistries || containerRegistries.length === 0 ? (
            <EmptyState
              title="No image registries configured"
              description="Add a registry to enable container image builds"
              icon={Warehouse}
              actionText="Add Registry"
              onAction={() => { setEditRegistry(null); setShowDialog(true) }}
              actionIcon={Plus}
            />
          ) : (
            <DataTable
              borderless
              columns={columns}
              data={containerRegistries}
              searchKey="name"
              searchPlaceholder="Filter registries..."
              leftActions={() => (
                <Button onClick={() => { setEditRegistry(null); setShowDialog(true) }}>
                  <Plus />
                  Add Registry
                </Button>
              )}
            />
          )}
        </CardContent>
      </Card>

      <ContainerRegistryDialog
        open={showDialog}
        onOpenChange={setShowDialog}
        scope={scope}
        scopeId={scopeId}
        registry={editRegistry}
      />
    </>
  )
}
