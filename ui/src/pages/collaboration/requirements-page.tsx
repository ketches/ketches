import { collaborationApi, type Requirement, PlanningStatus } from "@/api/collaboration"

import { DataTable } from "@/components/data-table/data-table"
import { PageHeader } from "@/components/layout/page-header"
import { StatusBadge, PriorityBadge } from "@/components/collaboration/collab-badges"
import { CreateRequirementDialog, EditRequirementDialog, DeleteRequirementDialog } from "@/components/collaboration/requirement-dialogs"
import { flattenTree, type TreeItem } from "@/components/collaboration/tree-utils"
import { EmptyState } from "@/components/shared/empty-state"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
// Remove unused imports
import { useDebounce } from "@/hooks/use-debounce"
import { formatDate } from "@/lib/utils"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import { ChevronDown, ChevronRight, FileText, MoreHorizontal, Pencil, Plus, RotateCcw, Trash2 } from "lucide-react"

import { useMemo, useState } from "react"
import { toast } from "sonner"
import { useParams } from "react-router-dom"

interface RequirementsPageProps {
  projectId?: string
}

export default function RequirementsPage({ projectId: propProjectId }: RequirementsPageProps) {
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

  // Dialog states
  const [createOpen, setCreateOpen] = useState(false)
  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [selectedItem, setSelectedItem] = useState<Requirement | null>(null)
  const [parentForCreate, setParentForCreate] = useState<{ id: string; title: string } | undefined>(undefined)

  const { data: response, isLoading } = useQuery({

    queryKey: ["requirements", projectId, pagination.pageIndex, pagination.pageSize, debouncedSearch],
    queryFn: () => {
      if (!projectId) throw new Error("Project ID is required")
      return collaborationApi.listRequirements(projectId, {
        page: pagination.pageIndex + 1,
        page_size: pagination.pageSize,
        search: debouncedSearch,
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
      const err = error as { response?: { data?: { error?: string } } }
      toast.error("Failed to return to backlog", {
        description: err.response?.data?.error || "Unknown error occurred"
      })
    }
  })

  const handleReturnToBacklog = (item: Requirement) => {
    returnToBacklogMutation.mutate(item.id)
  }

  const columns: ColumnDef<TreeItem<Requirement>>[] = [
    {
      accessorKey: "title",
      header: "Title",
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
      cell: ({ row }) => <StatusBadge status={row.original.status} />,
    },
    {
      accessorKey: "priority",
      header: "Priority",
      cell: ({ row }) => <PriorityBadge priority={row.original.priority} />,
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
              <DropdownMenuTrigger className="flex h-8 w-8 items-center justify-center rounded-md hover:bg-muted transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2">
                <MoreHorizontal className="h-4 w-4" />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                {item.depth === 0 && (
                   <DropdownMenuItem onClick={() => handleCreateChild(item)}>
                    <Plus className="mr-2 h-3.5 w-3.5" />
                    Add Child
                  </DropdownMenuItem>
                )}
                  <DropdownMenuItem onClick={() => handleEdit(item)}>
                    <Pencil className="mr-2 h-3.5 w-3.5" />
                    Edit
                  </DropdownMenuItem>
                  {item.planning_status !== PlanningStatus.BACKLOG && (
                    <DropdownMenuItem onClick={() => handleReturnToBacklog(item)}>
                      <RotateCcw className="mr-2 h-3.5 w-3.5" />
                      Return to Backlog
                    </DropdownMenuItem>
                  )}
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




      {!isLoading && requirements.length === 0 ? (
        <EmptyState
          title="No requirements found"
          description="Create your first requirement to get started."
          icon={FileText}
          actionText="Create Requirement"
          onAction={() => { setParentForCreate(undefined); setCreateOpen(true) }}
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
            <Button onClick={() => { setParentForCreate(undefined); setCreateOpen(true) }}>
              <Plus className="mr-2 h-4 w-4" />
              New Requirement
            </Button>
          </div>
          <DataTable
            columns={columns}
            data={tableData}
            isLoading={isLoading}
            manualPagination
            totalCount={totalCount}
            pagination={pagination}
            onPaginationChange={setPagination}
          />
        </>
      )}

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
