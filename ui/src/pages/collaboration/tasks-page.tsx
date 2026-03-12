import { collaborationApi, type Task } from "@/api/collaboration"

import { DataTable } from "@/components/data-table/data-table"
import { PageHeader } from "@/components/layout/page-header"
import { StatusBadge, PriorityBadge } from "@/components/collaboration/collab-badges"
import { CreateTaskDialog, EditTaskDialog } from "@/components/collaboration/task-dialogs"
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
import { useDebounce } from "@/hooks/use-debounce"
import { formatDate } from "@/lib/utils"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import { ChevronDown, ChevronRight, CheckSquare, MoreHorizontal, Pencil, Plus, Trash2 } from "lucide-react"

import { useMemo, useState } from "react"
import { useParams } from "react-router-dom"
import { toast } from "sonner"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"

interface TasksPageProps {
  projectId?: string
}

export default function TasksPage({ projectId: propProjectId }: TasksPageProps) {
  const params = useParams<{ projectId: string }>()
  const projectId = propProjectId || params.projectId


  const queryClient = useQueryClient()

  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  })
  const [search, setSearch] = useState("")
  const debouncedSearch = useDebounce(search, 300)
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set())

  // Dialog states
  const [createOpen, setCreateOpen] = useState(false)
  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [selectedItem, setSelectedItem] = useState<Task | null>(null)
  const [parentForCreate, setParentForCreate] = useState<{ id: string; title: string } | undefined>(undefined)

  const { data: response, isLoading } = useQuery({

    queryKey: ["tasks", projectId, pagination.pageIndex, pagination.pageSize, debouncedSearch],
    queryFn: () => {
      if (!projectId) throw new Error("Project ID is required")
      return collaborationApi.listTasks(projectId, {
        page: pagination.pageIndex + 1,
        page_size: pagination.pageSize,
        search: debouncedSearch,
      })
    },
    enabled: !!projectId,
  })

  const tasks = response?.items || []
  const totalCount = response?.pagination?.total || 0

  const tableData = useMemo(() => {
    return flattenTree(tasks, expandedIds)
  }, [tasks, expandedIds])

  const toggleExpand = (id: string) => {
    const newSet = new Set(expandedIds)
    if (newSet.has(id)) {
      newSet.delete(id)
    } else {
      newSet.add(id)
    }
    setExpandedIds(newSet)
  }

  const deleteMutation = useMutation({
    mutationFn: (id: string) => {
      if (!projectId) throw new Error("Project ID is required")
      return collaborationApi.deleteTask(projectId, id)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tasks", projectId] })
      toast.success("Task deleted")
      setDeleteOpen(false)
      setSelectedItem(null)
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: string } } }
      toast.error("Failed to delete task", {
        description: err.response?.data?.error || "Unknown error occurred"
      })
    }


  })

  const handleCreateChild = (item: Task) => {
    setParentForCreate({ id: item.id, title: item.title })
    setCreateOpen(true)
  }

  const handleEdit = (item: Task) => {
    setSelectedItem(item)
    setEditOpen(true)
  }

  const handleDelete = (item: Task) => {
    setSelectedItem(item)
    setDeleteOpen(true)
  }

  const columns: ColumnDef<TreeItem<Task>>[] = [
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
              <div className="w-4" />
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
      accessorKey: "estimate_hours",
      header: "Estimate",
      cell: ({ row }) => row.original.estimate_hours ? <span className="text-xs">{row.original.estimate_hours}h</span> : <span className="text-xs text-muted-foreground">-</span>,
    },
    {
      accessorKey: "due_date",
      header: "Due",
      cell: ({ row }) => row.original.due_date ? <span className="text-xs">{formatDate(row.original.due_date)}</span> : <span className="text-xs text-muted-foreground">-</span>,
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
      {!propProjectId && <PageHeader items={[{ label: "Tasks", icon: CheckSquare }]} />}

      {!propProjectId && (
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">Tasks</h1>
            <p className="text-sm text-muted-foreground">
              Manage tasks and track progress.
            </p>
          </div>
        </div>
      )}




      {!isLoading && tasks.length === 0 ? (
        <EmptyState
          title="No tasks found"
          description="Create your first task to get started."
          icon={CheckSquare}
          actionText="Create Task"
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
              New Task
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

      <CreateTaskDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        projectId={projectId}
        parentId={parentForCreate?.id}
        parentTitle={parentForCreate?.title}
      />

      <EditTaskDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        projectId={projectId}
        task={selectedItem}
      />

      <AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Task?</AlertDialogTitle>
            <AlertDialogDescription>
              This will permanently delete "{selectedItem?.title}".
              {selectedItem?.depth === 0 && " All child tasks will also be deleted."}
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
