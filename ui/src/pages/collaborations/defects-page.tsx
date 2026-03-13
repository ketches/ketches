import { collaborationApi, type Defect, type DefectStatus, DefectStatusOptions } from "@/api/collaboration"
import { SeverityBadge } from "@/components/collaboration/collab-badges"
import { CreateDefectDialog, DeleteDefectDialog, EditDefectDialog, TransitionDefectDialog } from "@/components/collaboration/defect-dialogs"
import { InlineStatusEditor } from "@/components/collaboration/inline-editors"
import { DataTable } from "@/components/data-table/data-table"
import { PageHeader } from "@/components/layout/page-header"
import { EmptyState } from "@/components/shared/empty-state"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { useDebounce } from "@/hooks/use-debounce"
import { formatDate } from "@/lib/utils"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import { ArrowRightLeft, Bug, MoreVertical, Pencil, Plus, Trash2 } from "lucide-react"
import { useMemo, useState } from "react"
import { useParams } from "react-router-dom"
import { toast } from "sonner"

interface DefectsPageProps {
  projectId?: string
  assigneeId?: string
  sprintId?: string
}

export default function DefectsPage({ projectId: propProjectId, assigneeId, sprintId }: DefectsPageProps) {
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
  const [transitionOpen, setTransitionOpen] = useState(false)
  const [selectedItem, setSelectedItem] = useState<Defect | null>(null)

  const { data: response, isLoading } = useQuery({
    queryKey: ["defects", projectId, pagination.pageIndex, pagination.pageSize, debouncedSearch, assigneeId, sprintId],
    queryFn: () => {
      if (!projectId) throw new Error("Project ID is required")
      return collaborationApi.listDefects(projectId, {
        page: pagination.pageIndex + 1,
        page_size: pagination.pageSize,
        search: debouncedSearch,
        assignee_id: assigneeId,
        sprint_id: sprintId,
      })
    },
    enabled: !!projectId,
  })

  const defects = response?.items || []
  const totalCount = response?.pagination?.total || 0

  const transitionDefectMutation = useMutation({
    mutationFn: ({ defectId, status }: { defectId: string; status: string }) => {
      if (!projectId) throw new Error("Project ID is required")
      return collaborationApi.transitionDefect(projectId, defectId, status as DefectStatus)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["defects", projectId] })
      toast.success("Defect status updated")
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: string } } }
      toast.error("Failed to update status", {
        description: err.response?.data?.error || "Unknown error occurred"
      })
    }
  })

  const handleEdit = (item: Defect) => {
    setSelectedItem(item)
    setEditOpen(true)
  }

  const handleDelete = (item: Defect) => {
    setSelectedItem(item)
    setDeleteOpen(true)
  }

  const handleTransition = (item: Defect) => {
    setSelectedItem(item)
    setTransitionOpen(true)
  }

  const columns: ColumnDef<Defect>[] = useMemo(() => [
    {
      accessorKey: "title",
      header: "Title",
      cell: ({ row }) => (
        <div className="flex flex-col">
          <span className="font-medium truncate max-w-100">{row.original.title}</span>
          <span className="text-xs text-muted-foreground font-mono">{row.original.id.slice(0, 8)}</span>
        </div>
      ),
    },
    {
      accessorKey: "severity",
      header: "Severity",
      cell: ({ row }) => <SeverityBadge severity={row.original.severity} />,
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => (
        <InlineStatusEditor
          currentStatus={row.original.status}

          statusOptions={DefectStatusOptions}
          onStatusChange={(status) => transitionDefectMutation.mutate({ defectId: row.original.id, status })}
        />
      ),
    },
    {
      accessorKey: "created_at",
      header: "Created",
      cell: ({ row }) => <span className="text-xs text-muted-foreground">{formatDate(row.original.created_at)}</span>,
    },
    {
      id: "actions",
      cell: ({ row }) => {
        const item = row.original
        return (
          <div className="flex justify-end">
            <DropdownMenu>
              <DropdownMenuTrigger render={<Button variant="ghost" size="icon-sm" onClick={(e) => e.stopPropagation()} />}>
                <MoreVertical />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => handleTransition(item)}>
                  <ArrowRightLeft />
                  Transition
                </DropdownMenuItem>
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
  ], [])

  if (!projectId) return null

  return (
    <div className="flex flex-col h-full gap-6">
      {!propProjectId && <PageHeader items={[{ label: "Defects", icon: Bug }]} />}

      {!propProjectId && (
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">Defects</h1>
            <p className="text-sm text-muted-foreground">
              Track and manage project defects and bugs.
            </p>
          </div>
        </div>
      )}

      {!isLoading && defects.length === 0 ? (
        <EmptyState
          title="No defects found"
          description="Good job! No bugs reported yet."
          icon={Bug}
          actionText="Report Defect"
          onAction={() => setCreateOpen(true)}
          actionIcon={Plus}
        />
      ) : (
        <>
          <div className="flex flex-wrap items-center justify-between gap-4">
            <Input
              className="max-w-xs"
              placeholder="Search defects..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
            <Button onClick={() => setCreateOpen(true)}>
              <Plus className="mr-2 h-4 w-4" />
              Report Defect
            </Button>
          </div>
          <DataTable
            columns={columns}
            data={defects}
            isLoading={isLoading}
            manualPagination
            totalCount={totalCount}
            pagination={pagination}
            onPaginationChange={setPagination}
          />
        </>
      )}

      <CreateDefectDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        projectId={projectId}
      />

      <EditDefectDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        projectId={projectId}
        defect={selectedItem}
      />

      {selectedItem && (
        <>
          <DeleteDefectDialog
            open={deleteOpen}
            onOpenChange={setDeleteOpen}
            projectId={projectId}
            defectId={selectedItem.id}
            onSuccess={() => {
              setDeleteOpen(false)
              setSelectedItem(null)
            }}
          />
          <TransitionDefectDialog
            open={transitionOpen}
            onOpenChange={setTransitionOpen}
            projectId={projectId}
            defect={selectedItem}
            onSuccess={() => {
              setTransitionOpen(false)
              // Don't nullify selectedItem here to avoid glitch
            }}
          />
        </>
      )}
    </div>
  )
}
