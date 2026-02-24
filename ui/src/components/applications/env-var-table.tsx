import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef } from "@tanstack/react-table"
import { Edit2, Key, Plus, Trash2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import type { App } from "@/api/apps"
import { appsApi } from "@/api/apps"
import type { EnvVarSpec } from "@/components/applications/env-var-dialog"
import { EnvVarDialog } from "@/components/applications/env-var-dialog"
import { DataTable } from "@/components/data-table/data-table"
import { EmptyState } from "@/components/shared/empty-state"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"

interface EnvVarTableProps {
  app: App
}

export function EnvVarTable({ app }: EnvVarTableProps) {
  const queryClient = useQueryClient()
  const [isDialogOpen, setIsDialogOpen] = React.useState(false)
  const [editingEnvVar, setEditingEnvVar] = React.useState<EnvVarSpec | null>(null)
  const [searchQuery, setSearchQuery] = React.useState("")
  const [rowSelection, setRowSelection] = React.useState({})
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [deletingEnvVar, setDeletingEnvVar] = React.useState<EnvVarSpec | null>(null)
  const [bulkDeleteDialogOpen, setBulkDeleteDialogOpen] = React.useState(false)
  const [selectedEnvVarIds, setSelectedEnvVarIds] = React.useState<string[]>([])

  const { data: envVars = [], isLoading } = useQuery({
    queryKey: ["app-env-vars", app.id],
    queryFn: async () => {
      const response = await appsApi.listEnvVars(app.id)
      // Transform backend response to match EnvVarSpec
      return response.map((ev: any) => ({
        id: ev.ID || ev.id,
        key: ev.Key || ev.key,
        value: ev.Value || ev.value,
      }))
    },
  })

  // Filter env vars based on search query
  const filteredEnvVars = React.useMemo(() => {
    if (!searchQuery) return envVars
    const lowQuery = searchQuery.toLowerCase()
    return envVars.filter(
      (ev: EnvVarSpec) =>
        ev.key?.toLowerCase().includes(lowQuery) ||
        ev.value?.toLowerCase().includes(lowQuery)
    )
  }, [envVars, searchQuery])

  const deleteMutation = useMutation({
    mutationFn: (envVarId: string) => {
      return appsApi.deleteEnvVar(envVarId)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["app-env-vars", app.id] })
      toast.success("Environment variable deleted")
    },
    onError: (err: unknown) => {
      const error = err as { response?: { data?: { error?: string } } }
      toast.error("Failed to delete environment variable", {
        description: error.response?.data?.error || "Unknown error",
      })
    },
  })

  const bulkDeleteMutation = useMutation({
    mutationFn: async (envVarIds: string[]) => {
      return Promise.all(envVarIds.map((id) => appsApi.deleteEnvVar(id)))
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["app-env-vars", app.id] })
      toast.success("Environment variables deleted")
      setRowSelection({})
    },
    onError: (err: unknown) => {
      const error = err as { response?: { data?: { error?: string } } }
      toast.error("Failed to delete environment variables", {
        description: error.response?.data?.error || "Unknown error",
      })
    },
  })

  const handleEdit = (envVar: EnvVarSpec) => {
    setEditingEnvVar(envVar)
    setIsDialogOpen(true)
  }

  const handleAdd = () => {
    setEditingEnvVar(null)
    setIsDialogOpen(true)
  }

  const handleDelete = (envVar: EnvVarSpec) => {
    if (!envVar.id) return
    setDeletingEnvVar(envVar)
    setDeleteDialogOpen(true)
  }

  const handleDialogSuccess = () => {
    setIsDialogOpen(false)
    setEditingEnvVar(null)
  }

  const envVarColumns: ColumnDef<EnvVarSpec>[] = [
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
      accessorKey: "key",
      header: "Key",
      cell: ({ row }) => (
        <span className="font-mono text-sm font-medium">{row.original.key}</span>
      ),
    },
    {
      accessorKey: "value",
      header: "Value",
      cell: ({ row }) => {
        const value = row.original.value
        const displayValue = value && value.length > 50 ? value.substring(0, 50) + "..." : value
        return (
          <span className="font-mono text-sm text-muted-foreground" title={value}>
            {displayValue || <span className="text-muted-foreground">-</span>}
          </span>
        )
      },
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
        <CardHeader className="pb-2">
          <CardTitle className="text-sm flex items-center gap-2">
            <Key className="h-4 w-4" /> Environment Variables
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8 text-muted-foreground text-sm">
            Loading environment variables...
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm flex items-center gap-2">
          <Key className="h-4 w-4" /> Environment Variables
        </CardTitle>
        <CardDescription>
          Configure key-value pairs for your application environment
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {envVars.length === 0 ? (
          <EmptyState
            title="No environment variables configured"
            description="Add environment variables for your application."
            icon={Key}
            actionText="Add Variable"
            onAction={handleAdd}
            actionIcon={Plus}
          />
        ) : (
          <>
            <div className="flex flex-wrap items-center justify-between gap-4">
              <Input
                className="flex flex-1 max-w-sm min-w-75"
                placeholder="Filter by key, value..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
              />

              <div className="flex items-center gap-2">
                {Object.keys(rowSelection).length > 0 && (
                  <Button
                    variant="destructive"
                    onClick={() => {
                      const selectedIndices = Object.keys(rowSelection).filter(
                        (key) => rowSelection[key as keyof typeof rowSelection]
                      )
                      const selectedIds = selectedIndices
                        .map((idx) => filteredEnvVars[parseInt(idx)]?.id)
                        .filter(Boolean) as string[]

                      setSelectedEnvVarIds(selectedIds)
                      setBulkDeleteDialogOpen(true)
                    }}
                    disabled={bulkDeleteMutation.isPending}
                  >
                    <Trash2 />
                    Delete (
                    {
                      Object.keys(rowSelection).filter(
                        (key) => rowSelection[key as keyof typeof rowSelection]
                      ).length
                    }
                    )
                  </Button>
                )}
                <Button onClick={handleAdd}>
                  <Plus />
                  Add Variable
                </Button>
              </div>
            </div>
            <DataTable
              borderless
              columns={envVarColumns}
              data={filteredEnvVars}
              rowSelection={rowSelection}
              onRowSelectionChange={setRowSelection}
              hidePagination
            />
          </>
        )}
      </CardContent>

      <EnvVarDialog
        app={app}
        envVar={editingEnvVar}
        open={isDialogOpen}
        onOpenChange={setIsDialogOpen}
        onSuccess={handleDialogSuccess}
      />
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Environment Variable</AlertDialogTitle>
            <AlertDialogDescription>
              {deletingEnvVar
                ? `Are you sure you want to delete environment variable "${deletingEnvVar.key}"? This action cannot be undone.`
                : "Are you sure you want to delete this environment variable?"}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (deletingEnvVar) {
                  deleteMutation.mutate(deletingEnvVar.id!)
                }
                setDeleteDialogOpen(false)
                setDeletingEnvVar(null)
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
            <AlertDialogTitle>Delete Environment Variables</AlertDialogTitle>
            <AlertDialogDescription>
              {selectedEnvVarIds.length > 0
                ? `Are you sure you want to delete ${selectedEnvVarIds.length} environment variable(s)? This action cannot be undone.`
                : "Are you sure you want to delete these environment variables?"}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (selectedEnvVarIds.length > 0) {
                  bulkDeleteMutation.mutate(selectedEnvVarIds)
                  setRowSelection({})
                }
                setBulkDeleteDialogOpen(false)
                setSelectedEnvVarIds([])
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
