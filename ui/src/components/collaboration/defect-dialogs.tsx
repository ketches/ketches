import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Label } from "@/components/ui/label"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import {
  collaborationApi,
  DefectStatus,
  DefectSeverity,
  type Defect,
  type CreateDefectRequest,
  type UpdateDefectRequest,
  type Requirement,
  type Task,
  type TestCase,
} from "@/api/collaboration"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useState, useEffect } from "react"
import { toast } from "sonner"
import { Combobox, ComboboxContent, ComboboxInput, ComboboxItem, ComboboxList } from "@/components/ui/combobox"

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
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px] max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Report Defect</DialogTitle>
          <DialogDescription>
            Report a new defect found in the project.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-4">
          <div className="grid gap-2">
            <Label htmlFor="title">Title</Label>
            <Input
              id="title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="e.g. Login fails with 500 error"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="grid gap-2">
              <Label>Severity</Label>
              <Combobox value={severity} onValueChange={(v) => v && setSeverity(v as DefectSeverity)}>
                <ComboboxInput placeholder="Select severity" />
                <ComboboxContent>
                  <ComboboxList>
                    {Object.values(DefectSeverity).map((s) => (
                      <ComboboxItem key={s} value={s}>{s.toUpperCase()}</ComboboxItem>
                    ))}
                  </ComboboxList>
                </ComboboxContent>
              </Combobox>
            </div>
            <div className="grid gap-2">
              <Label>Status</Label>
              <Combobox value={status} onValueChange={(v) => v && setStatus(v as DefectStatus)}>
                <ComboboxInput placeholder="Select status" />
                <ComboboxContent>
                  <ComboboxList>
                    {Object.values(DefectStatus).map((s) => (
                      <ComboboxItem key={s} value={s}>{s.toUpperCase().replace('_', ' ')}</ComboboxItem>
                    ))}
                  </ComboboxList>
                </ComboboxContent>
              </Combobox>
            </div>
          </div>

          <div className="grid gap-2">
            <Label htmlFor="description">Description</Label>
            <Textarea
              id="description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Detailed description of the issue..."
              rows={3}
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="reproduction-steps">Reproduction Steps</Label>
            <Textarea
              id="reproduction-steps"
              value={reproductionSteps}
              onChange={(e) => setReproductionSteps(e.target.value)}
              placeholder="1. Go to page X..."
              rows={4}
            />
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="grid gap-2">
              <Label>Requirement (Optional)</Label>
              <Combobox value={requirementId} onValueChange={(v) => setRequirementId(v || "")}>
                <ComboboxInput placeholder="Select requirement" />
                <ComboboxContent>
                  <ComboboxList>
                    {requirements?.items.map((req: Requirement) => (
                      <ComboboxItem key={req.id} value={req.id}>{req.title}</ComboboxItem>
                    ))}
                  </ComboboxList>
                </ComboboxContent>
              </Combobox>
            </div>
            <div className="grid gap-2">
              <Label>Task (Optional)</Label>
              <Combobox value={taskId} onValueChange={(v) => setTaskId(v || "")}>
                <ComboboxInput placeholder="Select task" />
                <ComboboxContent>
                  <ComboboxList>
                    {tasks?.items.map((t: Task) => (
                      <ComboboxItem key={t.id} value={t.id}>{t.title}</ComboboxItem>
                    ))}
                  </ComboboxList>
                </ComboboxContent>
              </Combobox>
            </div>
            <div className="grid gap-2">
              <Label>Test Case (Optional)</Label>
              <Combobox value={testCaseId} onValueChange={(v) => setTestCaseId(v || "")}>
                <ComboboxInput placeholder="Select test case" />
                <ComboboxContent>
                  <ComboboxList>
                    {testCases?.items.map((tc: TestCase) => (
                      <ComboboxItem key={tc.id} value={tc.id}>{tc.title}</ComboboxItem>
                    ))}
                  </ComboboxList>
                </ComboboxContent>
              </Combobox>
            </div>
          </div>
          <p className="text-xs text-muted-foreground">
            * At least one upstream link (Requirement, Task, or Test Case) is required.
          </p>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button 
            onClick={() => mutation.mutate()} 
            disabled={mutation.isPending || !title || !description || (!requirementId && !taskId && !testCaseId)}
          >
            {mutation.isPending ? "Creating..." : "Create Defect"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
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
    }
  }, [open, defect])

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
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px] max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Edit Defect</DialogTitle>
        </DialogHeader>

        <div className="grid gap-4 py-4">
          <div className="grid gap-2">
            <Label htmlFor="edit-title">Title</Label>
            <Input
              id="edit-title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="grid gap-2">
              <Label>Severity</Label>
              <Combobox value={severity} onValueChange={(v) => v && setSeverity(v as DefectSeverity)}>
                <ComboboxInput />
                <ComboboxContent>
                  <ComboboxList>
                    {Object.values(DefectSeverity).map((s) => (
                      <ComboboxItem key={s} value={s}>{s.toUpperCase()}</ComboboxItem>
                    ))}
                  </ComboboxList>
                </ComboboxContent>
              </Combobox>
            </div>
            <div className="grid gap-2">
              <Label>Status</Label>
              <Combobox value={status} onValueChange={(v) => v && setStatus(v as DefectStatus)}>
                <ComboboxInput />
                <ComboboxContent>
                  <ComboboxList>
                    {Object.values(DefectStatus).map((s) => (
                      <ComboboxItem key={s} value={s}>{s.toUpperCase().replace('_', ' ')}</ComboboxItem>
                    ))}
                  </ComboboxList>
                </ComboboxContent>
              </Combobox>
            </div>
          </div>

          <div className="grid gap-2">
            <Label htmlFor="edit-description">Description</Label>
            <Textarea
              id="edit-description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={3}
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="edit-reproduction-steps">Reproduction Steps</Label>
            <Textarea
              id="edit-reproduction-steps"
              value={reproductionSteps}
              onChange={(e) => setReproductionSteps(e.target.value)}
              rows={4}
            />
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="grid gap-2">
              <Label>Requirement</Label>
              <Combobox value={requirementId} onValueChange={(v) => setRequirementId(v || "")}>
                <ComboboxInput />
                <ComboboxContent>
                  <ComboboxList>
                    {requirements?.items.map((req: Requirement) => (
                      <ComboboxItem key={req.id} value={req.id}>{req.title}</ComboboxItem>
                    ))}
                  </ComboboxList>
                </ComboboxContent>
              </Combobox>
            </div>
            <div className="grid gap-2">
              <Label>Task</Label>
              <Combobox value={taskId} onValueChange={(v) => setTaskId(v || "")}>
                <ComboboxInput />
                <ComboboxContent>
                  <ComboboxList>
                    {tasks?.items.map((t: Task) => (
                      <ComboboxItem key={t.id} value={t.id}>{t.title}</ComboboxItem>
                    ))}
                  </ComboboxList>
                </ComboboxContent>
              </Combobox>
            </div>
            <div className="grid gap-2">
              <Label>Test Case</Label>
              <Combobox value={testCaseId} onValueChange={(v) => setTestCaseId(v || "")}>
                <ComboboxInput />
                <ComboboxContent>
                  <ComboboxList>
                    {testCases?.items.map((tc: TestCase) => (
                      <ComboboxItem key={tc.id} value={tc.id}>{tc.title}</ComboboxItem>
                    ))}
                  </ComboboxList>
                </ComboboxContent>
              </Combobox>
            </div>
          </div>

          <div className="grid gap-2">
            <Label htmlFor="edit-fix-note">Fix Note (Optional)</Label>
            <Textarea
              id="edit-fix-note"
              value={fixNote}
              onChange={(e) => setFixNote(e.target.value)}
              placeholder="Details about the fix or resolution..."
              rows={2}
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button 
            onClick={() => mutation.mutate()} 
            disabled={mutation.isPending || !title || !description || (!requirementId && !taskId && !testCaseId && !defect?.test_run_id)}
          >
            {mutation.isPending ? "Save Changes" : "Save Changes"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
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
      <DialogContent className="sm:max-w-[400px]">
        <DialogHeader>
          <DialogTitle>Transition Defect</DialogTitle>
          <DialogDescription>
            Change the status of "{defect.title}".
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-4">
          <div className="grid gap-2">
            <Label>New Status</Label>
            <Combobox value={status} onValueChange={(v) => v && setStatus(v as DefectStatus)}>
              <ComboboxInput />
              <ComboboxContent>
                <ComboboxList>
                  {Object.values(DefectStatus).map((s) => (
                    <ComboboxItem key={s} value={s}>{s.toUpperCase().replace('_', ' ')}</ComboboxItem>
                  ))}
                </ComboboxList>
              </ComboboxContent>
            </Combobox>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button onClick={() => mutation.mutate()} disabled={mutation.isPending}>
            {mutation.isPending ? "Updating..." : "Update Status"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
