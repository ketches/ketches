import { collaborationApi, CollabPriority, TaskStatus, TaskStatusOptions, type Task } from "@/api/collaboration"

import { DueDateBadge } from "@/components/collaborations/collab-badges"
import { AssigneeFilter, PriorityFilter, StatusFilter } from "@/components/collaborations/collab-filters"
import { InlineAssigneeEditor, InlinePriorityEditor, InlineStatusEditor } from "@/components/collaborations/inline-editors"
import { KanbanBoard } from "@/components/collaborations/kanban-board"
import { CreateTaskDialog, EditTaskDialog } from "@/components/collaborations/task-dialogs"
import { flattenTree, type TreeItem } from "@/components/collaborations/tree-utils"
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
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useDebounce } from "@/hooks/use-debounce"
import { buildTaskUpdateRequest } from "@/lib/collaboration-update-payloads"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import { CheckSquare, ChevronDown, ChevronRight, Columns3, ListTodo, ListTree, MoreVertical, Pencil, Plus, RefreshCw, Trash2 } from "lucide-react"

import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { useMemo, useState } from "react"
import { useParams } from "react-router-dom"
import { toast } from "sonner"

interface TasksPageProps {
  projectId?: string
  viewMode?: "list" | "kanban"
  onViewModeChange?: (viewMode: "list" | "kanban") => void
  assigneeId?: string
  sprintId?: string
}

export default function TasksPage({ projectId: propProjectId, viewMode = "list", onViewModeChange, assigneeId, sprintId }: TasksPageProps) {
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

  const [statusFilter, setStatusFilter] = useState("")
  const [priorityFilter, setPriorityFilter] = useState("")
  const [assigneeFilter, setAssigneeFilter] = useState("")

  // Dialog states
  const [createOpen, setCreateOpen] = useState(false)
  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [selectedItem, setSelectedItem] = useState<Task | null>(null)
  const [parentForCreate, setParentForCreate] = useState<{ id: string; title: string } | undefined>(undefined)

  const { data: response, isLoading, isFetching, refetch } = useQuery({

    queryKey: ["tasks", projectId, pagination.pageIndex, pagination.pageSize, debouncedSearch, assigneeId, sprintId, statusFilter, priorityFilter, assigneeFilter],
    queryFn: () => {
      if (!projectId) throw new Error("Project ID is required")
      return collaborationApi.listTasks(projectId, {
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

  const transitionTaskMutation = useMutation({
    mutationFn: ({ taskId, status }: { taskId: string; status: string }) => {
      if (!projectId) throw new Error("Project ID is required")
      return collaborationApi.transitionTask(projectId, taskId, status as TaskStatus)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tasks", projectId] })
      toast.success("Task status updated")
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: string } } }
      toast.error("Failed to update status", {
        description: err.response?.data?.error || "Unknown error occurred"
      })
    }
  })

  const updateTaskPriorityMutation = useMutation({
    mutationFn: ({ task, priority }: { task: Task; priority: string }) => {
      if (!projectId) throw new Error("Project ID is required")
      return collaborationApi.updateTask(
        projectId,
        task.id,
        buildTaskUpdateRequest(task, { priority: priority as CollabPriority })
      )
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tasks", projectId] })
      toast.success("Task priority updated")
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: string } } }
      toast.error("Failed to update priority", {
        description: err.response?.data?.error || "Unknown error occurred"
      })
    }
  })

  const updateTaskAssigneeMutation = useMutation({
    mutationFn: ({ task, assigneeId }: { task: Task; assigneeId: string }) => {
      if (!projectId) throw new Error("Project ID is required")
      return collaborationApi.updateTask(
        projectId,
        task.id,
        buildTaskUpdateRequest(task, { assignee_id: assigneeId })
      )
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tasks", projectId] })
      toast.success("Task assignee updated")
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: string } } }
      toast.error("Failed to update assignee", {
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
      cell: ({ row }) => (
        <InlineStatusEditor
          currentStatus={row.original.status}
          entityType="task"
          statusOptions={TaskStatusOptions}
          onStatusChange={(status) => transitionTaskMutation.mutate({ taskId: row.original.id, status })}
        />
      ),
    },
    {
      accessorKey: "priority",
      header: "Priority",
      cell: ({ row }) => (
        <InlinePriorityEditor
          currentPriority={row.original.priority}
          onPriorityChange={(priority) => updateTaskPriorityMutation.mutate({ task: row.original, priority })}
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
          onAssigneeChange={(assigneeId) => updateTaskAssigneeMutation.mutate({ task: row.original, assigneeId })}
        />
      ),
    },
    {
      accessorKey: "estimate_hours",
      header: "Estimate",
      cell: ({ row }) => row.original.estimate_hours ? <span className="text-xs">{row.original.estimate_hours}h</span> : <span className="text-xs text-muted-foreground">-</span>,
    },
    {
      accessorKey: "due_date",
      header: "Due",
      cell: ({ row }) => <DueDateBadge dueDate={row.original.due_date} />,
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

  const taskViewTabs = (
    <Tabs
      value={viewMode}
      onValueChange={(value) => {
        if (value === "list" || value === "kanban") {
          onViewModeChange?.(value)
        }
      }}
      className="h-7"
    >
      <TabsList>
        <TabsTrigger value="list">
          <ListTree className="h-3.5 w-3.5" />
        </TabsTrigger>
        <TabsTrigger value="kanban">
          <Columns3 className="h-3.5 w-3.5" />
        </TabsTrigger>
      </TabsList>
    </Tabs>
  )

  if (!projectId) return null

  return (
    <div className="flex flex-col h-full gap-4">
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




      {viewMode === "kanban" ? (
        <>
          <div className="flex flex-wrap items-center justify-between gap-4 w-full">
            <div className="flex flex-1 items-center gap-2">
              <Input
                className="max-w-xs"
                placeholder="Search..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
              />
              <StatusFilter value={statusFilter} onChange={setStatusFilter} options={TaskStatusOptions} />
              <PriorityFilter value={priorityFilter} onChange={setPriorityFilter} />
              <AssigneeFilter projectId={projectId!} value={assigneeFilter} onChange={setAssigneeFilter} />
            </div>
            <div className="ml-auto flex items-center gap-2">
              {taskViewTabs}
              <Button
                variant="secondary"
                size="icon"
                aria-label="Refresh tasks"
                disabled={isLoading}
                onClick={() => refetch()}
              >
                <RefreshCw className={isFetching && !isLoading ? "animate-spin" : undefined} />
              </Button>
              <Button onClick={() => { setParentForCreate(undefined); setCreateOpen(true) }}>
                <Plus className="h-4 w-4" />
                New Task
              </Button>
            </div>
          </div>
          {!isLoading && tasks.length === 0 ? (
            <EmptyState
              title=""
              description="No results."
              icon={ListTodo}
            />
          ) : (
            <KanbanBoard
              tasks={tasks}
              projectId={projectId}
              onCreateChild={handleCreateChild}
              onEdit={handleEdit}
              onDelete={handleDelete}
            />
          )}
        </>
      ) : (
        <DataTable
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
              icon={CheckSquare}
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
              <StatusFilter value={statusFilter} onChange={setStatusFilter} options={TaskStatusOptions} />
              <PriorityFilter value={priorityFilter} onChange={setPriorityFilter} />
              <AssigneeFilter projectId={projectId!} value={assigneeFilter} onChange={setAssigneeFilter} />
            </>
          )}
          rightToolbar={() => (
            <>
              {taskViewTabs}
              <Button onClick={() => { setParentForCreate(undefined); setCreateOpen(true) }}>
                <Plus className="h-4 w-4" />
                New Task
              </Button>
            </>
          )}
        />
      )}

      <CreateTaskDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        projectId={projectId}
        parentId={parentForCreate?.id}
        parentTitle={parentForCreate?.title}
        defaultSprintId={sprintId}
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
