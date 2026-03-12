import { collaborationApi, type Defect } from "@/api/collaboration"
import { DataTable } from "@/components/data-table/data-table"
import { PageHeader } from "@/components/layout/page-header"
import { StatusBadge, SeverityBadge } from "@/components/collaboration/collab-badges"
import { CreateDefectDialog, EditDefectDialog, DeleteDefectDialog, TransitionDefectDialog } from "@/components/collaboration/defect-dialogs"
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
import { useQuery } from "@tanstack/react-query"
import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import { MoreHorizontal, Pencil, Plus, Trash2, Bug, ArrowRightLeft } from "lucide-react"
import { useMemo, useState } from "react"
import { useParams } from "react-router-dom"

interface DefectsPageProps {
  projectId?: string
  assigneeId?: string
  sprintId?: string
}

export default function DefectsPage({ projectId: propProjectId, assigneeId, sprintId }: DefectsPageProps) {
  const params = useParams<{ projectId: string }>()
  const projectId = propProjectId || params.projectId

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
          <span className="font-medium truncate max-w-[400px]">{row.original.title}</span>
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
        const item = row.original
        return (
          <div className="flex justify-end">
            <DropdownMenu>
              <DropdownMenuTrigger className="flex h-8 w-8 items-center justify-center rounded-md hover:bg-muted transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2">
                <MoreHorizontal className="h-4 w-4" />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => handleTransition(item)}>
                  <ArrowRightLeft className="mr-2 h-3.5 w-3.5" />
                  Transition
                </DropdownMenuItem>
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
