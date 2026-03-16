import { collaborationApi, type Defect, type DefectSeverity, type DefectStatus, DefectStatusOptions, type UpdateDefectRequest } from "@/api/collaboration"
import { AssigneeFilter, PriorityFilter, StatusFilter } from "@/components/collaborations/collab-filters"
import { CreateDefectDialog, DeleteDefectDialog, EditDefectDialog } from "@/components/collaborations/defect-dialogs"
import { InlineAssigneeEditor, InlineSeverityEditor, InlineStatusEditor } from "@/components/collaborations/inline-editors"
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
import { Bug, MoreVertical, Pencil, Plus, Trash2 } from "lucide-react"
import { useState } from "react"
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


  const [statusFilter, setStatusFilter] = useState("")
  const [priorityFilter, setPriorityFilter] = useState("")
  const [assigneeFilter, setAssigneeFilter] = useState("")
  // Dialog states
  const [createOpen, setCreateOpen] = useState(false)
  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [selectedItem, setSelectedItem] = useState<Defect | null>(null)

  const { data: response, isLoading, refetch } = useQuery({
    queryKey: ["defects", projectId, pagination.pageIndex, pagination.pageSize, debouncedSearch, assigneeId, sprintId, statusFilter, priorityFilter, assigneeFilter],
    queryFn: () => {
      if (!projectId) throw new Error("Project ID is required")
      return collaborationApi.listDefects(projectId, {
        page: pagination.pageIndex + 1,
        page_size: pagination.pageSize,
        search: debouncedSearch,
        assignee_id: assigneeId || assigneeFilter || undefined,
        sprint_id: sprintId,
        status: statusFilter || undefined,
        priority: priorityFilter || undefined,
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

  const updateDefectSeverityMutation = useMutation({
    mutationFn: ({ defect, severity }: { defect: Defect; severity: string }) => {
      if (!projectId) throw new Error("Project ID is required")
      const data: UpdateDefectRequest = {
        title: defect.title,
        description: defect.description,
        severity: severity as DefectSeverity,
        status: defect.status,
        assignee_id: defect.assignee_id,
        sprint_id: defect.sprint_id,
        requirement_id: defect.requirement_id,
        task_id: defect.task_id,
        test_case_id: defect.test_case_id,
        test_run_id: defect.test_run_id,
        reproduction_steps: defect.reproduction_steps,
        fix_note: defect.fix_note,
        runtime_context_json: defect.runtime_context_json,
      }
      return collaborationApi.updateDefect(projectId, defect.id, data)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["defects", projectId] })
      toast.success("Defect severity updated")
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: string } } }
      toast.error("Failed to update severity", {
        description: err.response?.data?.error || "Unknown error occurred"
      })
    }
  })

  const updateDefectAssigneeMutation = useMutation({
    mutationFn: ({ defect, assigneeId }: { defect: Defect; assigneeId: string }) => {
      if (!projectId) throw new Error("Project ID is required")
      const data: UpdateDefectRequest = {
        title: defect.title,
        description: defect.description,
        severity: defect.severity,
        status: defect.status,
        assignee_id: assigneeId || undefined,
        sprint_id: defect.sprint_id,
        requirement_id: defect.requirement_id,
        task_id: defect.task_id,
        test_case_id: defect.test_case_id,
        test_run_id: defect.test_run_id,
        reproduction_steps: defect.reproduction_steps,
        fix_note: defect.fix_note,
        runtime_context_json: defect.runtime_context_json,
      }
      return collaborationApi.updateDefect(projectId, defect.id, data)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["defects", projectId] })
      toast.success("Defect assignee updated")
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: string } } }
      toast.error("Failed to update assignee", {
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

  const columns: ColumnDef<Defect>[] = [
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
      cell: ({ row }) => (
        <InlineSeverityEditor
          currentSeverity={row.original.severity}
          onSeverityChange={(severity) => updateDefectSeverityMutation.mutate({ defect: row.original, severity })}
        />
      ),
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => (
        <InlineStatusEditor
          currentStatus={row.original.status}
          entityType="defect"
          statusOptions={DefectStatusOptions}
          onStatusChange={(status) => transitionDefectMutation.mutate({ defectId: row.original.id, status })}
        />
      ),
    },
    {
      accessorKey: "assignee_id",
      header: "Assignee",
      cell: ({ row }) => (
        <InlineAssigneeEditor
          projectId={projectId!}
          currentAssigneeId={row.original.assignee_id}
          onAssigneeChange={(assigneeId) => updateDefectAssigneeMutation.mutate({ defect: row.original, assigneeId })}
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

      {<DataTable
        columns={columns}
        data={defects}
        isLoading={isLoading}
        manualPagination
        totalCount={totalCount}
        pagination={pagination}
        onPaginationChange={setPagination}
        onRefresh={() => refetch()}
        emptyContent={
          <EmptyState
            title=""
            description="No results."
            icon={Bug}
            border={false}
          />
        }
        leftToolbar={() => (
          <>
            <Input
              className="max-w-xs"
              placeholder="Search defects..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
            <StatusFilter value={statusFilter} onChange={setStatusFilter} options={DefectStatusOptions} />
            <PriorityFilter value={priorityFilter} onChange={setPriorityFilter} />
            <AssigneeFilter projectId={projectId!} value={assigneeFilter} onChange={setAssigneeFilter} />
          </>
        )}
        rightToolbar={() => (
          <Button onClick={() => setCreateOpen(true)}>
            <Plus className="h-4 w-4" />
            Report Defect
          </Button>
        )}
      />}

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
      )}
    </div>
  )
}
