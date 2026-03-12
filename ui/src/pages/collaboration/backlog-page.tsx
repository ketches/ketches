import { collaborationApi, type Requirement } from "@/api/collaboration"
import { DataTable } from "@/components/data-table/data-table"
import { PageHeader } from "@/components/layout/page-header"
import { PriorityBadge, StatusBadge } from "@/components/collaboration/collab-badges"
import { PlanToSprintDialog } from "@/components/collaboration/backlog-dialogs"
import { EmptyState } from "@/components/shared/empty-state"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { useDebounce } from "@/hooks/use-debounce"
import { formatDate } from "@/lib/utils"
import { useQuery } from "@tanstack/react-query"
import { type ColumnDef, type PaginationState, type RowSelectionState } from "@tanstack/react-table"
import { Archive, ArrowRight, LayoutList, MoreHorizontal, Plus, Pencil, Trash2, CornerDownRight } from "lucide-react"
import { useState } from "react"
import { useParams } from "react-router-dom"

import { CreateRequirementDialog, EditRequirementDialog, DeleteRequirementDialog } from "@/components/collaboration/requirement-dialogs"
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

  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  })
  const [search, setSearch] = useState("")
  const debouncedSearch = useDebounce(search, 300)
  const [rowSelection, setRowSelection] = useState<RowSelectionState>({})

  // Dialog states
  const [planToSprintOpen, setPlanToSprintOpen] = useState(false)

  const [createOpen, setCreateOpen] = useState(false)
  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [selectedRequirement, setSelectedRequirement] = useState<Requirement | null>(null)
  const [parentRequirementId, setParentRequirementId] = useState<string | undefined>(undefined)
  const { data: response, isLoading } = useQuery({
    queryKey: ["backlog", projectId, pagination.pageIndex, pagination.pageSize, debouncedSearch],
    queryFn: () => {
      if (!projectId) throw new Error("Project ID is required")
      return collaborationApi.listBacklog(projectId, {
        page: pagination.pageIndex + 1,
        page_size: pagination.pageSize,
        search: debouncedSearch,
      })
    },
    enabled: !!projectId,
  })

  const requirements = response?.items || []
  const totalCount = response?.pagination?.total || 0

  const selectedIds = Object.keys(rowSelection)

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
          className="translate-y-[2px]"
        />
      ),
      cell: ({ row }) => (
        <Checkbox
          checked={row.getIsSelected()}
          onCheckedChange={(value) => row.toggleSelected(!!value)}
          aria-label="Select row"
          className="translate-y-[2px]"
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
            <span className="font-medium truncate block max-w-[300px]">{row.original.title}</span>
          </div>
        )
      },
    },
    {
      accessorKey: "priority",
      header: "Priority",
      cell: ({ row }) => <PriorityBadge priority={row.original.priority} />,
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <StatusBadge status={row.original.status} />,
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
          <DropdownMenu>
            <DropdownMenuTrigger className="h-8 w-8 p-0 inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 hover:bg-accent hover:text-accent-foreground">
              <span className="sr-only">Open menu</span>
              <MoreHorizontal className="h-4 w-4" />
            </DropdownMenuTrigger>

            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => {
                setSelectedRequirement(requirement)
                setEditOpen(true)
              }}>
                <Pencil className="mr-2 h-4 w-4" />
                Edit
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => {
                setParentRequirementId(requirement.id)
                setCreateOpen(true)
              }}>
                <Plus className="mr-2 h-4 w-4" />
                Create Child
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => {
                setSelectedRequirement(requirement)
                setDeleteOpen(true)
              }} className="text-destructive focus:text-destructive">
                <Trash2 className="mr-2 h-4 w-4" />
                Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )
      },
    },
  ]

  if (!projectId) return null

  return (
    <div className="flex flex-col h-full gap-6">
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

      {!isLoading && requirements.length === 0 ? (
        <EmptyState
          title="Backlog is empty"
          description="Requirements without a sprint will appear here."
          icon={LayoutList}
          actionText="Create Requirement"
          onAction={() => {
            setParentRequirementId(undefined)
            setCreateOpen(true)
          }}
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
            <div className="flex items-center gap-2">
              <Button onClick={() => {
                setParentRequirementId(undefined)
                setCreateOpen(true)
              }}>
                <Plus className="mr-2 h-4 w-4" />
                Create Requirement
              </Button>
              <Button 
                disabled={selectedIds.length === 0} 
                onClick={() => setPlanToSprintOpen(true)}
              >
                <ArrowRight className="mr-2 h-4 w-4" />
                Plan to Sprint ({selectedIds.length})
              </Button>
            </div>
          </div>
          <DataTable
            columns={columns}
            data={requirements}
            isLoading={isLoading}
            manualPagination
            totalCount={totalCount}
            pagination={pagination}
            onPaginationChange={setPagination}
            rowSelection={rowSelection}
            onRowSelectionChange={setRowSelection}
          />
        </>
      )}

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
