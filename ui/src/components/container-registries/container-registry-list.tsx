import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Database, Loader2, Pencil, Plus, Star, Trash2, Warehouse } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import {
  containerRegistriesApi,
  registryProviderLabels,
  type ContainerRegistry,
  type RegistryScope as ContainerRegistryScope,
} from "@/api/container-registries"
import { ContainerRegistryDialog } from "@/components/container-registries/container-registry-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { EmptyState } from "../shared/empty-state"

interface ContainerRegistryListProps {
  scope: ContainerRegistryScope
  scopeId: string
}

export function ContainerRegistryList({ scope, scopeId }: ContainerRegistryListProps) {
  const queryClient = useQueryClient()
  const [showDialog, setShowDialog] = React.useState(false)
  const [editRegistry, setEditRegistry] = React.useState<ContainerRegistry | null>(null)

  const { data: containerRegistries, isLoading } = useQuery({
    queryKey: ['registries', scope, scopeId],
    queryFn: () => scope === 'cluster'
      ? containerRegistriesApi.listByCluster(scopeId)
      : containerRegistriesApi.listByProject(scopeId),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => containerRegistriesApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['registries', scope, scopeId] })
      toast.success('Container registry deleted')
    },
    onError: (err: any) => {
      toast.error(err?.response?.data?.error || 'Failed to delete registry')
    },
  })

  return (
    <>
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-sm flex items-center gap-2">
                <Database className="h-4 w-4" />
                Image Registries
              </CardTitle>
              <CardDescription>
                {scope === 'cluster'
                  ? 'Cluster-level registries available to all projects'
                  : 'Project-level registries for this project'}
              </CardDescription>
            </div>
            <Button size="sm" onClick={() => { setEditRegistry(null); setShowDialog(true) }}>
              <Plus />
              Add Registry
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center p-8">
              <Loader2 className="h-6 w-6 animate-spin" />
            </div>
          ) : !containerRegistries || containerRegistries.length === 0 ? (
            <EmptyState title="No image registries configured" description="Add a registry to enable container image builds" icon={Warehouse} />
          ) : (
            <div className="border-y border-x-0">
              <Table>
                <TableHeader>

                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Provider</TableHead>
                    <TableHead>Server</TableHead>
                    <TableHead>Namespace</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {containerRegistries.map((reg) => (
                    <TableRow key={reg.id}>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          {reg.name}
                          {reg.is_default && (
                            <Badge variant="secondary" className="gap-1 text-xs">
                              <Star className="h-3 w-3" />Default
                            </Badge>
                          )}
                        </div>
                      </TableCell>
                      <TableCell>{registryProviderLabels[reg.provider]}</TableCell>
                      <TableCell className="font-mono text-xs max-w-48 truncate">{reg.endpoint}</TableCell>
                      <TableCell>{reg.namespace || '-'}</TableCell>
                      <TableCell>
                        <Badge variant={reg.enabled ? 'default' : 'outline'}>
                          {reg.enabled ? 'Enabled' : 'Disabled'}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex items-center gap-1 justify-end">
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => { setEditRegistry(reg); setShowDialog(true) }}
                          >
                            <Pencil />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            className="text-destructive hover:text-destructive hover:bg-destructive/10"
                            onClick={() => deleteMutation.mutate(reg.id)}
                            disabled={deleteMutation.isPending}
                          >
                            <Trash2 />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
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
