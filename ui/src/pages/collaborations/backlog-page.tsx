import { collaborationApi, CollabPriority, RequirementStatus, RequirementStatusOptions, type Requirement } from "@/api/collaboration"
import { PlanToSprintDialog } from "@/components/collaborations/backlog-dialogs"
import { AssigneeFilter, PriorityFilter, StatusFilter } from "@/components/collaborations/collab-filters"
import { InlineAssigneeEditor, InlinePriorityEditor, InlineStatusEditor } from "@/components/collaborations/inline-editors"
import { DataTable } from "@/components/data-table/data-table"
import { PageHeader } from "@/components/layout/page-header"
import { EmptyState } from "@/components/shared/empty-state"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { useDebounce } from "@/hooks/use-debounce"
import { buildRequirementUpdateRequest } from "@/lib/collaboration-update-payloads"
import { formatDate } from "@/lib/utils"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef, type PaginationState, type RowSelectionState } from "@tanstack/react-table"
import { Archive, ArrowRight, CornerDownRight, Logs, MoreVertical, Pencil, Plus, Trash2 } from "lucide-react"
import { useState } from "react"
import { useParams } from "react-router-dom"
import { toast } from "sonner"

import { CreateRequirementDialog, DeleteRequirementDialog, EditRequirementDialog } from "@/components/collaborations/requirement-dialogs"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
interface BacklogPageProps {
  projectId?: string
}

export default function BacklogPage({ projectId: propProjectId }: BacklogPageProps) {
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

  const [rowSelection, setRowSelection] = useState<RowSelectionState>({})

  // Dialog states
  const [planToSprintOpen, setPlanToSprintOpen] = useState(false)

  const [createOpen, setCreateOpen] = useState(false)
  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [selectedRequirement, setSelectedRequirement] = useState<Requirement | null>(null)
  const [parentRequirementId, setParentRequirementId] = useState<string | undefined>(undefined)
  const { data: response, isLoading, refetch } = useQuery({
    queryKey: ["backlog", projectId, pagination.pageIndex, pagination.pageSize, debouncedSearch, statusFilter, priorityFilter, assigneeFilter],
    queryFn: () => {
      if (!projectId) throw new Error("Project ID is required")
      return collaborationApi.listBacklog(projectId, {
        page: pagination.pageIndex + 1,
        page_size: pagination.pageSize,
        search: debouncedSearch,
        status: statusFilter || undefined,
        priority: priorityFilter || undefined,
        assignee_id: assigneeFilter || undefined,
      })
    },
    enabled: !!projectId,
  })

  const requirements = response?.items || []
  const totalCount = response?.pagination?.total || 0

  const selectedIds = Object.keys(rowSelection)

  const transitionRequirementMutation = useMutation({
    mutationFn: ({ requirementId, status }: { requirementId: string; status: string }) => {
      if (!projectId) throw new Error("Project ID is required")
      return collaborationApi.transitionRequirement(projectId, requirementId, status as RequirementStatus)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["backlog", projectId] })
      toast.success("Requirement status updated")
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: string } } }
      toast.error("Failed to update status", {
        description: err.response?.data?.error || "Unknown error occurred"
      })
    }
  })

  const updateRequirementPriorityMutation = useMutation({
    mutationFn: ({ requirement, priority }: { requirement: Requirement; priority: string }) => {
      if (!projectId) throw new Error("Project ID is required")
      return collaborationApi.updateRequirement(
        projectId,
        requirement.id,
        buildRequirementUpdateRequest(requirement, { priority: priority as CollabPriority })
      )
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["backlog", projectId] })
      toast.success("Requirement priority updated")
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: string } } }
      toast.error("Failed to update priority", {
        description: err.response?.data?.error || "Unknown error occurred"
      })
    }
  })

  const updateBacklogAssigneeMutation = useMutation({
    mutationFn: ({ requirement, assigneeId }: { requirement: Requirement; assigneeId: string }) => {
      if (!projectId) throw new Error("Project ID is required")
      return collaborationApi.updateRequirement(
        projectId,
        requirement.id,
        buildRequirementUpdateRequest(requirement, { assignee_id: assigneeId })
      )
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["backlog", projectId] })
      toast.success("Assignee updated")
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: string } } }
      toast.error("Failed to update assignee", {
        description: err.response?.data?.error || "Unknown error occurred"
      })
    }
  })

  const columns: ColumnDef<Requirement>[] = [
    {
      id: "select",
      header: ({ table }) => (
        <Checkbox
          checked={
            table.getIsAllPageRowsSelected() ||
            table.getIsSomePageRowsSelected()
          }
          onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
          aria-label="Select all"
          className="translate-y-0.5"
        />
      ),
      cell: ({ row }) => (
        <Checkbox
          checked={row.getIsSelected()}
          onCheckedChange={(value) => row.toggleSelected(!!value)}
          aria-label="Select row"
          className="translate-y-0.5"
        />
      ),
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: "title",
      header: "Title",
      cell: ({ row }) => {
        const depth = row.original.depth || 0
        return (
          <div className="flex items-center" style={{ paddingLeft: `${depth * 24}px` }}>
            {depth > 0 && <CornerDownRight className="h-4 w-4 mr-2 text-muted-foreground" />}
            <span className="font-medium truncate block max-w-75">{row.original.title}</span>
          </div>
        )
      },
    },
    {
      accessorKey: "priority",
      header: "Priority",
      cell: ({ row }) => (
        <InlinePriorityEditor
          currentPriority={row.original.priority}
          onPriorityChange={(priority) => updateRequirementPriorityMutation.mutate({ requirement: row.original, priority })}
        />
      ),
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => (
        <InlineStatusEditor
          currentStatus={row.original.status}
          entityType="requirement"
          statusOptions={RequirementStatusOptions}
          onStatusChange={(status) => transitionRequirementMutation.mutate({ requirementId: row.original.id, status })}
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
          onAssigneeChange={(assigneeId) => updateBacklogAssigneeMutation.mutate({ requirement: row.original, assigneeId })}
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
        const requirement = row.original
        return (
          <div className="flex justify-end">
            <DropdownMenu>
              <DropdownMenuTrigger render={<Button variant="ghost" size="icon-sm" onClick={(e) => e.stopPropagation()} />}>
                <MoreVertical />
              </DropdownMenuTrigger>

              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => {
                  setSelectedRequirement(requirement)
                  setEditOpen(true)
                }}>
                  <Pencil />
                  Edit
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => {
                  setParentRequirementId(requirement.id)
                  setCreateOpen(true)
                }}>
                  <Plus />
                  Create Child
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => {
                  setSelectedRequirement(requirement)
                  setDeleteOpen(true)
                }} variant="destructive" className="text-destructive hover:text-destructive hover:bg-destructive/10">
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
      {!propProjectId && <PageHeader items={[{ label: "Backlog", icon: Archive }]} />}

      {!propProjectId && (
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">Backlog</h1>
            <p className="text-sm text-muted-foreground">
              Prioritize requirements for future sprints.
            </p>
          </div>
        </div>
      )}

      {<DataTable
        columns={columns}
        data={requirements}
        isLoading={isLoading}
        manualPagination
        totalCount={totalCount}
        pagination={pagination}
        onPaginationChange={setPagination}
        onRefresh={() => refetch()}
        rowSelection={rowSelection}
        onRowSelectionChange={setRowSelection}
        emptyContent={
          <EmptyState
            title=""
            description="No results."
            icon={Logs}
            border={false}
          />
        }
        leftToolbar={() => (
          <>
            <Input
              className="max-w-xs"
              placeholder="Search..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
            <StatusFilter value={statusFilter} onChange={setStatusFilter} options={RequirementStatusOptions} />
            <PriorityFilter value={priorityFilter} onChange={setPriorityFilter} />
            <AssigneeFilter projectId={projectId!} value={assigneeFilter} onChange={setAssigneeFilter} />
          </>
        )}
        rightToolbar={() => (
          <>
            <Button onClick={() => {
              setParentRequirementId(undefined)
              setCreateOpen(true)
            }}>
              <Plus className="h-4 w-4" />
              Create Requirement
            </Button>
            <Button
              disabled={selectedIds.length === 0}
              onClick={() => setPlanToSprintOpen(true)}
            >
              <ArrowRight className="h-4 w-4" />
              Plan to Sprint ({selectedIds.length})
            </Button>
          </>
        )}
      />}

      <PlanToSprintDialog
        open={planToSprintOpen}
        onOpenChange={setPlanToSprintOpen}
        projectId={projectId}
        requirementIds={selectedIds.map(index => requirements[parseInt(index)]?.id).filter(Boolean)}
        onSuccess={() => setRowSelection({})}
      />
      <CreateRequirementDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        projectId={projectId}
        parentId={parentRequirementId}
        onSuccess={() => setCreateOpen(false)}
      />

      <EditRequirementDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        projectId={projectId}
        requirement={selectedRequirement}
        onSuccess={() => {
          setEditOpen(false)
          setSelectedRequirement(null)
        }}
      />

      {selectedRequirement && (
        <DeleteRequirementDialog
          open={deleteOpen}
          onOpenChange={setDeleteOpen}
          projectId={projectId}
          requirementId={selectedRequirement.id}
          onSuccess={() => {
            setDeleteOpen(false)
            setSelectedRequirement(null)
          }}
        />
      )}
    </div>
  )
}
