import { collaborationApi, CollabPriority, PlanningStatus, RequirementStatus, RequirementStatusOptions, type Requirement } from "@/api/collaboration"

import { AssigneeFilter, PriorityFilter, StatusFilter } from "@/components/collaborations/collab-filters"
import { InlineAssigneeEditor, InlinePriorityEditor, InlineStatusEditor } from "@/components/collaborations/inline-editors"
import { CreateRequirementDialog, DeleteRequirementDialog, EditRequirementDialog } from "@/components/collaborations/requirement-dialogs"
import { flattenTree, type TreeItem } from "@/components/collaborations/tree-utils"
import { DataTable } from "@/components/data-table/data-table"
import { PageHeader } from "@/components/layout/page-header"
import { getErrorMessage } from "@/lib/utils"
import { EmptyState } from "@/components/shared/empty-state"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
// Remove unused imports
import { useDebounce } from "@/hooks/use-debounce"
import { buildRequirementUpdateRequest } from "@/lib/collaboration-update-payloads"
import { formatDate } from "@/lib/utils"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import { ChevronDown, ChevronRight, FileText, MoreVertical, Pencil, Plus, RotateCcw, Trash2 } from "lucide-react"

import { useMemo, useState } from "react"
import { useParams } from "react-router-dom"
import { toast } from "sonner"

interface RequirementsPageProps {
  projectId?: string
  assigneeId?: string
  sprintId?: string
}

export default function RequirementsPage({ projectId: propProjectId, assigneeId, sprintId }: RequirementsPageProps) {
  const params = useParams<{ projectId: string }>()
  const projectId = propProjectId || params.projectId

  const queryClient = useQueryClient()

  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20, // Default larger size for tree view
  })
  const [search, setSearch] = useState("")
  const debouncedSearch = useDebounce(search, 300)
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set())

  const [statusFilter, setStatusFilter] = useState("")
  const [priorityFilter, setPriorityFilter] = useState("")
  const [assigneeFilter, setAssigneeFilter] = useState("")

  // Dialog states
  const [createOpen, setCreateOpen] = useState(false)
  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [selectedItem, setSelectedItem] = useState<Requirement | null>(null)
  const [parentForCreate, setParentForCreate] = useState<{ id: string; title: string } | undefined>(undefined)

  const { data: response, isLoading, refetch } = useQuery({

    queryKey: ["requirements", projectId, pagination.pageIndex, pagination.pageSize, debouncedSearch, assigneeId, sprintId, statusFilter, priorityFilter, assigneeFilter],
    queryFn: () => {
      if (!projectId) throw new Error("Project ID is required")
      return collaborationApi.listRequirements(projectId, {
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

  const requirements = response?.items || []
  const totalCount = response?.pagination?.total || 0

  // Flatten tree for display
  const tableData = useMemo(() => {
    return flattenTree(requirements, expandedIds)
  }, [requirements, expandedIds])

  const toggleExpand = (id: string) => {
    const newSet = new Set(expandedIds)
    if (newSet.has(id)) {
      newSet.delete(id)
    } else {
      newSet.add(id)
    }
    setExpandedIds(newSet)
  }

  const handleCreateChild = (item: Requirement) => {
    setParentForCreate({ id: item.id, title: item.title })
    setCreateOpen(true)
  }

  const handleEdit = (item: Requirement) => {
    setSelectedItem(item)
    setEditOpen(true)
  }

  const handleDelete = (item: Requirement) => {
    setSelectedItem(item)
    setDeleteOpen(true)
  }
  const returnToBacklogMutation = useMutation({
    mutationFn: (id: string) => {
      if (!projectId) throw new Error("Project ID is required")
      return collaborationApi.returnToBacklog(projectId, [id])
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["requirements", projectId] })
      toast.success("Returned to backlog")
    },
    onError: (error: unknown) => {
      toast.error("Failed to return to backlog", {
        description: getErrorMessage(error, "Unknown error occurred")
      })
    }
  })

  const handleReturnToBacklog = (item: Requirement) => {
    returnToBacklogMutation.mutate(item.id)
  }

  const transitionRequirementMutation = useMutation({
    mutationFn: ({ requirementId, status }: { requirementId: string; status: string }) => {
      if (!projectId) throw new Error("Project ID is required")
      return collaborationApi.transitionRequirement(projectId, requirementId, status as RequirementStatus)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["requirements", projectId] })
      toast.success("Requirement status updated")
    },
    onError: (error: unknown) => {
      toast.error("Failed to update status", {
        description: getErrorMessage(error, "Unknown error occurred")
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
      queryClient.invalidateQueries({ queryKey: ["requirements", projectId] })
      toast.success("Requirement priority updated")
    },
    onError: (error: unknown) => {
      toast.error("Failed to update priority", {
        description: getErrorMessage(error, "Unknown error occurred")
      })
    }
  })

  const updateRequirementAssigneeMutation = useMutation({
    mutationFn: ({ requirement, assigneeId }: { requirement: Requirement; assigneeId: string }) => {
      if (!projectId) throw new Error("Project ID is required")
      return collaborationApi.updateRequirement(
        projectId,
        requirement.id,
        buildRequirementUpdateRequest(requirement, { assignee_id: assigneeId })
      )
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["requirements", projectId] })
      toast.success("Requirement assignee updated")
    },
    onError: (error: unknown) => {
      toast.error("Failed to update assignee", {
        description: getErrorMessage(error, "Unknown error occurred")
      })
    }
  })

  const columns: ColumnDef<TreeItem<Requirement>>[] = [
    {
      accessorKey: "title",
      header: () => <div className="ml-6">Title</div>,
      cell: ({ row }) => {
        const item = row.original
        const hasChildren = item.children && item.children.length > 0
        const isExpanded = expandedIds.has(item.id)

        return (
          <div className="flex items-center gap-2" style={{ paddingLeft: `${item.depth * 20}px` }}>
            {hasChildren ? (
              <Button
                variant="ghost"
                size="icon-sm"
                className="h-4 w-4 p-0 hover:bg-transparent"
                onClick={(e) => {
                  e.stopPropagation()
                  toggleExpand(item.id)
                }}
              >
                {isExpanded ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
              </Button>
            ) : (
              <div className="w-4" /> // Spacer
            )}
            <span className="truncate font-medium">
              {item.title}
            </span>
            <div className="text-[10px] text-muted-foreground font-mono ml-2">
              {item.id.slice(0, 8)}
            </div>
          </div>
        )
      },
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
      accessorKey: "assignee_id",
      header: "Assignee",
      cell: ({ row }) => (
        <InlineAssigneeEditor
          projectId={projectId!}
          currentAssigneeId={row.original.assignee_id}
          onAssigneeChange={(assigneeId) => updateRequirementAssigneeMutation.mutate({ requirement: row.original, assigneeId })}
        />
      ),
    },
    {
      accessorKey: "created_at",
      header: "Created At",
      cell: ({ row }) => <span className="text-xs text-muted-foreground">{formatDate(row.original.created_at)}</span>,
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
              <DropdownMenuContent align="end" className="w-fit">
                {item.depth === 0 && (
                  <DropdownMenuItem onClick={() => handleCreateChild(item)}>
                    <Plus />
                    Add Child
                  </DropdownMenuItem>
                )}
                <DropdownMenuItem onClick={() => handleEdit(item)}>
                  <Pencil />
                  Edit
                </DropdownMenuItem>
                {item.planning_status !== PlanningStatus.BACKLOG && (
                  <DropdownMenuItem onClick={() => handleReturnToBacklog(item)}>
                    <RotateCcw />
                    Return to Backlog
                  </DropdownMenuItem>
                )}
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
      {!propProjectId && <PageHeader items={[{ label: "Requirements", icon: FileText }]} />}

      {!propProjectId && (
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">Requirements</h1>
            <p className="text-sm text-muted-foreground">
              Manage product requirements and specifications.
            </p>
          </div>
        </div>
      )}




      {<DataTable
        columns={columns}
        data={tableData}
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
            icon={FileText}
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
          <Button onClick={() => { setParentForCreate(undefined); setCreateOpen(true) }}>
            <Plus className="h-4 w-4" />
            New Requirement
          </Button>
        )}
      />}

      <CreateRequirementDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        projectId={projectId}
        parentId={parentForCreate?.id}
        parentTitle={parentForCreate?.title}
      />

      <EditRequirementDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        projectId={projectId}
        requirement={selectedItem}
      />

      {selectedItem && (
        <DeleteRequirementDialog
          open={deleteOpen}
          onOpenChange={setDeleteOpen}
          projectId={projectId}
          requirementId={selectedItem.id}
          onSuccess={() => {
            setDeleteOpen(false)
            setSelectedItem(null)
          }}
        />
      )}
    </div>
  )
}
