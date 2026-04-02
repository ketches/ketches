import { collaborationApi, TaskStatusOptions, type Task, type TaskStatus } from "@/api/collaboration"
import { projectsApi } from "@/api/projects"
import { DueDateBadge, PriorityBadge } from "@/components/collaborations/collab-badges"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { DndContext, DragOverlay, PointerSensor, useDraggable, useDroppable, useSensor, useSensors, type DragEndEvent, type DragStartEvent } from "@dnd-kit/core"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { getErrorMessage } from "@/lib/utils"
import type { LucideIcon } from "lucide-react"
import { MoreVertical, Pencil, Plus, Trash2 } from "lucide-react"
import { useState } from "react"
import { toast } from "sonner"

interface KanbanBoardProps {
  tasks: Task[]
  projectId: string
  onCreateChild?: (task: Task) => void
  onEdit?: (task: Task) => void
  onDelete?: (task: Task) => void
}

function KanbanColumn({ status, label, icon: Icon, color, children }: { status: string; label: string; icon: LucideIcon; color: string, children: React.ReactNode }) {
  const { setNodeRef, isOver } = useDroppable({ id: status })
  return (
    <div
      ref={setNodeRef}
      className={`bg-linear-to-b/increasing from-${color}-500/10 to-transparent data-[active=true]:bg-transparent flex flex-1 flex-col min-w-56 overflow-hidden rounded-lg border bg-muted/50 ${isOver ? "border-dashed border-primary/50 ring-1 ring-inset ring-primary/30" : ""}`}
    >
      <div className="flex items-center justify-between px-3 py-2">
        <span className="flex items-center gap-2 text-sm font-medium">
          <Icon className={`size-4 text-${color}-500`} />
          {label}
        </span>
      </div>
      <div className="flex min-h-24 flex-1 flex-col gap-2 overflow-x-hidden overflow-y-hidden p-2">
        {children}
      </div>
    </div>
  )
}

function KanbanCard({
  task,
  assigneeLabel,
  onCreateChild,
  onEdit,
  onDelete,
}: {
  task: Task
  assigneeLabel: string
  onCreateChild?: (task: Task) => void
  onEdit?: (task: Task) => void
  onDelete?: (task: Task) => void
}) {
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({ id: task.id, data: { task } })
  const style = transform ? { transform: `translate(${transform.x}px, ${transform.y}px)` } : undefined

  return (
    <Card
      ref={setNodeRef}
      {...listeners}
      {...attributes}
      style={style}
      className={`group relative w-full p-3 cursor-grab active:cursor-grabbing space-y-2 ${isDragging ? "opacity-0" : ""}`}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="text-sm font-medium line-clamp-2">{task.title}</div>
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button
                variant="ghost"
                size="icon-sm"
                className="h-6 w-6 opacity-0 transition-opacity group-hover:opacity-100"
                onPointerDown={(e) => e.stopPropagation()}
                onClick={(e) => e.stopPropagation()}
              >
                <MoreVertical className="size-3.5" />
              </Button>
            }
          />
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={() => onCreateChild?.(task)}>
              <Plus />
              Add Child
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => onEdit?.(task)}>
              <Pencil />
              Edit
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem variant="destructive" onClick={() => onDelete?.(task)}>
              <Trash2 />
              Delete
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
      <div className="flex items-center gap-2">
        <PriorityBadge priority={task.priority} />
        <DueDateBadge dueDate={task.due_date} />
      </div>
      <div className="flex items-center justify-between gap-2">
        <span className="text-xs text-muted-foreground truncate">
          {assigneeLabel}
        </span>
        <span className="text-[10px] text-muted-foreground font-mono">{task.id.slice(0, 8)}</span>
      </div>
    </Card>
  )
}

function DragOverlayCard({ task, assigneeLabel, width }: { task: Task; assigneeLabel: string; width?: number | null }) {
  return (
    <Card className="space-y-2 p-3 shadow-lg" style={width ? { width } : undefined}>
      <div className="text-sm font-medium line-clamp-2">{task.title}</div>
      <div className="flex items-center gap-2">
        <PriorityBadge priority={task.priority} />
        <DueDateBadge dueDate={task.due_date} />
      </div>
      <div className="flex items-center justify-between gap-2">
        <span className="text-xs text-muted-foreground truncate">
          {assigneeLabel}
        </span>
        <span className="text-[10px] text-muted-foreground font-mono">{task.id.slice(0, 8)}</span>
      </div>
    </Card>
  )
}

export function KanbanBoard({ tasks, projectId, onCreateChild, onEdit, onDelete }: KanbanBoardProps) {
  const queryClient = useQueryClient()
  const [activeTask, setActiveTask] = useState<Task | null>(null)
  const [activeTaskWidth, setActiveTaskWidth] = useState<number | null>(null)
  const { data: membersData } = useQuery({
    queryKey: ["project-members", projectId],
    queryFn: () => projectsApi.listMembers(projectId, { page: 1, page_size: 100 }),
    enabled: !!projectId,
  })

  const memberLabels = new Map(
    (membersData?.items ?? []).map((member) => [
      member.user_id,
      member.fullname || member.username,
    ])
  )

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } })
  )

  const transitionMutation = useMutation({
    mutationFn: ({ taskId, status }: { taskId: string; status: TaskStatus }) =>
      collaborationApi.transitionTask(projectId, taskId, status),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tasks", projectId] })
    },
    onError: (error: unknown) => {
      toast.error("Transition failed", {
        description: getErrorMessage(error, "This status transition is not allowed"),
      })
      queryClient.invalidateQueries({ queryKey: ["tasks", projectId] })
    },
  })

  const handleDragStart = (event: DragStartEvent) => {
    const task = event.active.data.current?.task as Task | undefined
    setActiveTask(task ?? null)
    setActiveTaskWidth(event.active.rect.current.initial?.width ?? null)
  }

  const handleDragEnd = (event: DragEndEvent) => {
    setActiveTask(null)
    setActiveTaskWidth(null)
    const { active, over } = event
    if (!over) return

    const taskId = active.id as string
    const newStatus = over.id as TaskStatus
    const task = active.data.current?.task as Task | undefined
    if (!task || task.status === newStatus) return

    transitionMutation.mutate({ taskId, status: newStatus })
  }

  // Group tasks by status
  const groupedTasks = TaskStatusOptions.reduce(
    (acc, opt) => {
      acc[opt.value] = tasks.filter((t) => t.status === opt.value)
      return acc
    },
    {} as Record<string, Task[]>
  )

  const getAssigneeLabel = (assigneeId?: string) => {
    if (!assigneeId) {
      return "Unassigned"
    }

    return memberLabels.get(assigneeId) || assigneeId
  }

  return (
    <DndContext sensors={sensors} onDragStart={handleDragStart} onDragEnd={handleDragEnd}>
      <div className="flex w-full gap-4 overflow-x-auto pb-4">
        {TaskStatusOptions.map((opt) => (
          <KanbanColumn key={opt.value} status={opt.value} label={opt.label} icon={opt.icon} color={opt.color}>
            {groupedTasks[opt.value]?.map((task) => (
              <KanbanCard
                key={task.id}
                task={task}
                assigneeLabel={getAssigneeLabel(task.assignee_id)}
                onCreateChild={onCreateChild}
                onEdit={onEdit}
                onDelete={onDelete}
              />
            ))}
          </KanbanColumn>
        ))}
      </div>
      <DragOverlay>
        {activeTask ? (
          <DragOverlayCard
            task={activeTask}
            assigneeLabel={getAssigneeLabel(activeTask.assignee_id)}
            width={activeTaskWidth}
          />
        ) : null}
      </DragOverlay>
    </DndContext >
  )
}
