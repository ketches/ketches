import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef } from "@tanstack/react-table"
import { Edit2, HardDrive, Plus, Trash2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import type { App } from "@/api/apps"
import { appsApi } from "@/api/apps"
import type { VolumeSpec } from "@/components/applications/volume-dialog"
import { VolumeDialog } from "@/components/applications/volume-dialog"
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

interface VolumesTableProps {
  app: App
}

const VOLUME_TYPE_LABELS: Record<string, string> = {
  pvc: "Persistent (PVC)",
  emptyDir: "Temporary (emptyDir)",
  hostPath: "Local (hostPath)",
}

export function VolumesTable({ app }: VolumesTableProps) {
  const queryClient = useQueryClient()
  const [isDialogOpen, setIsDialogOpen] = React.useState(false)
  const [editingVolume, setEditingVolume] = React.useState<VolumeSpec | null>(null)
  const [searchQuery, setSearchQuery] = React.useState("")
  const [rowSelection, setRowSelection] = React.useState({})
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [deletingVolume, setDeletingVolume] = React.useState<VolumeSpec | null>(null)
  const [bulkDeleteDialogOpen, setBulkDeleteDialogOpen] = React.useState(false)
  const [selectedVolumeIds, setSelectedVolumeIds] = React.useState<string[]>([])

  const { data: volumes = [], isLoading } = useQuery({
    queryKey: ["app-volumes", app.id],
    queryFn: async () => {
      const response = await appsApi.listVolumes(app.id)
      // Transform backend response to match VolumeSpec
      return response.map((vol: any) => ({
        id: vol.ID || vol.id,
        slug: vol.Slug || vol.slug,
        volume_type: vol.VolumeType || vol.volume_type || "pvc",
        mount_path: vol.MountPath || vol.mount_path,
        sub_path: vol.SubPath || vol.sub_path,
        storage_class: vol.StorageClass || vol.storage_class,
        capacity: vol.Capacity || vol.capacity || 1,
        access_modes: vol.AccessModes || vol.access_modes || "ReadWriteOnce",
        volume_mode: vol.VolumeMode || vol.volume_mode || "Filesystem",
      }))
    },
  })

  // Filter volumes based on search query
  const filteredVolumes = React.useMemo(() => {
    if (!searchQuery) return volumes
    const lowQuery = searchQuery.toLowerCase()
    return volumes.filter(
      (vol: VolumeSpec) =>
        vol.slug?.toLowerCase().includes(lowQuery) ||
        vol.mount_path?.toLowerCase().includes(lowQuery)
    )
  }, [volumes, searchQuery])

  const deleteMutation = useMutation({
    mutationFn: (volumeId: string) => {
      return appsApi.deleteVolume(volumeId)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["app-volumes", app.id] })
      toast.success("Volume deleted successfully")
    },
    onError: (err: unknown) => {
      const error = err as { response?: { data?: { error?: string } } }
      toast.error("Failed to delete volume", {
        description: error.response?.data?.error || "Unknown error",
      })
    },
  })

  const bulkDeleteMutation = useMutation({
    mutationFn: async (volumeIds: string[]) => {
      return Promise.all(volumeIds.map(id => appsApi.deleteVolume(id)))
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["app-volumes", app.id] })
      toast.success("Volumes deleted successfully")
      setRowSelection({})
    },
    onError: (err: unknown) => {
      const error = err as { response?: { data?: { error?: string } } }
      toast.error("Failed to delete volumes", {
        description: error.response?.data?.error || "Unknown error",
      })
    },
  })

  const handleEdit = (volume: VolumeSpec) => {
    setEditingVolume(volume)
    setIsDialogOpen(true)
  }

  const handleAdd = () => {
    setEditingVolume(null)
    setIsDialogOpen(true)
  }

  const handleDelete = (volume: VolumeSpec) => {
    if (!volume.id) return
    setDeletingVolume(volume)
    setDeleteDialogOpen(true)
  }

  const handleDialogSuccess = () => {
    setIsDialogOpen(false)
    setEditingVolume(null)
  }

  const volumeColumns: ColumnDef<VolumeSpec>[] = [
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
      accessorKey: "volume_type",
      header: "Type",
      cell: ({ row }) => (
        <Badge variant="outline">
          {VOLUME_TYPE_LABELS[row.original.volume_type] || row.original.volume_type}
        </Badge>
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
      accessorKey: "sub_path",
      header: "SubPath",
      cell: ({ row }) => (
        <span className="font-mono text-xs text-muted-foreground">
          {row.original.sub_path || <span className="text-muted-foreground">-</span>}
        </span>
      ),
    },
    {
      accessorKey: "capacity",
      header: "Capacity",
      cell: ({ row }) => (
        row.original.volume_type === "pvc" ? (
          <span className="text-sm">{row.original.capacity} GiB</span>
        ) : (
          <span className="text-muted-foreground text-xs">-</span>
        )
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
            <HardDrive className="h-4 w-4" /> Storage Volumes
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8 text-muted-foreground text-sm">
            Loading storage volumes...
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm flex items-center gap-2">
          <HardDrive className="h-4 w-4" /> Storage Volumes
        </CardTitle>
        <CardDescription>
          Configure persistent or ephemeral storage for your application
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {volumes.length === 0 ? (
          <EmptyState
            title="No volumes configured"
            description="Add storage volumes for your application."
            icon={HardDrive}
            actionText="Add Volume"
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
                      const selectedIds = selectedIndices.map(idx => filteredVolumes[parseInt(idx)]?.id).filter(Boolean) as string[]

                      setSelectedVolumeIds(selectedIds)
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
                  Add Volume
                </Button>
              </div>
            </div>
            <DataTable
              borderless
              columns={volumeColumns}
              data={filteredVolumes}
              rowSelection={rowSelection}
              onRowSelectionChange={setRowSelection}
              hidePagination
            />
          </>
        )}
      </CardContent>

      <VolumeDialog
        app={app}
        volume={editingVolume}
        open={isDialogOpen}
        onOpenChange={setIsDialogOpen}
        onSuccess={handleDialogSuccess}
      />
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Volume</AlertDialogTitle>
            <AlertDialogDescription>
              {deletingVolume
                ? `Are you sure you want to delete volume "${deletingVolume.slug}"? This may result in data loss and cannot be undone.`
                : "Are you sure you want to delete this volume?"}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (deletingVolume) {
                  deleteMutation.mutate(deletingVolume.id!)
                }
                setDeleteDialogOpen(false)
                setDeletingVolume(null)
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
            <AlertDialogTitle>Delete Volumes</AlertDialogTitle>
            <AlertDialogDescription>
              {selectedVolumeIds.length > 0
                ? `Are you sure you want to delete ${selectedVolumeIds.length} volume(s)? This may result in data loss and cannot be undone.`
                : "Are you sure you want to delete these volumes?"}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (selectedVolumeIds.length > 0) {
                  bulkDeleteMutation.mutate(selectedVolumeIds)
                  setRowSelection({})
                }
                setBulkDeleteDialogOpen(false)
                setSelectedVolumeIds([])
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
