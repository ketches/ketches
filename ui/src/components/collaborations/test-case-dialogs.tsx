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
import { Field, FieldContent, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { getErrorMessage } from "@/lib/utils"
import { Textarea } from "@/components/ui/textarea"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useEffect, useState } from "react"
import { toast } from "sonner"

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
      toast.error("Failed to create test case", {
        description: getErrorMessage(error, "Unknown error occurred")
      })
    }
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-4xl">
        <DialogHeader>
          <DialogTitle>Create Test Case</DialogTitle>
          <DialogDescription>
            Add a new test case to the project.
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-4">
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
              <Textarea
                value={precondition}
                onChange={(event) => setPrecondition(event.target.value)}
                placeholder="e.g. User is logged out"
                className="min-h-24"
              />
            </FieldContent>
          </Field>

          <Field>
            <FieldLabel>Steps</FieldLabel>
            <FieldContent>
              <Textarea
                value={steps}
                onChange={(event) => setSteps(event.target.value)}
                placeholder="1. Navigate to login page..."
                className="min-h-32"
              />
            </FieldContent>
          </Field>

          <Field>
            <FieldLabel>Expected Result</FieldLabel>
            <FieldContent>
              <Textarea
                value={expectedResult}
                onChange={(event) => setExpectedResult(event.target.value)}
                placeholder="User is redirected to dashboard"
                className="min-h-24"
              />
            </FieldContent>
          </Field>
        </div>

        <DialogFooter>
          <div className="flex w-full items-center justify-end space-x-2">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
            <Button
              type="submit"
              disabled={mutation.isPending || !title.trim() || steps.trim().length === 0 || expectedResult.trim().length === 0}
              onClick={() => mutation.mutate()}
            >
              {mutation.isPending ? "Creating..." : "Create"}
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
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
      toast.error("Failed to update test case", {
        description: getErrorMessage(error, "Unknown error occurred")
      })
    }
  })

  if (!testCase) return null

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-4xl">
        <DialogHeader>
          <DialogTitle>Edit Test Case</DialogTitle>
        </DialogHeader>
        <form onSubmit={(e) => { e.preventDefault(); mutation.mutate() }}>
          <div className="grid gap-4">
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
                <Textarea
                  value={precondition}
                  onChange={(event) => setPrecondition(event.target.value)}
                  className="min-h-24"
                />
              </FieldContent>
            </Field>

            <Field>
              <FieldLabel>Steps</FieldLabel>
              <FieldContent>
                <Textarea
                  value={steps}
                  onChange={(event) => setSteps(event.target.value)}
                  className="min-h-32"
                />
              </FieldContent>
            </Field>

            <Field>
              <FieldLabel>Expected Result</FieldLabel>
              <FieldContent>
                <Textarea
                  value={expectedResult}
                  onChange={(event) => setExpectedResult(event.target.value)}
                  className="min-h-24"
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
          </div>

          <DialogFooter>
            <div className="flex w-full items-center justify-end space-x-2">
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
              <Button
                type="submit"
                disabled={mutation.isPending || !title.trim() || steps.trim().length === 0 || expectedResult.trim().length === 0}
              >
                {mutation.isPending ? "Save Changes" : "Save Changes"}
              </Button>
            </div>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
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
      toast.error("Failed to delete test case", {
        description: getErrorMessage(error, "Unknown error occurred")
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
      toast.error("Failed to record test run", {
        description: getErrorMessage(error, "Unknown error occurred")
      })
    }
  })

  if (!testCase) return null

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-4xl">
        <DialogHeader>
          <DialogTitle>Record Test Run</DialogTitle>
          <DialogDescription>
            Record the execution result for "{testCase.title}".
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={(e) => { e.preventDefault(); mutation.mutate() }}>
          <div className="grid gap-4">
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
          </div>

          <DialogFooter>
            <div className="flex w-full items-center justify-end space-x-2">
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
              <Button type="submit" disabled={mutation.isPending}>
                {mutation.isPending ? "Recording..." : "Record Result"}
              </Button>
            </div>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
