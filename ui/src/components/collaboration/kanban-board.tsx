import { collaborationApi, TaskStatusOptions, type Task, type TaskStatus } from "@/api/collaboration"
import { PriorityBadge } from "@/components/collaboration/collab-badges"
import { Card } from "@/components/ui/card"
import { DndContext, DragOverlay, PointerSensor, useDraggable, useDroppable, useSensor, useSensors, type DragEndEvent, type DragStartEvent } from "@dnd-kit/core"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useState } from "react"
import { toast } from "sonner"

interface KanbanBoardProps {
  tasks: Task[]
  projectId: string
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

function KanbanCard({ task }: { task: Task }) {
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({ id: task.id, data: { task } })
  const style = transform ? { transform: `translate(${transform.x}px, ${transform.y}px)` } : undefined

  return (
    <Card
      ref={setNodeRef}
      {...listeners}
      {...attributes}
      style={style}
      className={`p-3 cursor-grab active:cursor-grabbing space-y-1 ${isDragging ? "opacity-50" : ""}`}
    >
      <div className="text-sm font-medium line-clamp-2">{task.title}</div>
      <div className="flex items-center justify-between">
        <PriorityBadge priority={task.priority} />
        <span className="text-[10px] text-muted-foreground font-mono">{task.id.slice(0, 8)}</span>
      </div>
    </Card>
  )
}

function DragOverlayCard({ task }: { task: Task }) {
  return (
    <Card className="p-3 shadow-lg space-y-1 w-56">
      <div className="text-sm font-medium line-clamp-2">{task.title}</div>
      <div className="flex items-center justify-between">
        <PriorityBadge priority={task.priority} />
        <span className="text-[10px] text-muted-foreground font-mono">{task.id.slice(0, 8)}</span>
      </div>
    </Card>
  )
}

export function KanbanBoard({ tasks, projectId }: KanbanBoardProps) {
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
              <KanbanCard key={task.id} task={task} />
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