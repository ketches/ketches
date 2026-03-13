import { collaborationApi, TaskStatusOptions, type Task, type TaskStatus } from "@/api/collaboration"
import { DueDateBadge, PriorityBadge } from "@/components/collaboration/collab-badges"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { DndContext, DragOverlay, PointerSensor, useDraggable, useDroppable, useSensor, useSensors, type DragEndEvent, type DragStartEvent } from "@dnd-kit/core"
import { useMutation, useQueryClient } from "@tanstack/react-query"
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

function KanbanColumn({ status, label, children }: { status: string; label: string; children: React.ReactNode }) {
  const { setNodeRef, isOver } = useDroppable({ id: status })
  return (
    <div
      ref={setNodeRef}
      className={`flex flex-1 flex-col min-w-56 rounded-lg border bg-muted/50 ${isOver ? "ring-2 ring-primary/50" : ""}`}
    >
      <div className="flex items-center justify-between px-3 py-2 border-b">
        <span className="text-sm font-medium">{label}</span>
      </div>
      <div className="flex flex-col gap-2 p-2 overflow-y-auto min-h-24 flex-1">
        {children}
      </div>
    </div>
  )
}

function KanbanCard({
  task,
  onCreateChild,
  onEdit,
  onDelete,
}: {
  task: Task
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
      className={`group relative p-3 cursor-grab active:cursor-grabbing space-y-2 ${isDragging ? "opacity-50" : ""}`}
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
          {task.assignee_id || "Unassigned"}
        </span>
        <span className="text-[10px] text-muted-foreground font-mono">{task.id.slice(0, 8)}</span>
      </div>
    </Card>
  )
}

function DragOverlayCard({ task }: { task: Task }) {
  return (
    <Card className="p-3 shadow-lg space-y-2 w-56">
      <div className="text-sm font-medium line-clamp-2">{task.title}</div>
      <div className="flex items-center gap-2">
        <PriorityBadge priority={task.priority} />
        <DueDateBadge dueDate={task.due_date} />
      </div>
      <div className="flex items-center justify-between gap-2">
        <span className="text-xs text-muted-foreground truncate">
          {task.assignee_id || "Unassigned"}
        </span>
        <span className="text-[10px] text-muted-foreground font-mono">{task.id.slice(0, 8)}</span>
      </div>
    </Card>
  )
}

export function KanbanBoard({ tasks, projectId, onCreateChild, onEdit, onDelete }: KanbanBoardProps) {
  const queryClient = useQueryClient()
  const [activeTask, setActiveTask] = useState<Task | null>(null)

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
      const err = error as { response?: { data?: { error?: string } } }
      toast.error("Transition failed", {
        description: err.response?.data?.error || "This status transition is not allowed",
      })
      queryClient.invalidateQueries({ queryKey: ["tasks", projectId] })
    },
  })

  const handleDragStart = (event: DragStartEvent) => {
    const task = event.active.data.current?.task as Task | undefined
    setActiveTask(task ?? null)
  }

  const handleDragEnd = (event: DragEndEvent) => {
    setActiveTask(null)
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

  return (
    <DndContext sensors={sensors} onDragStart={handleDragStart} onDragEnd={handleDragEnd}>
      <div className="flex w-full gap-4 overflow-x-auto pb-4">
        {TaskStatusOptions.map((opt) => (
          <KanbanColumn key={opt.value} status={opt.value} label={opt.label}>
            {groupedTasks[opt.value]?.map((task) => (
              <KanbanCard key={task.id} task={task} onCreateChild={onCreateChild} onEdit={onEdit} onDelete={onDelete} />
            ))}
          </KanbanColumn>
        ))}
      </div>
      <DragOverlay>
        {activeTask ? <DragOverlayCard task={activeTask} /> : null}
      </DragOverlay>
    </DndContext>
  )
}
