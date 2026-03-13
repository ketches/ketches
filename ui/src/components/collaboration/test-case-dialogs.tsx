import {
  collaborationApi,
  TestRunStatus,
  TestRunStatusOptions,
  type CreateTestCaseRequest,
  type CreateTestRunRequest,
  type Sprint,
  type TestCase,
  type UpdateTestCaseRequest
} from "@/api/collaboration"
import { Button } from "@/components/ui/button"
import { Combobox, ComboboxContent, ComboboxInput, ComboboxItem, ComboboxList } from "@/components/ui/combobox"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Sheet, SheetContent, SheetDescription, SheetFooter, SheetHeader, SheetTitle } from "@/components/ui/sheet"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useEffect, useState } from "react"
import { toast } from "sonner"
import { Field, FieldContent, FieldLabel } from "../ui/field"
import { RichTextEditor } from "./rich-text-editor"
import { isRichTextEmpty } from "./rich-text-utils"

// ── Create Dialog ─────────────────────────────────────────────────────────────

interface CreateTestCaseDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectId: string
  onSuccess?: () => void
}

export function CreateTestCaseDialog({
  open,
  onOpenChange,
  projectId,
  onSuccess
}: CreateTestCaseDialogProps) {
  const queryClient = useQueryClient()

  const [title, setTitle] = useState("")
  const [precondition, setPrecondition] = useState("")
  const [steps, setSteps] = useState("")
  const [expectedResult, setExpectedResult] = useState("")
  const [sprintId, setSprintId] = useState<string>("")

  // Reset form on open
  useEffect(() => {
    if (open) {
      setTitle("")
      setPrecondition("")
      setSteps("")
      setExpectedResult("")
      setSprintId("")
    }
  }, [open])

  const { data: sprints } = useQuery({
    queryKey: ["sprints", projectId, "all"],
    queryFn: () => collaborationApi.listSprints(projectId, { page_size: 100 }),
    enabled: open,
  })

  const mutation = useMutation({
    mutationFn: () => {
      const data: CreateTestCaseRequest = {
        title,
        precondition,
        steps,
        expected_result: expectedResult,
        sprint_id: sprintId || undefined,
      }
      return collaborationApi.createTestCase(projectId, data)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["test-cases", projectId] })
      toast.success("Test case created")
      onOpenChange(false)
      onSuccess?.()
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: string } } }
      toast.error("Failed to create test case", {
        description: err.response?.data?.error || "Unknown error occurred"
      })
    }
  })

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="sm:max-w-xl overflow-y-auto">
        <form onSubmit={(e) => { e.preventDefault(); mutation.mutate() }} className="space-y-4">
          <SheetHeader>
            <SheetTitle>Create Test Case</SheetTitle>
            <SheetDescription>
              Add a new test case to the project.
            </SheetDescription>
          </SheetHeader>

          <Field>
            <FieldLabel>Title</FieldLabel>
            <FieldContent>
              <Input
                id="title"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="e.g. Verify Login Functionality"
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
                      <ComboboxItem key={s.id} value={s.id}>
                        {s.name}
                      </ComboboxItem>
                    ))}
                  </ComboboxList>
                </ComboboxContent>
              </Combobox>
            </FieldContent>
          </Field>

          <Field>
            <FieldLabel>Precondition</FieldLabel>
            <FieldContent>
              <RichTextEditor
                value={precondition}
                onChange={setPrecondition}
                placeholder="e.g. User is logged out"
              />
            </FieldContent>
          </Field>

          <Field>
            <FieldLabel>Steps</FieldLabel>
            <FieldContent>
              <RichTextEditor
                value={steps}
                onChange={setSteps}
                placeholder="1. Navigate to login page..."
              />
            </FieldContent>
          </Field>

          <Field>
            <FieldLabel>Expected Result</FieldLabel>
            <FieldContent>
              <RichTextEditor
                value={expectedResult}
                onChange={setExpectedResult}
                placeholder="User is redirected to dashboard"
              />
            </FieldContent>
          </Field>

          <SheetFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
            <Button
              type="submit"
              disabled={mutation.isPending || !title || isRichTextEmpty(steps) || isRichTextEmpty(expectedResult)}
            >
              {mutation.isPending ? "Creating..." : "Create"}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  )
}

// ── Edit Dialog ──────────────────────────────────────────────────────────────

interface EditTestCaseDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectId: string
  testCase: TestCase | null
  onSuccess?: () => void
}

export function EditTestCaseDialog({
  open,
  onOpenChange,
  projectId,
  testCase,
  onSuccess
}: EditTestCaseDialogProps) {
  const queryClient = useQueryClient()

  const [title, setTitle] = useState("")
  const [precondition, setPrecondition] = useState("")
  const [steps, setSteps] = useState("")
  const [expectedResult, setExpectedResult] = useState("")
  const [sprintId, setSprintId] = useState<string>("")

  useEffect(() => {
    if (open && testCase) {
      setTitle(testCase.title)
      setPrecondition(testCase.precondition || "")
      setSteps(testCase.steps)
      setExpectedResult(testCase.expected_result)
      setSprintId(testCase.sprint_id || "")
    }
  }, [open, testCase])

  const { data: sprints } = useQuery({
    queryKey: ["sprints", projectId, "all"],
    queryFn: () => collaborationApi.listSprints(projectId, { page_size: 100 }),
    enabled: open,
  })

  const mutation = useMutation({
    mutationFn: () => {
      if (!testCase) throw new Error("No test case selected")

      const data: UpdateTestCaseRequest = {
        title,
        precondition,
        steps,
        expected_result: expectedResult,
        sprint_id: sprintId || undefined,
        requirement_id: testCase.requirement_id,
        task_id: testCase.task_id
      }
      return collaborationApi.updateTestCase(projectId, testCase.id, data)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["test-cases", projectId] })
      toast.success("Test case updated")
      onOpenChange(false)
      onSuccess?.()
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: string } } }
      toast.error("Failed to update test case", {
        description: err.response?.data?.error || "Unknown error occurred"
      })
    }
  })

  if (!testCase) return null

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="sm:max-w-xl overflow-y-auto">
        <form onSubmit={(e) => { e.preventDefault(); mutation.mutate() }} className="space-y-4">
          <SheetHeader>
            <SheetTitle>Edit Test Case</SheetTitle>
          </SheetHeader>

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

          <Field>
            <FieldLabel>Precondition</FieldLabel>
            <FieldContent>
              <RichTextEditor
                value={precondition}
                onChange={setPrecondition}
              />
            </FieldContent>
          </Field>

          <Field>
            <FieldLabel>Steps</FieldLabel>
            <FieldContent>
              <RichTextEditor
                value={steps}
                onChange={setSteps}
              />
            </FieldContent>
          </Field>

          <Field>
            <FieldLabel>Expected Result</FieldLabel>
            <FieldContent>
              <RichTextEditor
                value={expectedResult}
                onChange={setExpectedResult}
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
                      <ComboboxItem key={s.id} value={s.id}>
                        {s.name}
                      </ComboboxItem>
                    ))}
                  </ComboboxList>
                </ComboboxContent>
              </Combobox>
            </FieldContent>
          </Field>

          <SheetFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
            <Button
              type="submit"
              disabled={mutation.isPending || !title || isRichTextEmpty(steps) || isRichTextEmpty(expectedResult)}
            >
              {mutation.isPending ? "Save Changes" : "Save Changes"}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  )
}

// ── Delete Dialog ────────────────────────────────────────────────────────────

interface DeleteTestCaseDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectId: string
  testCaseId: string
  onSuccess?: () => void
}

export function DeleteTestCaseDialog({
  open,
  onOpenChange,
  projectId,
  testCaseId,
  onSuccess
}: DeleteTestCaseDialogProps) {
  const queryClient = useQueryClient()

  const mutation = useMutation({
    mutationFn: () => collaborationApi.deleteTestCase(projectId, testCaseId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["test-cases", projectId] })
      toast.success("Test case deleted")
      onOpenChange(false)
      onSuccess?.()
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: string } } }
      toast.error("Failed to delete test case", {
        description: err.response?.data?.error || "Unknown error occurred"
      })
    }
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete Test Case</DialogTitle>
          <DialogDescription>
            Are you sure you want to delete this test case? This action cannot be undone.
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

// ── Run Test Dialog ────────────────────────────────────────────────────────────

interface CreateTestRunDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectId: string
  testCase: TestCase | null
  onSuccess?: () => void
}

export function CreateTestRunDialog({
  open,
  onOpenChange,
  projectId,
  testCase,
  onSuccess
}: CreateTestRunDialogProps) {
  // queryClient not used

  const [status, setStatus] = useState<TestRunStatus>(TestRunStatus.PASSED)
  const [comment, setComment] = useState("")

  useEffect(() => {
    if (open) {
      setStatus(TestRunStatus.PASSED)
      setComment("")
    }
  }, [open])

  const mutation = useMutation({
    mutationFn: () => {
      if (!testCase) throw new Error("No test case selected")

      const data: CreateTestRunRequest = {
        status,
        comment
      }
      return collaborationApi.createTestRun(projectId, testCase.id, data)
    },
    onSuccess: () => {
      // Maybe we want to invalidate test runs list if we had one, but here we just notify
      toast.success("Test run recorded")
      onOpenChange(false)
      onSuccess?.()
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: string } } }
      toast.error("Failed to record test run", {
        description: err.response?.data?.error || "Unknown error occurred"
      })
    }
  })

  if (!testCase) return null

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="sm:max-w-xl overflow-y-auto">
        <form onSubmit={(e) => { e.preventDefault(); mutation.mutate() }} className="space-y-4">
          <SheetHeader>
            <SheetTitle>Record Test Run</SheetTitle>
            <SheetDescription>
              Record the execution result for "{testCase.title}".
            </SheetDescription>
          </SheetHeader>

          <Field>
            <FieldLabel>Status</FieldLabel>
            <FieldContent>
              <Combobox value={status} onValueChange={(v) => v && setStatus(v as TestRunStatus)} itemToStringLabel={(item) => TestRunStatusOptions.find(opt => opt.value === item)?.label || item}>
                <ComboboxInput />
                <ComboboxContent>
                  <ComboboxList>
                    {TestRunStatusOptions.map((s) => (
                      <ComboboxItem key={s.value} value={s.value}>{s.label}</ComboboxItem>
                    ))}
                  </ComboboxList>
                </ComboboxContent>
              </Combobox>
            </FieldContent>
          </Field>

          <Field>
            <FieldLabel>Comment (Optional)</FieldLabel>
            <FieldContent>
              <Textarea
                id="run-comment"
                value={comment}
                onChange={(e) => setComment(e.target.value)}
                placeholder="Any observations or details..."
                rows={3}
              />
            </FieldContent>
          </Field>

          <SheetFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
            <Button type="submit" disabled={mutation.isPending}>
              {mutation.isPending ? "Recording..." : "Record Result"}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  )
}
