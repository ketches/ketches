import { collaborationApi, SprintStatusOptions, type Sprint, type UpdateSprintRequest } from "@/api/collaboration"
import { InlineStatusEditor } from "@/components/collaborations/inline-editors"
import { CreateSprintDialog, EditSprintDialog } from "@/components/collaborations/sprint-dialogs"
import { DataTable } from "@/components/data-table/data-table"
import { PageHeader } from "@/components/layout/page-header"
import { EmptyState } from "@/components/shared/empty-state"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { useDebounce } from "@/hooks/use-debounce"
import { formatDate, getErrorMessage } from "@/lib/utils"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import { CalendarRange, Goal, MoreVertical, Pencil, Plus, Trash2 } from "lucide-react"
import { useState } from "react"
import { useParams } from "react-router-dom"
import { toast } from "sonner"

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

  const { data: response, isLoading, refetch } = useQuery({
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
  const hasActiveFilters = Boolean(search.trim())

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
      toast.error("Failed to delete sprint", {
        description: getErrorMessage(error, "Unknown error occurred")
      })
    }

  })

  const updateSprintStatusMutation = useMutation({
    mutationFn: ({ sprint, status }: { sprint: Sprint; status: string }) => {
      if (!projectId) throw new Error("Project ID is required")
      const data: UpdateSprintRequest = {
        name: sprint.name,
        goal: sprint.goal,
        status: status as Sprint["status"],
        start_date: sprint.start_date,
        end_date: sprint.end_date,
      }
      return collaborationApi.updateSprint(projectId, sprint.id, data)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sprints", projectId] })
      toast.success("Sprint status updated")
    },
    onError: (error: unknown) => {
      toast.error("Failed to update status", {
        description: getErrorMessage(error, "Unknown error occurred")
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
      cell: ({ row }) => (
        <InlineStatusEditor
          currentStatus={row.original.status}
          entityType="sprint"
          statusOptions={SprintStatusOptions}
          onStatusChange={(status) => updateSprintStatusMutation.mutate({ sprint: row.original, status })}
        />
      ),
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
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => {
        const item = row.original
        return (
          <div className="flex justify-end">
            <DropdownMenu>
              <DropdownMenuTrigger render={<Button variant="ghost" size="icon-sm" onClick={(e) => e.stopPropagation()} />}>
                <MoreVertical />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => handleEdit(item)}>
                  <Pencil />
                  Edit
                </DropdownMenuItem>
                <DropdownMenuItem variant="destructive" className="text-destructive hover:text-destructive hover:bg-destructive/10" onClick={() => handleDelete(item)}>
                  <Trash2 />
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
    <div className="flex flex-col h-full gap-4">
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

      {<DataTable
        columns={columns}
        data={sprints}
        sourceDataCount={totalCount}
        isLoading={isLoading}
        sourceEmptyContent={
          <EmptyState
            title="No sprints yet"
            description="Create your first sprint to plan iteration work."
            icon={Goal}
            actionText="New Sprint"
            onAction={() => setCreateOpen(true)}
            actionIcon={Plus}
          />
        }
        useStandaloneEmptyState={!hasActiveFilters}
        manualPagination
        totalCount={totalCount}
        pagination={pagination}
        onPaginationChange={setPagination}
        onRefresh={() => refetch()}
        emptyContent={
          <EmptyState
            title=""
            description="No matching results."
            icon={Goal}
            border={false}
          />
        }
        leftToolbar={() => (
          <Input
            className="max-w-xs"
            placeholder="Search..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        )}
        rightToolbar={() => (
          <Button onClick={() => setCreateOpen(true)}>
            <Plus className="h-4 w-4" />
            New Sprint
          </Button>
        )}
      />}

      <CreateSprintDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        projectId={projectId}
      />

      <EditSprintDialog
        key={selectedItem?.id ?? "sprint-edit"}
        open={editOpen}
        onOpenChange={(nextOpen) => {
          setEditOpen(nextOpen)
          if (!nextOpen) {
            setSelectedItem(null)
          }
        }}
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
