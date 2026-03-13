import { collaborationApi, CollabPriority, CollabPriorityOptions, TaskStatus, TaskStatusOptions, type CreateTaskRequest, type Task, type UpdateTaskRequest } from "@/api/collaboration"
import { projectsApi } from "@/api/projects"
import { Button } from "@/components/ui/button"
import { Combobox, ComboboxContent, ComboboxInput, ComboboxItem, ComboboxList } from "@/components/ui/combobox"
import { Input } from "@/components/ui/input"
import { Sheet, SheetContent, SheetDescription, SheetFooter, SheetHeader, SheetTitle } from "@/components/ui/sheet"
import { useAuthStore } from "@/stores/auth"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { format, isValid, parse } from "date-fns"
import { CalendarIcon } from "lucide-react"
import { useEffect, useMemo, useState } from "react"
import { toast } from "sonner"
import { Calendar } from "../ui/calendar"
import { Field, FieldContent, FieldLabel } from "../ui/field"
import { Popover, PopoverContent, PopoverTrigger } from "../ui/popover"
import { RichTextEditor } from "./rich-text-editor"
import { MemberAvatar } from "./inline-editors"

const TASK_DATE_FORMAT = "yyyy-MM-dd"

function formatTaskDate(date: Date) {
  return format(date, TASK_DATE_FORMAT)
}

function parseTaskDate(value?: string) {
  if (!value) {
    return undefined
  }

  const parsed = parse(value.slice(0, 10), TASK_DATE_FORMAT, new Date())
  return isValid(parsed) ? parsed : undefined
}

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
  const authUser = useAuthStore(s => s.user)
  const [formData, setFormData] = useState<CreateTaskRequest>({
    title: "",
    description: "",
    status: TaskStatus.TODO,
    priority: CollabPriority.P2,
    parent_task_id: parentId,
    sprint_id: defaultSprintId || "",
    assignee_id: authUser?.id,
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
        assignee_id: authUser?.id,
      })
    }
  }, [open, parentId, defaultSprintId, authUser?.id])

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

  const { data: membersData } = useQuery({
    queryKey: ["project-members", projectId],
    queryFn: () => projectsApi.listMembers(projectId, { page: 1, page_size: 100 }),
    enabled: !!projectId,
  })
  const memberItems = useMemo(() => {
    const members = membersData?.items ?? []
    return [
      { label: "Unassigned", value: "" },
      ...members.map(m => ({ label: m.fullname || m.username, value: m.user_id })),
    ]
  }, [membersData?.items])

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

          <Field>
            <FieldLabel>Assignee</FieldLabel>
            <FieldContent>
              <Combobox
                value={formData.assignee_id || ""}
                onValueChange={(val) => setFormData({ ...formData, assignee_id: val || undefined })}
                itemToStringLabel={(item) => memberItems.find(opt => opt.value === item)?.label || item}
              >
                <ComboboxInput placeholder="Select assignee" />
                <ComboboxContent>
                  <ComboboxList>
                    {memberItems.map((opt) => (
                      <ComboboxItem key={opt.value} value={opt.value}>
                        {opt.value ? <MemberAvatar name={opt.label} /> : null}
                        {opt.label}
                      </ComboboxItem>
                    ))}
                  </ComboboxList>
                </ComboboxContent>
              </Combobox>
            </FieldContent>
          </Field>

          <div className="grid grid-cols-2 gap-4">
            <Field>
              <FieldLabel>Due Date</FieldLabel>
              <FieldContent>
                <Popover>
                  <PopoverTrigger render={<Button variant={"outline"} data-empty={!formData.due_date} className="w-full justify-between text-left font-normal data-[empty=true]:text-muted-foreground">{formData.due_date ? format(parseTaskDate(formData.due_date) ?? new Date(formData.due_date), "PPP") : <span>Pick a date</span>}<CalendarIcon data-icon="inline-end" /></Button>} />
                  <PopoverContent className="w-auto p-0" align="start">
                    <Calendar
                      mode="single"
                      selected={parseTaskDate(formData.due_date)}
                      onSelect={(date) => setFormData({ ...formData, due_date: date ? formatTaskDate(date) : undefined })}
                      defaultMonth={parseTaskDate(formData.due_date)}
                    />
                  </PopoverContent>
                </Popover>
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

  const { data: editMembersData } = useQuery({
    queryKey: ["project-members", projectId],
    queryFn: () => projectsApi.listMembers(projectId, { page: 1, page_size: 100 }),
    enabled: !!projectId,
  })
  const editMemberItems = useMemo(() => {
    const members = editMembersData?.items ?? []
    return [
      { label: "Unassigned", value: "" },
      ...members.map(m => ({ label: m.fullname || m.username, value: m.user_id })),
    ]
  }, [editMembersData?.items])

  if (!task) return null

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="sm:max-w-xl overflow-y-auto">
        <SheetHeader>
          <SheetTitle>Edit Task</SheetTitle>
          <SheetDescription>
            Update task details.
          </SheetDescription>
        </SheetHeader>

        <div className="grid flex-1 auto-rows-min gap-4 px-4">
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

          <Field>
            <FieldLabel>Assignee</FieldLabel>
            <FieldContent>
              <Combobox
                value={formData.assignee_id || ""}
                onValueChange={(val) => setFormData({ ...formData, assignee_id: val || undefined })}
                itemToStringLabel={(item) => editMemberItems.find(opt => opt.value === item)?.label || item}
              >
                <ComboboxInput placeholder="Select assignee" />
                <ComboboxContent>
                  <ComboboxList>
                    {editMemberItems.map((opt) => (
                      <ComboboxItem key={opt.value} value={opt.value}>
                        {opt.value ? <MemberAvatar name={opt.label} /> : null}
                        {opt.label}
                      </ComboboxItem>
                    ))}
                  </ComboboxList>
                </ComboboxContent>
              </Combobox>
            </FieldContent>
          </Field>

          <div className="grid grid-cols-2 gap-4">
            <Field>
              <FieldLabel>Due Date</FieldLabel>
              <FieldContent>
                <Popover>
                  <PopoverTrigger render={<Button variant={"outline"} data-empty={!formData.due_date} className="w-full justify-between text-left font-normal data-[empty=true]:text-muted-foreground">{formData.due_date ? format(parseTaskDate(formData.due_date) ?? new Date(formData.due_date), "PPP") : <span>Pick a date</span>}<CalendarIcon data-icon="inline-end" /></Button>} />
                  <PopoverContent className="w-auto p-0" align="start">
                    <Calendar
                      mode="single"
                      selected={parseTaskDate(formData.due_date)}
                      onSelect={(date) => setFormData({ ...formData, due_date: date ? formatTaskDate(date) : undefined })}
                      defaultMonth={parseTaskDate(formData.due_date)}
                    />
                  </PopoverContent>
                </Popover>
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
        </div>

        <SheetFooter>
          <div className="flex w-full items-center justify-end space-x-2">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={mutation.isPending} onClick={handleSubmit}>
              {mutation.isPending ? "Save Changes" : "Save"}
            </Button>
          </div>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
