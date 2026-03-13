import { collaborationApi, CollabPriority, CollabPriorityOptions, TaskStatus, TaskStatusOptions, type CreateTaskRequest, type Task, type UpdateTaskRequest } from "@/api/collaboration"
import { Button } from "@/components/ui/button"
import { Combobox, ComboboxContent, ComboboxInput, ComboboxItem, ComboboxList } from "@/components/ui/combobox"
import { Input } from "@/components/ui/input"
import { Sheet, SheetContent, SheetDescription, SheetFooter, SheetHeader, SheetTitle } from "@/components/ui/sheet"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useEffect, useMemo, useState } from "react"
import { toast } from "sonner"
import { Field, FieldContent, FieldLabel } from "../ui/field"
import { RichTextEditor } from "./rich-text-editor"

interface CreateTaskDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectId: string
  parentId?: string
  parentTitle?: string
  onSuccess?: () => void
  defaultSprintId?: string
}

export function CreateTaskDialog({
  open,
  onOpenChange,
  projectId,
  parentId,
  parentTitle,
  onSuccess,
  defaultSprintId
}: CreateTaskDialogProps) {
  const queryClient = useQueryClient()
  const [formData, setFormData] = useState<CreateTaskRequest>({
    title: "",
    description: "",
    status: TaskStatus.TODO,
    priority: CollabPriority.P2,
    parent_task_id: parentId,
    sprint_id: defaultSprintId || "",
  })

  useEffect(() => {
    if (open) {
      setFormData({
        title: "",
        description: "",
        status: TaskStatus.TODO,
        priority: CollabPriority.P2,
        parent_task_id: parentId,
        sprint_id: defaultSprintId || "",
      })
    }
  }, [open, parentId, defaultSprintId])

  // Sync defaultSprintId changes
  useEffect(() => {
    if (defaultSprintId) {
      setFormData(prev => ({ ...prev, sprint_id: defaultSprintId }))
    }
  }, [defaultSprintId])

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

  const { data: sprintsData } = useQuery({
    queryKey: ["sprints", projectId, "all"],
    queryFn: () => collaborationApi.listSprints(projectId, { page: 1, page_size: 100 }),
    enabled: !!projectId,
  })
  const sprintItems = useMemo(() => {
    return (sprintsData?.items ?? []).map(s => ({ label: s.name, value: s.id }))
  }, [sprintsData?.items])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!formData.sprint_id) {
      toast.error("Sprint is required")
      return
    }
    mutation.mutate(formData)
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="sm:max-w-xl overflow-y-auto">
        <SheetHeader>
          <SheetTitle>{parentId ? "Create Child Task" : "Create Task"}</SheetTitle>
          <SheetDescription>
            {parentId ? `Add a sub-task to "${parentTitle}"` : "Add a new task to the project."}
          </SheetDescription>
        </SheetHeader>

        <div className="grid flex-1 auto-rows-min gap-4 px-4">
          <Field>
            <FieldLabel>Title</FieldLabel>
            <FieldContent>
              <Input
                id="title"
                value={formData.title}
                onChange={(e) => setFormData({ ...formData, title: e.target.value })}
                placeholder="Task title"
                required
              />
            </FieldContent>
          </Field>

          <Field>
            <FieldLabel>Sprint</FieldLabel>
            <FieldContent>
              <Combobox
                value={formData.sprint_id || ""}
                onValueChange={(val) => val && setFormData({ ...formData, sprint_id: val })}
                itemToStringLabel={(item) => sprintItems.find(opt => opt.value === item)?.label || item}
              >
                <ComboboxInput placeholder="Select sprint" />
                <ComboboxContent>
                  <ComboboxList>
                    {sprintItems.map((opt) => (
                      <ComboboxItem key={opt.value} value={opt.value}>
                        {opt.label}
                      </ComboboxItem>
                    ))}
                  </ComboboxList>
                </ComboboxContent>
              </Combobox>
            </FieldContent>
          </Field>

          <Field>
            <FieldLabel>Description</FieldLabel>
            <FieldContent>
              <RichTextEditor
                value={formData.description || ""}
                onChange={(json) => setFormData({ ...formData, description: json })}
                placeholder="Detailed description..."
              />
            </FieldContent>
          </Field>

          <div className="grid grid-cols-2 gap-4">
            <Field>
              <FieldLabel>Status</FieldLabel>
              <FieldContent>
                <Combobox
                  value={formData.status}
                  onValueChange={(val) => val && setFormData({ ...formData, status: val as TaskStatus })}
                  itemToStringLabel={(item) => TaskStatusOptions.find(opt => opt.value === item)?.label || item}
                >
                  <ComboboxInput placeholder="Select status" />
                  <ComboboxContent>
                    <ComboboxList>
                      {TaskStatusOptions.map((opt) => (
                        <ComboboxItem key={opt.value} value={opt.value}>
                          {opt.label}
                        </ComboboxItem>
                      ))}
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
              </FieldContent>
            </Field>

            <Field>
              <FieldLabel>Priority</FieldLabel>
              <FieldContent>
                <Combobox
                  value={formData.priority}
                  onValueChange={(val) => val && setFormData({ ...formData, priority: val as CollabPriority })}
                  itemToStringLabel={(item) => CollabPriorityOptions.find(opt => opt.value === item)?.label || item}
                >
                  <ComboboxInput placeholder="Select priority" />
                  <ComboboxContent>
                    <ComboboxList>
                      {CollabPriorityOptions.map((opt) => (
                        <ComboboxItem key={opt.value} value={opt.value}>
                          {opt.label}
                        </ComboboxItem>
                      ))}
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
              </FieldContent>
            </Field>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <Field>
              <FieldLabel>Due Date</FieldLabel>
              <FieldContent>
                <Input
                  id="due_date"
                  type="date"
                  value={formData.due_date ? formData.due_date.split('T')[0] : ''}
                  onChange={(e) => setFormData({ ...formData, due_date: e.target.value ? new Date(e.target.value).toISOString() : undefined })}
                />
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel>Estimate (hours)</FieldLabel>
              <FieldContent>
                <Input
                  id="estimate"
                  type="number"
                  min="0"
                  step="0.5"
                  value={formData.estimate_hours || ''}
                  onChange={(e) => setFormData({ ...formData, estimate_hours: e.target.value ? parseFloat(e.target.value) : undefined })}
                />
              </FieldContent>
            </Field>
          </div>
        </div>

        <SheetFooter>
          <div className="flex w-full items-center justify-end space-x-2">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={mutation.isPending || !formData.title} onClick={handleSubmit}>
              {mutation.isPending ? "Creating..." : "Create"}
            </Button>
          </div>
        </SheetFooter>
      </SheetContent>
    </Sheet>
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
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="sm:max-w-xl overflow-y-auto">
        <form onSubmit={handleSubmit} className="space-y-4">
          <SheetHeader>
            <SheetTitle>Edit Task</SheetTitle>
            <SheetDescription>
              Update task details.
            </SheetDescription>
          </SheetHeader>

          <Field>
            <FieldLabel>Title</FieldLabel>
            <FieldContent>
              <Input
                id="edit-title"
                value={formData.title}
                onChange={(e) => setFormData({ ...formData, title: e.target.value })}
                placeholder="Task title"
                required
              />
            </FieldContent>
          </Field>

          <Field>
            <FieldLabel>Description</FieldLabel>
            <FieldContent>
              <RichTextEditor
                value={formData.description || ""}
                onChange={(json) => setFormData({ ...formData, description: json })}
                placeholder="Detailed description..."
              />
            </FieldContent>
          </Field>

          <div className="grid grid-cols-2 gap-4">
            <Field>
              <FieldLabel>Status</FieldLabel>
              <FieldContent>
                <Combobox
                  value={formData.status}
                  onValueChange={(val) => val && setFormData({ ...formData, status: val as TaskStatus })}
                  itemToStringLabel={(item) => TaskStatusOptions.find(opt => opt.value === item)?.label || item}
                >
                  <ComboboxInput placeholder="Select status" />
                  <ComboboxContent>
                    <ComboboxList>
                      {TaskStatusOptions.map((opt) => (
                        <ComboboxItem key={opt.value} value={opt.value}>
                          {opt.label}
                        </ComboboxItem>
                      ))}
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
              </FieldContent>
            </Field>

            <Field>
              <FieldLabel>Priority</FieldLabel>
              <FieldContent>
                <Combobox
                  value={formData.priority}
                  onValueChange={(val) => val && setFormData({ ...formData, priority: val as CollabPriority })}
                  itemToStringLabel={(item) => CollabPriorityOptions.find(opt => opt.value === item)?.label || item}
                >
                  <ComboboxInput placeholder="Select priority" />
                  <ComboboxContent>
                    <ComboboxList>
                      {CollabPriorityOptions.map((opt) => (
                        <ComboboxItem key={opt.value} value={opt.value}>
                          {opt.label}
                        </ComboboxItem>
                      ))}
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
              </FieldContent>
            </Field>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <Field>
              <FieldLabel>Due Date</FieldLabel>
              <FieldContent>
                <Input
                  id="edit-due_date"
                  type="date"
                  value={formData.due_date ? formData.due_date.split('T')[0] : ''}
                  onChange={(e) => setFormData({ ...formData, due_date: e.target.value ? new Date(e.target.value).toISOString() : undefined })}
                />
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel>Estimate (hours)</FieldLabel>
              <FieldContent>
                <Input
                  id="edit-estimate"
                  type="number"
                  min="0"
                  step="0.5"
                  value={formData.estimate_hours || ''}
                  onChange={(e) => setFormData({ ...formData, estimate_hours: e.target.value ? parseFloat(e.target.value) : undefined })}
                />
              </FieldContent>
            </Field>
          </div>

          <SheetFooter>
            <div className="flex w-full items-center justify-end space-x-2">
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={mutation.isPending}>
                {mutation.isPending ? "Save Changes" : "Save"}
              </Button>
            </div>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  )
}
