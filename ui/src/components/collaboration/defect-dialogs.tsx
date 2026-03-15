import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Sheet, SheetContent, SheetDescription, SheetFooter, SheetHeader, SheetTitle } from "@/components/ui/sheet"

import {
  collaborationApi,
  DefectSeverity,
  DefectSeverityOptions,
  DefectStatus,
  DefectStatusOptions,
  type CreateDefectRequest,
  type Defect,
  type Requirement,
  type Sprint,
  type Task,
  type TestCase,
  type UpdateDefectRequest,
} from "@/api/collaboration"
import { BasicEditor } from "@/components/editor/basic-editor"
import { isBasicEditorEmpty } from "@/components/editor/basic-editor-value"
import { Combobox, ComboboxContent, ComboboxInput, ComboboxItem, ComboboxList } from "@/components/ui/combobox"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useEffect, useState } from "react"
import { toast } from "sonner"
import { Field, FieldContent, FieldLabel } from "../ui/field"

// ── Create Dialog ─────────────────────────────────────────────────────────────

interface CreateDefectDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectId: string
  onSuccess?: () => void
}

export function CreateDefectDialog({
  open,
  onOpenChange,
  projectId,
  onSuccess
}: CreateDefectDialogProps) {
  const queryClient = useQueryClient()

  const [title, setTitle] = useState("")
  const [description, setDescription] = useState("")
  const [reproductionSteps, setReproductionSteps] = useState("")
  const [severity, setSeverity] = useState<DefectSeverity>(DefectSeverity.MEDIUM)
  const [status, setStatus] = useState<DefectStatus>(DefectStatus.NEW)
  const [requirementId, setRequirementId] = useState<string>("")
  const [taskId, setTaskId] = useState<string>("")
  const [testCaseId, setTestCaseId] = useState<string>("")
  const [sprintId, setSprintId] = useState<string>("")

  // Fetch options
  const { data: requirements } = useQuery({
    queryKey: ["requirements", projectId],
    queryFn: () => collaborationApi.listRequirements(projectId, { page_size: 100 }),
    enabled: open,
  })

  const { data: tasks } = useQuery({
    queryKey: ["tasks", projectId],
    queryFn: () => collaborationApi.listTasks(projectId, { page_size: 100 }),
    enabled: open,
  })

  const { data: testCases } = useQuery({
    queryKey: ["test-cases", projectId],
    queryFn: () => collaborationApi.listTestCases(projectId, { page_size: 100 }),
    enabled: open,
  })

  const { data: sprints } = useQuery({
    queryKey: ["sprints", projectId, "all"],
    queryFn: () => collaborationApi.listSprints(projectId, { page_size: 100 }),
    enabled: open,
  })


  // Reset form on open
  useEffect(() => {
    if (open) {
      setTitle("")
      setDescription("")
      setReproductionSteps("")
      setSeverity(DefectSeverity.MEDIUM)
      setStatus(DefectStatus.NEW)
      setRequirementId("")
      setTaskId("")
      setTestCaseId("")
      setSprintId("")
    }
  }, [open])

  const mutation = useMutation({
    mutationFn: () => {
      const data: CreateDefectRequest = {
        title,
        description,
        reproduction_steps: reproductionSteps,
        severity,
        status,
        sprint_id: sprintId || undefined,
        requirement_id: requirementId || undefined,
        task_id: taskId || undefined,
        test_case_id: testCaseId || undefined,
      }
      return collaborationApi.createDefect(projectId, data)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["defects", projectId] })
      toast.success("Defect created")
      onOpenChange(false)
      onSuccess?.()
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: string } } }
      toast.error("Failed to create defect", {
        description: err.response?.data?.error || "Unknown error occurred"
      })
    }
  })

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="sm:max-w-xl overflow-y-auto">
        <SheetHeader>
          <SheetTitle>Report Defect</SheetTitle>
          <SheetDescription>
            Report a new defect found in the project.
          </SheetDescription>
        </SheetHeader>
        <div className="grid flex-1 auto-rows-min gap-4 px-4">
          <Field>
            <FieldLabel>Title</FieldLabel>
            <FieldContent>
              <Input
                id="title"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="e.g. Login fails with 500 error"
              />
            </FieldContent>
          </Field>

          <div className="grid grid-cols-2 gap-4">
            <Field>
              <FieldLabel>Severity</FieldLabel>
              <FieldContent>
                <Combobox
                  value={severity}
                  onValueChange={(v) => v && setSeverity(v as DefectSeverity)}
                  itemToStringLabel={(item) => DefectSeverityOptions.find(opt => opt.value === item)?.label || item}
                >
                  <ComboboxInput placeholder="Select severity" />
                  <ComboboxContent>
                    <ComboboxList>
                      {DefectSeverityOptions.map((s) => (
                        <ComboboxItem key={s.value} value={s.value}>{s.label}</ComboboxItem>
                      ))}
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel>Status</FieldLabel>
              <FieldContent>
                <Combobox value={status} onValueChange={(v) => v && setStatus(v as DefectStatus)} itemToStringLabel={(item) => DefectStatusOptions.find(opt => opt.value === item)?.label || item}>
                  <ComboboxInput placeholder="Select status" />
                  <ComboboxContent>
                    <ComboboxList>
                      {DefectStatusOptions.map((s) => (
                        <ComboboxItem key={s.value} value={s.value}>{s.label}</ComboboxItem>
                      ))}
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
              </FieldContent>
            </Field>
          </div>

          <Field>
            <FieldLabel>Description</FieldLabel>
            <FieldContent>
              <BasicEditor
                value={description}
                onChange={setDescription}
                placeholder="Detailed description of the issue..."
              />
            </FieldContent>
          </Field>

          <Field>
            <FieldLabel>Reproduction Steps</FieldLabel>
            <FieldContent>
              <BasicEditor
                value={reproductionSteps}
                onChange={setReproductionSteps}
                placeholder="1. Go to page X..."
              />
            </FieldContent>
          </Field>

          <Field>
            <FieldLabel>Sprint (Optional)</FieldLabel>
            <FieldContent>
              <Combobox value={sprintId} onValueChange={(v) => setSprintId(v || "")} itemToStringLabel={(item) => sprints?.items.find(s => s.id === item)?.name || item}>
                <ComboboxInput placeholder="Select sprint" />
                <ComboboxContent>
                  <ComboboxList>
                    {sprints?.items.map((s: Sprint) => (
                      <ComboboxItem key={s.id} value={s.id}>{s.name}</ComboboxItem>
                    ))}
                  </ComboboxList>
                </ComboboxContent>
              </Combobox>
            </FieldContent>
          </Field>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <Field>
              <FieldLabel>Requirement (Optional)</FieldLabel>
              <FieldContent>
                <Combobox value={requirementId} onValueChange={(v) => setRequirementId(v || "")} itemToStringLabel={(item) => requirements?.items.find(req => req.id === item)?.title || item}>
                  <ComboboxInput placeholder="Select requirement" />
                  <ComboboxContent>
                    <ComboboxList>
                      {requirements?.items.map((req: Requirement) => (
                        <ComboboxItem key={req.id} value={req.id}>{req.title}</ComboboxItem>
                      ))}
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel>Task (Optional)</FieldLabel>
              <FieldContent>
                <Combobox value={taskId} onValueChange={(v) => setTaskId(v || "")} itemToStringLabel={(item) => tasks?.items.find(t => t.id === item)?.title || item}>
                  <ComboboxInput placeholder="Select task" />
                  <ComboboxContent>
                    <ComboboxList>
                      {tasks?.items.map((t: Task) => (
                        <ComboboxItem key={t.id} value={t.id}>{t.title}</ComboboxItem>
                      ))}
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel>Test Case (Optional)</FieldLabel>
              <FieldContent>
                <Combobox value={testCaseId} onValueChange={(v) => setTestCaseId(v || "")} itemToStringLabel={(item) => testCases?.items.find(tc => tc.id === item)?.title || item}>
                  <ComboboxInput placeholder="Select test case" />
                  <ComboboxContent>
                    <ComboboxList>
                      {testCases?.items.map((tc: TestCase) => (
                        <ComboboxItem key={tc.id} value={tc.id}>{tc.title}</ComboboxItem>
                      ))}
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
              </FieldContent>
            </Field>
          </div>
          <p className="text-xs text-muted-foreground">
            * At least one upstream link (Requirement, Task, or Test Case) is required.
          </p>
        </div>

        <SheetFooter>
          <div className="flex w-full items-center justify-end space-x-2">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
            <Button
              type="submit"
              disabled={mutation.isPending || !title || isBasicEditorEmpty(description) || (!requirementId && !taskId && !testCaseId)}
              onClick={() => mutation.mutate()}
            >
              {mutation.isPending ? "Creating..." : "Create Defect"}
            </Button>
          </div>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}

// ── Edit Dialog ──────────────────────────────────────────────────────────────

interface EditDefectDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectId: string
  defect: Defect | null
  onSuccess?: () => void
}

export function EditDefectDialog({
  open,
  onOpenChange,
  projectId,
  defect,
  onSuccess
}: EditDefectDialogProps) {
  const queryClient = useQueryClient()

  const [title, setTitle] = useState("")
  const [description, setDescription] = useState("")
  const [reproductionSteps, setReproductionSteps] = useState("")
  const [severity, setSeverity] = useState<DefectSeverity>(DefectSeverity.MEDIUM)
  const [status, setStatus] = useState<DefectStatus>(DefectStatus.NEW)
  const [fixNote, setFixNote] = useState("")
  const [requirementId, setRequirementId] = useState<string>("")
  const [taskId, setTaskId] = useState<string>("")
  const [testCaseId, setTestCaseId] = useState<string>("")
  const [sprintId, setSprintId] = useState<string>("")

  // Fetch options
  const { data: requirements } = useQuery({
    queryKey: ["requirements", projectId],
    queryFn: () => collaborationApi.listRequirements(projectId, { page_size: 100 }),
    enabled: open,
  })

  const { data: tasks } = useQuery({
    queryKey: ["tasks", projectId],
    queryFn: () => collaborationApi.listTasks(projectId, { page_size: 100 }),
    enabled: open,
  })

  const { data: testCases } = useQuery({
    queryKey: ["test-cases", projectId],
    queryFn: () => collaborationApi.listTestCases(projectId, { page_size: 100 }),
    enabled: open,
  })

  const { data: sprints } = useQuery({
    queryKey: ["sprints", projectId, "all"],
    queryFn: () => collaborationApi.listSprints(projectId, { page_size: 100 }),
    enabled: open,
  })


  useEffect(() => {
    if (open && defect) {
      setTitle(defect.title)
      setDescription(defect.description)
      setReproductionSteps(defect.reproduction_steps || "")
      setSeverity(defect.severity)
      setStatus(defect.status)
      setFixNote(defect.fix_note || "")
      setRequirementId(defect.requirement_id || "")
      setTaskId(defect.task_id || "")
      setTestCaseId(defect.test_case_id || "")
      setSprintId(defect.sprint_id || "")
    }
  }, [open, defect])

  const requirementLabel = requirements?.items.find((req) => req.id === requirementId)?.title || requirementId
  const taskLabel = tasks?.items.find((task) => task.id === taskId)?.title || taskId
  const testCaseLabel = testCases?.items.find((tc) => tc.id === testCaseId)?.title || testCaseId

  const mutation = useMutation({
    mutationFn: () => {
      if (!defect) throw new Error("No defect selected")

      const data: UpdateDefectRequest = {
        title,
        description,
        reproduction_steps: reproductionSteps,
        severity,
        status,
        fix_note: fixNote,
        sprint_id: sprintId || undefined,
        requirement_id: requirementId || undefined,
        task_id: taskId || undefined,
        test_case_id: testCaseId || undefined,
        test_run_id: defect.test_run_id, // Preserve if it exists, not editable
      }
      return collaborationApi.updateDefect(projectId, defect.id, data)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["defects", projectId] })
      toast.success("Defect updated")
      onOpenChange(false)
      onSuccess?.()
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: string } } }
      toast.error("Failed to update defect", {
        description: err.response?.data?.error || "Unknown error occurred"
      })
    }
  })

  if (!defect) return null

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="sm:max-w-xl overflow-y-auto">
        <SheetHeader>
          <SheetTitle>Edit Defect</SheetTitle>
        </SheetHeader>
        <div className="grid flex-1 auto-rows-min gap-4 px-4">
          <Field>
            <FieldLabel>Title</FieldLabel>
            <FieldContent>
              <Input
                id="edit-title"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
              />
            </FieldContent>
          </Field>

          <div className="grid grid-cols-2 gap-4">
            <Field>
              <FieldLabel>Severity</FieldLabel>
              <FieldContent>
                <Combobox value={severity} onValueChange={(v) => v && setSeverity(v as DefectSeverity)} itemToStringLabel={(item) => DefectSeverityOptions.find(opt => opt.value === item)?.label || item}>
                  <ComboboxInput />
                  <ComboboxContent>
                    <ComboboxList>
                      {DefectSeverityOptions.map((s) => (
                        <ComboboxItem key={s.value} value={s.value}>{s.label}</ComboboxItem>
                      ))}
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel>Status</FieldLabel>
              <FieldContent>
                <Combobox value={status} onValueChange={(v) => v && setStatus(v as DefectStatus)} itemToStringLabel={(item) => DefectStatusOptions.find(opt => opt.value === item)?.label || item}>
                  <ComboboxInput />
                  <ComboboxContent>
                    <ComboboxList>
                      {DefectStatusOptions.map((s) => (
                        <ComboboxItem key={s.value} value={s.value}>{s.label}</ComboboxItem>
                      ))}
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
              </FieldContent>
            </Field>
          </div>

          <Field>
            <FieldLabel>Description</FieldLabel>
            <FieldContent>
              <BasicEditor
                value={description}
                onChange={setDescription}
              />
            </FieldContent>
          </Field>

          <Field>
            <FieldLabel>Reproduction Steps</FieldLabel>
            <FieldContent>
              <BasicEditor
                value={reproductionSteps}
                onChange={setReproductionSteps}
              />
            </FieldContent>
          </Field>

          <Field>
            <FieldLabel>Sprint (Optional)</FieldLabel>
            <FieldContent>
              <Combobox value={sprintId} onValueChange={(v) => setSprintId(v || "")} itemToStringLabel={(item) => sprints?.items.find(s => s.id === item)?.name || item}>
                <ComboboxInput placeholder="Select sprint" />
                <ComboboxContent>
                  <ComboboxList>
                    {sprints?.items.map((s: Sprint) => (
                      <ComboboxItem key={s.id} value={s.id}>{s.name}</ComboboxItem>
                    ))}
                  </ComboboxList>
                </ComboboxContent>
              </Combobox>
            </FieldContent>
          </Field>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <Field>
              <FieldLabel>Requirement</FieldLabel>
              <FieldContent>
                <Combobox
                  key={`defect-edit-requirement-${requirementLabel}`}
                  value={requirementId}
                  onValueChange={(v) => setRequirementId(v || "")}
                  itemToStringLabel={(item) => requirements?.items.find(req => req.id === item)?.title || item}
                >
                  <ComboboxInput />
                  <ComboboxContent>
                    <ComboboxList>
                      {requirements?.items.map((req: Requirement) => (
                        <ComboboxItem key={req.id} value={req.id}>{req.title}</ComboboxItem>
                      ))}
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel>Task</FieldLabel>
              <FieldContent>
                <Combobox
                  key={`defect-edit-task-${taskLabel}`}
                  value={taskId}
                  onValueChange={(v) => setTaskId(v || "")}
                  itemToStringLabel={(item) => tasks?.items.find(t => t.id === item)?.title || item}
                >
                  <ComboboxInput />
                  <ComboboxContent>
                    <ComboboxList>
                      {tasks?.items.map((t: Task) => (
                        <ComboboxItem key={t.id} value={t.id}>{t.title}</ComboboxItem>
                      ))}
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel>Test Case</FieldLabel>
              <FieldContent>
                <Combobox
                  key={`defect-edit-test-case-${testCaseLabel}`}
                  value={testCaseId}
                  onValueChange={(v) => setTestCaseId(v || "")}
                  itemToStringLabel={(item) => testCases?.items.find(tc => tc.id === item)?.title || item}
                >
                  <ComboboxInput />
                  <ComboboxContent>
                    <ComboboxList>
                      {testCases?.items.map((tc: TestCase) => (
                        <ComboboxItem key={tc.id} value={tc.id}>{tc.title}</ComboboxItem>
                      ))}
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
              </FieldContent>
            </Field>
          </div>

          <Field>
            <FieldLabel>Fix Note (Optional)</FieldLabel>
            <FieldContent>
              <Textarea
                id="edit-fix-note"
                value={fixNote}
                onChange={(e) => setFixNote(e.target.value)}
                placeholder="Details about the fix or resolution..."
                rows={2}
              />
            </FieldContent>
          </Field>
        </div>

        <SheetFooter>
          <div className="flex w-full items-center justify-end space-x-2">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
            <Button
              type="submit"
              disabled={mutation.isPending || !title || isBasicEditorEmpty(description) || (!requirementId && !taskId && !testCaseId && !defect?.test_run_id)}
              onClick={() => mutation.mutate()}
            >
              {mutation.isPending ? "Save Changes" : "Save Changes"}
            </Button>
          </div>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}

// ── Delete Dialog ────────────────────────────────────────────────────────────

interface DeleteDefectDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectId: string
  defectId: string
  onSuccess?: () => void
}

export function DeleteDefectDialog({
  open,
  onOpenChange,
  projectId,
  defectId,
  onSuccess
}: DeleteDefectDialogProps) {
  const queryClient = useQueryClient()

  const mutation = useMutation({
    mutationFn: () => collaborationApi.deleteDefect(projectId, defectId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["defects", projectId] })
      toast.success("Defect deleted")
      onOpenChange(false)
      onSuccess?.()
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: string } } }
      toast.error("Failed to delete defect", {
        description: err.response?.data?.error || "Unknown error occurred"
      })
    }
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete Defect</DialogTitle>
          <DialogDescription>
            Are you sure you want to delete this defect? This action cannot be undone.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button variant="destructive" onClick={() => mutation.mutate()} disabled={mutation.isPending}>
            {mutation.isPending ? "Deleting..." : "Delete"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ── Transition Dialog ────────────────────────────────────────────────────────

interface TransitionDefectDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectId: string
  defect: Defect | null
  onSuccess?: () => void
}

export function TransitionDefectDialog({
  open,
  onOpenChange,
  projectId,
  defect,
  onSuccess
}: TransitionDefectDialogProps) {
  const queryClient = useQueryClient()

  const [status, setStatus] = useState<DefectStatus>(DefectStatus.NEW)

  useEffect(() => {
    if (open && defect) {
      setStatus(defect.status)
    }
  }, [open, defect])

  // We use transitionDefect API specifically
  const mutation = useMutation({
    mutationFn: () => {
      if (!defect) throw new Error("No defect selected")
      return collaborationApi.transitionDefect(projectId, defect.id, status)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["defects", projectId] })
      toast.success("Defect status updated")
      onOpenChange(false)
      onSuccess?.()
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: string } } }
      toast.error("Failed to transition defect", {
        description: err.response?.data?.error || "Unknown error occurred"
      })
    }
  })

  if (!defect) return null

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-160">
        <form onSubmit={(e) => { e.preventDefault(); mutation.mutate() }} className="space-y-4">
          <DialogHeader>
            <DialogTitle>Transition Defect</DialogTitle>
            <DialogDescription>
              Change the status of "{defect.title}".
            </DialogDescription>
          </DialogHeader>

          <Field>
            <FieldLabel>New Status</FieldLabel>
            <FieldContent>
              <Combobox value={status} onValueChange={(v) => v && setStatus(v as DefectStatus)} itemToStringLabel={(item) => DefectStatusOptions.find(opt => opt.value === item)?.label || item}>
                <ComboboxInput />
                <ComboboxContent>
                  <ComboboxList>
                    {DefectStatusOptions.map((s) => (
                      <ComboboxItem key={s.value} value={s.value}>{s.label}</ComboboxItem>
                    ))}
                  </ComboboxList>
                </ComboboxContent>
              </Combobox>
            </FieldContent>
          </Field>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
            <Button type="submit" disabled={mutation.isPending}>
              {mutation.isPending ? "Updating..." : "Update Status"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
