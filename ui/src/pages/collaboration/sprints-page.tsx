import { collaborationApi, type Sprint } from "@/api/collaboration"
import { DataTable } from "@/components/data-table/data-table"
import { PageHeader } from "@/components/layout/page-header"
import { StatusBadge } from "@/components/collaboration/collab-badges"
import { CreateSprintDialog, EditSprintDialog } from "@/components/collaboration/sprint-dialogs"
import { EmptyState } from "@/components/shared/empty-state"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { useDebounce } from "@/hooks/use-debounce"
import { formatDate } from "@/lib/utils"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import { CalendarRange, MoreHorizontal, Pencil, Plus, Trash2 } from "lucide-react"
import { useState } from "react"
import { useParams } from "react-router-dom"
import { toast } from "sonner"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"

interface SprintsPageProps {
  projectId?: string
}

export default function SprintsPage({ projectId: propProjectId }: SprintsPageProps) {
  const params = useParams<{ projectId: string }>()
  const projectId = propProjectId || params.projectId
  const queryClient = useQueryClient()

  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  })
  const [search, setSearch] = useState("")
  const debouncedSearch = useDebounce(search, 300)

  // Dialog states
  const [createOpen, setCreateOpen] = useState(false)
  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [selectedItem, setSelectedItem] = useState<Sprint | null>(null)

  const { data: response, isLoading } = useQuery({
    queryKey: ["sprints", projectId, pagination.pageIndex, pagination.pageSize, debouncedSearch],
    queryFn: () => {
      if (!projectId) throw new Error("Project ID is required")
      return collaborationApi.listSprints(projectId, {
        page: pagination.pageIndex + 1,
        page_size: pagination.pageSize,
        search: debouncedSearch,
      })
    },
    enabled: !!projectId,
  })

  const sprints = response?.items || []
  const totalCount = response?.pagination?.total || 0

  const deleteMutation = useMutation({
    mutationFn: (id: string) => {
      if (!projectId) throw new Error("Project ID is required")
      return collaborationApi.deleteSprint(projectId, id)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sprints", projectId] })
      toast.success("Sprint deleted")
      setDeleteOpen(false)
      setSelectedItem(null)
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: string } } }
      toast.error("Failed to delete sprint", {
        description: err.response?.data?.error || "Unknown error occurred"
      })
    }

  })

  const handleEdit = (item: Sprint) => {
    setSelectedItem(item)
    setEditOpen(true)
  }

  const handleDelete = (item: Sprint) => {
    setSelectedItem(item)
    setDeleteOpen(true)
  }

  const columns: ColumnDef<Sprint>[] = [
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => <span className="font-medium">{row.original.name}</span>,
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <StatusBadge status={row.original.status} />,
    },
    {
      accessorKey: "start_date",
      header: "Start Date",
      cell: ({ row }) => <span className="text-xs text-muted-foreground">{formatDate(row.original.start_date)}</span>,
    },
    {
      accessorKey: "end_date",
      header: "End Date",
      cell: ({ row }) => <span className="text-xs text-muted-foreground">{formatDate(row.original.end_date)}</span>,
    },
    {
      id: "actions",
      cell: ({ row }) => {
        const item = row.original
        return (
          <div className="flex justify-end">
            <DropdownMenu>
              <DropdownMenuTrigger className="flex h-8 w-8 items-center justify-center rounded-md hover:bg-muted transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2">
                <MoreHorizontal className="h-4 w-4" />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => handleEdit(item)}>
                  <Pencil className="mr-2 h-3.5 w-3.5" />
                  Edit
                </DropdownMenuItem>
                <DropdownMenuItem className="text-destructive focus:text-destructive" onClick={() => handleDelete(item)}>
                  <Trash2 className="mr-2 h-3.5 w-3.5" />
                  Delete
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        )
      },
    },
  ]

  if (!projectId) return null

  return (
    <div className="flex flex-col h-full gap-6">
      {!propProjectId && <PageHeader items={[{ label: "Sprints", icon: CalendarRange }]} />}

      {!propProjectId && (
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">Sprints</h1>
            <p className="text-sm text-muted-foreground">
              Manage development sprints and iterations.
            </p>
          </div>
        </div>
      )}

      {!isLoading && sprints.length === 0 ? (
        <EmptyState
          title="No sprints found"
          description="Create your first sprint to get started."
          icon={CalendarRange}
          actionText="Create Sprint"
          onAction={() => setCreateOpen(true)}
          actionIcon={Plus}
        />
      ) : (
        <>
          <div className="flex flex-wrap items-center justify-between gap-4">
            <Input
              className="max-w-xs"
              placeholder="Search..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
            <Button onClick={() => setCreateOpen(true)}>
              <Plus className="mr-2 h-4 w-4" />
              New Sprint
            </Button>
          </div>
          <DataTable
            columns={columns}
            data={sprints}
            isLoading={isLoading}
            manualPagination
            totalCount={totalCount}
            pagination={pagination}
            onPaginationChange={setPagination}
          />
        </>
      )}

      <CreateSprintDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        projectId={projectId}
      />

      <EditSprintDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        projectId={projectId}
        sprint={selectedItem}
      />

      <AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Sprint?</AlertDialogTitle>
            <AlertDialogDescription>
              This will permanently delete "{selectedItem?.name}".
              Related requirements will be unassigned from this sprint.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              onClick={() => selectedItem && deleteMutation.mutate(selectedItem.id)}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
