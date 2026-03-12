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
  TestRunStatus,
  type TestCase,
  type CreateTestCaseRequest,
  type UpdateTestCaseRequest,
  type CreateTestRunRequest
} from "@/api/collaboration"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useState, useEffect } from "react"
import { toast } from "sonner"
import { Combobox, ComboboxContent, ComboboxInput, ComboboxItem, ComboboxList } from "@/components/ui/combobox"

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

  // Reset form on open
  useEffect(() => {
    if (open) {
      setTitle("")
      setPrecondition("")
      setSteps("")
      setExpectedResult("")
    }
  }, [open])

  const mutation = useMutation({
    mutationFn: () => {
      const data: CreateTestCaseRequest = {
        title,
        precondition,
        steps,
        expected_result: expectedResult
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
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>Create Test Case</DialogTitle>
          <DialogDescription>
            Add a new test case to the project.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-4">
          <div className="grid gap-2">
            <Label htmlFor="title">Title</Label>
            <Input
              id="title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="e.g. Verify Login Functionality"
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="precondition">Precondition</Label>
            <Textarea
              id="precondition"
              value={precondition}
              onChange={(e) => setPrecondition(e.target.value)}
              placeholder="e.g. User is logged out"
              rows={2}
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="steps">Steps</Label>
            <Textarea
              id="steps"
              value={steps}
              onChange={(e) => setSteps(e.target.value)}
              placeholder="1. Navigate to login page..."
              rows={4}
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="expected-result">Expected Result</Label>
            <Textarea
              id="expected-result"
              value={expectedResult}
              onChange={(e) => setExpectedResult(e.target.value)}
              placeholder="User is redirected to dashboard"
              rows={2}
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button onClick={() => mutation.mutate()} disabled={mutation.isPending || !title || !steps || !expectedResult}>
            {mutation.isPending ? "Creating..." : "Create"}
          </Button>
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

  useEffect(() => {
    if (open && testCase) {
      setTitle(testCase.title)
      setPrecondition(testCase.precondition || "")
      setSteps(testCase.steps)
      setExpectedResult(testCase.expected_result)
    }
  }, [open, testCase])

  const mutation = useMutation({
    mutationFn: () => {
      if (!testCase) throw new Error("No test case selected")
      
      const data: UpdateTestCaseRequest = {
        title,
        precondition,
        steps,
        expected_result: expectedResult,
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
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>Edit Test Case</DialogTitle>
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

          <div className="grid gap-2">
            <Label htmlFor="edit-precondition">Precondition</Label>
            <Textarea
              id="edit-precondition"
              value={precondition}
              onChange={(e) => setPrecondition(e.target.value)}
              rows={2}
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="edit-steps">Steps</Label>
            <Textarea
              id="edit-steps"
              value={steps}
              onChange={(e) => setSteps(e.target.value)}
              rows={4}
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="edit-expected-result">Expected Result</Label>
            <Textarea
              id="edit-expected-result"
              value={expectedResult}
              onChange={(e) => setExpectedResult(e.target.value)}
              rows={2}
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button onClick={() => mutation.mutate()} disabled={mutation.isPending || !title || !steps || !expectedResult}>
            {mutation.isPending ? "Save Changes" : "Save Changes"}
          </Button>
        </DialogFooter>
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
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>Record Test Run</DialogTitle>
          <DialogDescription>
            Record the execution result for "{testCase.title}".
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-4">
          <div className="grid gap-2">
            <Label>Status</Label>
            <Combobox value={status} onValueChange={(v) => v && setStatus(v as TestRunStatus)}>
              <ComboboxInput />
              <ComboboxContent>
                <ComboboxList>
                  {Object.values(TestRunStatus).map((s) => (
                    <ComboboxItem key={s} value={s}>{s.toUpperCase()}</ComboboxItem>
                  ))}
                </ComboboxList>
              </ComboboxContent>
            </Combobox>
          </div>

          <div className="grid gap-2">
            <Label htmlFor="run-comment">Comment (Optional)</Label>
            <Textarea
              id="run-comment"
              value={comment}
              onChange={(e) => setComment(e.target.value)}
              placeholder="Any observations or details..."
              rows={3}
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button onClick={() => mutation.mutate()} disabled={mutation.isPending}>
            {mutation.isPending ? "Recording..." : "Record Result"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
