import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { collaborationApi, TaskStatus, CollabPriority, type Task, type CreateTaskRequest, type UpdateTaskRequest } from "@/api/collaboration"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useState, useEffect } from "react"
import { toast } from "sonner"
import { Combobox, ComboboxContent, ComboboxInput, ComboboxItem, ComboboxList } from "@/components/ui/combobox"

interface CreateTaskDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectId: string
  parentId?: string
  parentTitle?: string
  onSuccess?: () => void
}

export function CreateTaskDialog({
  open,
  onOpenChange,
  projectId,
  parentId,
  parentTitle,
  onSuccess
}: CreateTaskDialogProps) {
  const queryClient = useQueryClient()
  const [formData, setFormData] = useState<CreateTaskRequest>({
    title: "",
    description: "",
    status: TaskStatus.TODO,
    priority: CollabPriority.P2,
    parent_task_id: parentId,
  })

  useEffect(() => {
    if (open) {
      setFormData({
        title: "",
        description: "",
        status: TaskStatus.TODO,
        priority: CollabPriority.P2,
        parent_task_id: parentId,
      })
    }
  }, [open, parentId])

  const mutation = useMutation({
    mutationFn: (data: CreateTaskRequest) => {
      if (parentId) {
        return collaborationApi.createTaskChild(projectId, parentId, data)
      }
      return collaborationApi.createTask(projectId, data)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tasks", projectId] })
      toast.success(parentId ? "Child task created" : "Task created")
      onOpenChange(false)
      onSuccess?.()
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: string } } }
      toast.error("Failed to create task", {
        description: err.response?.data?.error || "Unknown error occurred"
      })
    }
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    mutation.mutate(formData)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>{parentId ? "Create Child Task" : "Create Task"}</DialogTitle>
          <DialogDescription>
            {parentId ? `Add a sub-task to "${parentTitle}"` : "Add a new task to the project."}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="title">Title</Label>
            <Input
              id="title"
              value={formData.title}
              onChange={(e) => setFormData({ ...formData, title: e.target.value })}
              placeholder="Task title"
              required
            />
          </div>
          
          <div className="space-y-2">
            <Label htmlFor="description">Description</Label>
            <Textarea
              id="description"
              value={formData.description || ""}
              onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              placeholder="Detailed description..."
              className="min-h-24"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>Status</Label>
              <Combobox
                value={formData.status}
                onValueChange={(val) => val && setFormData({ ...formData, status: val as TaskStatus })}
              >
                <ComboboxInput placeholder="Select status" />
                <ComboboxContent>
                  <ComboboxList>
                    {Object.values(TaskStatus).map((status) => (
                      <ComboboxItem key={status} value={status}>
                        {status.replace(/_/g, " ")}
                      </ComboboxItem>
                    ))}
                  </ComboboxList>
                </ComboboxContent>
              </Combobox>
            </div>

            <div className="space-y-2">
              <Label>Priority</Label>
              <Combobox
                value={formData.priority}
                onValueChange={(val) => val && setFormData({ ...formData, priority: val as CollabPriority })}
              >
                <ComboboxInput placeholder="Select priority" />
                <ComboboxContent>
                  <ComboboxList>
                    {Object.values(CollabPriority).map((priority) => (
                      <ComboboxItem key={priority} value={priority}>
                        {priority}
                      </ComboboxItem>
                    ))}
                  </ComboboxList>
                </ComboboxContent>
              </Combobox>
            </div>
          </div>
          
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="due_date">Due Date</Label>
              <Input
                id="due_date"
                type="date"
                value={formData.due_date ? formData.due_date.split('T')[0] : ''}
                onChange={(e) => setFormData({ ...formData, due_date: e.target.value ? new Date(e.target.value).toISOString() : undefined })}
              />
            </div>
             <div className="space-y-2">
              <Label htmlFor="estimate">Estimate (hours)</Label>
              <Input
                id="estimate"
                type="number"
                min="0"
                step="0.5"
                value={formData.estimate_hours || ''}
                onChange={(e) => setFormData({ ...formData, estimate_hours: e.target.value ? parseFloat(e.target.value) : undefined })}
              />
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={mutation.isPending}>
              {mutation.isPending ? "Creating..." : "Create"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

interface EditTaskDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectId: string
  task: Task | null
  onSuccess?: () => void
}

export function EditTaskDialog({
  open,
  onOpenChange,
  projectId,
  task,
  onSuccess
}: EditTaskDialogProps) {
  const queryClient = useQueryClient()
  const [formData, setFormData] = useState<UpdateTaskRequest>({
    title: "",
    description: "",
    status: TaskStatus.TODO,
    priority: CollabPriority.P2,
  })

  useEffect(() => {
    if (task && open) {
      setFormData({
        title: task.title,
        description: task.description,
        status: task.status,
        priority: task.priority,
        assignee_id: task.assignee_id,
        due_date: task.due_date,
        estimate_hours: task.estimate_hours,
        sprint_id: task.sprint_id,
        requirement_id: task.requirement_id,
      })
    }
  }, [task, open])

  const mutation = useMutation({
    mutationFn: (data: UpdateTaskRequest) => {
      if (!task) throw new Error("No task selected")
      return collaborationApi.updateTask(projectId, task.id, data)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tasks", projectId] })
      toast.success("Task updated")
      onOpenChange(false)
      onSuccess?.()
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: string } } }
      toast.error("Failed to update task", {
        description: err.response?.data?.error || "Unknown error occurred"
      })
    }
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    mutation.mutate(formData)
  }

  if (!task) return null

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>Edit Task</DialogTitle>
          <DialogDescription>
            Update task details.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="edit-title">Title</Label>
            <Input
              id="edit-title"
              value={formData.title}
              onChange={(e) => setFormData({ ...formData, title: e.target.value })}
              placeholder="Task title"
              required
            />
          </div>
          
          <div className="space-y-2">
            <Label htmlFor="edit-description">Description</Label>
            <Textarea
              id="edit-description"
              value={formData.description || ""}
              onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              placeholder="Detailed description..."
              className="min-h-24"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>Status</Label>
              <Combobox
                value={formData.status}
                onValueChange={(val) => val && setFormData({ ...formData, status: val as TaskStatus })}
              >
                <ComboboxInput placeholder="Select status" />
                <ComboboxContent>
                  <ComboboxList>
                    {Object.values(TaskStatus).map((status) => (
                      <ComboboxItem key={status} value={status}>
                        {status.replace(/_/g, " ")}
                      </ComboboxItem>
                    ))}
                  </ComboboxList>
                </ComboboxContent>
              </Combobox>
            </div>

            <div className="space-y-2">
              <Label>Priority</Label>
              <Combobox
                value={formData.priority}
                onValueChange={(val) => val && setFormData({ ...formData, priority: val as CollabPriority })}
              >
                <ComboboxInput placeholder="Select priority" />
                <ComboboxContent>
                  <ComboboxList>
                    {Object.values(CollabPriority).map((priority) => (
                      <ComboboxItem key={priority} value={priority}>
                        {priority}
                      </ComboboxItem>
                    ))}
                  </ComboboxList>
                </ComboboxContent>
              </Combobox>
            </div>
          </div>
          
          <div className="grid grid-cols-2 gap-4">
             <div className="space-y-2">
              <Label htmlFor="edit-due_date">Due Date</Label>
              <Input
                id="edit-due_date"
                type="date"
                value={formData.due_date ? formData.due_date.split('T')[0] : ''}
                onChange={(e) => setFormData({ ...formData, due_date: e.target.value ? new Date(e.target.value).toISOString() : undefined })}
              />
            </div>
             <div className="space-y-2">
              <Label htmlFor="edit-estimate">Estimate (hours)</Label>
              <Input
                id="edit-estimate"
                type="number"
                min="0"
                step="0.5"
                value={formData.estimate_hours || ''}
                onChange={(e) => setFormData({ ...formData, estimate_hours: e.target.value ? parseFloat(e.target.value) : undefined })}
              />
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={mutation.isPending}>
              {mutation.isPending ? "Save Changes" : "Save"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
