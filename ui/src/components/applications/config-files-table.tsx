import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef } from "@tanstack/react-table"
import { Edit2, FileCog, Plus, Trash2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import type { App } from "@/api/apps"
import { appsApi } from "@/api/apps"
import type { ConfigFileSpec } from "@/components/applications/config-file-dialog"
import { ConfigFileDialog } from "@/components/applications/config-file-dialog"
import { DataTable } from "@/components/data-table/data-table"
import { EmptyState } from "@/components/shared/empty-state"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Badge } from "@/components/ui/badge"
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

interface ConfigFilesTableProps {
  app: App
}

export function ConfigFilesTable({ app }: ConfigFilesTableProps) {
  const queryClient = useQueryClient()
  const [isDialogOpen, setIsDialogOpen] = React.useState(false)
  const [editingConfigFile, setEditingConfigFile] = React.useState<ConfigFileSpec | null>(
    null
  )
  const [searchQuery, setSearchQuery] = React.useState("")
  const [rowSelection, setRowSelection] = React.useState({})
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [deletingConfigFile, setDeletingConfigFile] = React.useState<ConfigFileSpec | null>(null)
  const [bulkDeleteDialogOpen, setBulkDeleteDialogOpen] = React.useState(false)
  const [selectedConfigFileIds, setSelectedConfigFileIds] = React.useState<string[]>([])

  const { data: configFiles = [], isLoading } = useQuery({
    queryKey: ["app-config-files", app.id],
    queryFn: async () => {
      const response = await appsApi.listConfigFiles(app.id)
      // Transform backend response to match ConfigFileSpec
      return response.map((cf: any) => ({
        id: cf.ID || cf.id,
        slug: cf.Slug || cf.slug,
        mount_path: cf.MountPath || cf.mount_path,
        file_mode: cf.FileMode || cf.file_mode || "0644",
        content: cf.Content || cf.content,
      }))
    },
  })

  // Filter config files based on search query
  const filteredConfigFiles = React.useMemo(() => {
    if (!searchQuery) return configFiles
    const lowQuery = searchQuery.toLowerCase()
    return configFiles.filter(
      (cf: ConfigFileSpec) =>
        cf.slug?.toLowerCase().includes(lowQuery) ||
        cf.mount_path?.toLowerCase().includes(lowQuery)
    )
  }, [configFiles, searchQuery])

  const deleteMutation = useMutation({
    mutationFn: (configFileId: string) => {
      return appsApi.deleteConfigFile(configFileId)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["app-config-files", app.id] })
      toast.success("Config file deleted successfully")
    },
    onError: (err: unknown) => {
      const error = err as { response?: { data?: { error?: string } } }
      toast.error("Failed to delete config file", {
        description: error.response?.data?.error || "Unknown error",
      })
    },
  })

  const bulkDeleteMutation = useMutation({
    mutationFn: async (configFileIds: string[]) => {
      return Promise.all(configFileIds.map(id => appsApi.deleteConfigFile(id)))
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["app-config-files", app.id] })
      toast.success("Config files deleted successfully")
      setRowSelection({})
    },
    onError: (err: unknown) => {
      const error = err as { response?: { data?: { error?: string } } }
      toast.error("Failed to delete config files", {
        description: error.response?.data?.error || "Unknown error",
      })
    },
  })

  const handleEdit = (configFile: ConfigFileSpec) => {
    setEditingConfigFile(configFile)
    setIsDialogOpen(true)
  }

  const handleAdd = () => {
    setEditingConfigFile(null)
    setIsDialogOpen(true)
  }

  const handleDelete = (configFile: ConfigFileSpec) => {
    if (!configFile.id) return
    setDeletingConfigFile(configFile)
    setDeleteDialogOpen(true)
  }

  const handleDialogSuccess = () => {
    setIsDialogOpen(false)
    setEditingConfigFile(null)
  }

  const configFileColumns: ColumnDef<ConfigFileSpec>[] = [
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
      accessorKey: "slug",
      header: "Slug",
      cell: ({ row }) => (
        <span className="font-mono text-sm font-medium">{row.original.slug}</span>
      ),
    },
    {
      accessorKey: "mount_path",
      header: "Mount Path",
      cell: ({ row }) => (
        <span className="font-mono text-sm">{row.original.mount_path}</span>
      ),
    },
    {
      accessorKey: "file_mode",
      header: "Permissions",
      cell: ({ row }) => (
        <Badge variant="outline" className="font-mono text-xs">
          {row.original.file_mode}
        </Badge>
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
        <CardHeader className="pb-2">
          <CardTitle className="text-sm flex items-center gap-2">
            <FileCog className="h-4 w-4" /> Config Files
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8 text-muted-foreground text-sm">
            Loading configuration files...
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm flex items-center gap-2">
          <FileCog className="h-4 w-4" /> Config Files
        </CardTitle>
        <CardDescription>
          Mount configuration files into your application instances
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {configFiles.length === 0 ? (
          <EmptyState
            title="No config files configured"
            description="Add configuration files to mount into your application."
            icon={FileCog}
            actionText="Add Config File"
            onAction={handleAdd}
            actionIcon={Plus}
          />
        ) : (
          <>
            <div className="flex flex-wrap items-center justify-between gap-4">
              <Input
                className="flex flex-1 max-w-sm min-w-75"
                placeholder="Filter by slug, mount path..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
              />

              <div className="flex items-center gap-2">
                {Object.keys(rowSelection).length > 0 && (
                  <Button
                    variant="destructive"
                    onClick={() => {
                      const selectedIndices = Object.keys(rowSelection).filter(key => rowSelection[key as keyof typeof rowSelection])
                      const selectedIds = selectedIndices.map(idx => filteredConfigFiles[parseInt(idx)]?.id).filter(Boolean) as string[]

                      setSelectedConfigFileIds(selectedIds)
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
                  Add Config File
                </Button>
              </div>
            </div>
            <DataTable
              borderless
              columns={configFileColumns}
              data={filteredConfigFiles}
              rowSelection={rowSelection}
              onRowSelectionChange={setRowSelection}
              hidePagination
            />
          </>
        )}
      </CardContent>

      <ConfigFileDialog
        app={app}
        configFile={editingConfigFile}
        open={isDialogOpen}
        onOpenChange={setIsDialogOpen}
        onSuccess={handleDialogSuccess}
      />
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Config File</AlertDialogTitle>
            <AlertDialogDescription>
              {deletingConfigFile
                ? `Are you sure you want to delete config file "${deletingConfigFile.slug}"? This action cannot be undone.`
                : "Are you sure you want to delete this config file?"}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (deletingConfigFile) {
                  deleteMutation.mutate(deletingConfigFile.id!)
                }
                setDeleteDialogOpen(false)
                setDeletingConfigFile(null)
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
            <AlertDialogTitle>Delete Config Files</AlertDialogTitle>
            <AlertDialogDescription>
              {selectedConfigFileIds.length > 0
                ? `Are you sure you want to delete ${selectedConfigFileIds.length} config file(s)? This action cannot be undone.`
                : "Are you sure you want to delete these config files?"}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (selectedConfigFileIds.length > 0) {
                  bulkDeleteMutation.mutate(selectedConfigFileIds)
                  setRowSelection({})
                }
                setBulkDeleteDialogOpen(false)
                setSelectedConfigFileIds([])
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
