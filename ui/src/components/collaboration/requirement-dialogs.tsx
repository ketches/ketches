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
  CollabPriority,
  RequirementStatus,
  type Requirement,
  type CreateRequirementRequest,
  type UpdateRequirementRequest,
  PlanningStatus
} from "@/api/collaboration"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useState, useEffect } from "react"
import { toast } from "sonner"
import { Combobox, ComboboxContent, ComboboxInput, ComboboxItem, ComboboxList } from "@/components/ui/combobox"

// ── Create Dialog ─────────────────────────────────────────────────────────────

interface CreateRequirementDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectId: string
  parentId?: string
  parentTitle?: string
  onSuccess?: () => void
}

export function CreateRequirementDialog({
  open,
  onOpenChange,
  projectId,
  parentId,
  parentTitle,
  onSuccess
}: CreateRequirementDialogProps) {

  const queryClient = useQueryClient()
  
  const [title, setTitle] = useState("")
  const [description, setDescription] = useState("")
  const [priority, setPriority] = useState<CollabPriority>(CollabPriority.P2)
  const [status, setStatus] = useState<RequirementStatus>(RequirementStatus.TRIAGE)

  // Reset form on open
  useEffect(() => {
    if (open) {
      setTitle("")
      setDescription("")
      setPriority(CollabPriority.P2)
      setStatus(RequirementStatus.TRIAGE)
    }
  }, [open])

  const mutation = useMutation({
    mutationFn: () => {
      const data: CreateRequirementRequest = {
        title,
        description,
        priority,
        status,
        parent_requirement_id: parentId
      }
      
      if (parentId) {
        return collaborationApi.createRequirementChild(projectId, parentId, data)
      } else {
        return collaborationApi.createRequirement(projectId, data)
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["backlog", projectId] })
      queryClient.invalidateQueries({ queryKey: ["requirements", projectId] })
      toast.success(parentId ? "Child requirement created" : "Requirement created")
      onOpenChange(false)
      onSuccess?.()
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: string } } }
      toast.error("Failed to create requirement", {
        description: err.response?.data?.error || "Unknown error occurred"
      })
    }
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>{parentId ? "Create Child Requirement" : "Create Requirement"}{parentTitle && ` for "${parentTitle}"`}</DialogTitle>
          <DialogDescription>
            Add a new requirement to the backlog.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-4">
          <div className="grid gap-2">
            <Label htmlFor="title">Title</Label>
            <Input
              id="title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="e.g. User Authentication"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="grid gap-2">
              <Label>Priority</Label>
              <Combobox value={priority} onValueChange={(v) => v && setPriority(v as CollabPriority)}>
                <ComboboxInput />
                <ComboboxContent>
                  <ComboboxList>
                    {Object.values(CollabPriority).map((p) => (
                      <ComboboxItem key={p} value={p}>{p.toUpperCase()}</ComboboxItem>
                    ))}
                  </ComboboxList>
                </ComboboxContent>
              </Combobox>
            </div>

            <div className="grid gap-2">
              <Label>Status</Label>
              <Combobox value={status} onValueChange={(v) => v && setStatus(v as RequirementStatus)}>
                <ComboboxInput />
                <ComboboxContent>
                  <ComboboxList>
                    {Object.values(RequirementStatus).map((s) => (
                      <ComboboxItem key={s} value={s}>{s.replace('_', ' ').toUpperCase()}</ComboboxItem>
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
              placeholder="Detailed description..."
              rows={5}
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button onClick={() => mutation.mutate()} disabled={mutation.isPending || !title}>
            {mutation.isPending ? "Creating..." : "Create"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ── Edit Dialog ──────────────────────────────────────────────────────────────

interface EditRequirementDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectId: string
  requirement: Requirement | null
  onSuccess?: () => void
}

export function EditRequirementDialog({
  open,
  onOpenChange,
  projectId,
  requirement,
  onSuccess
}: EditRequirementDialogProps) {
  const queryClient = useQueryClient()
  
  const [title, setTitle] = useState("")
  const [description, setDescription] = useState("")
  const [priority, setPriority] = useState<CollabPriority>(CollabPriority.P2)
  const [status, setStatus] = useState<RequirementStatus>(RequirementStatus.TRIAGE)
  const [planningStatus, setPlanningStatus] = useState<PlanningStatus>(PlanningStatus.BACKLOG)

  useEffect(() => {
    if (open && requirement) {
      setTitle(requirement.title)
      setDescription(requirement.description || "")
      setPriority(requirement.priority)
      setStatus(requirement.status)
      setPlanningStatus(requirement.planning_status)
    }
  }, [open, requirement])

  const mutation = useMutation({
    mutationFn: () => {
      if (!requirement) throw new Error("No requirement selected")
      
      const data: UpdateRequirementRequest = {
        title,
        description,
        priority,
        status,
        planning_status: planningStatus,
        assignee_id: requirement.assignee_id,
        sprint_id: requirement.sprint_id
      }
      return collaborationApi.updateRequirement(projectId, requirement.id, data)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["backlog", projectId] })
      queryClient.invalidateQueries({ queryKey: ["requirements", projectId] })
      toast.success("Requirement updated")
      onOpenChange(false)
      onSuccess?.()
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: string } } }
      toast.error("Failed to update requirement", {
        description: err.response?.data?.error || "Unknown error occurred"
      })
    }
  })

  if (!requirement) return null

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>Edit Requirement</DialogTitle>
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
              <Label>Priority</Label>
              <Combobox value={priority} onValueChange={(v) => v && setPriority(v as CollabPriority)}>
                <ComboboxInput />
                <ComboboxContent>
                  <ComboboxList>
                    {Object.values(CollabPriority).map((p) => (
                      <ComboboxItem key={p} value={p}>{p.toUpperCase()}</ComboboxItem>
                    ))}
                  </ComboboxList>
                </ComboboxContent>
              </Combobox>
            </div>

            <div className="grid gap-2">
              <Label>Status</Label>
              <Combobox value={status} onValueChange={(v) => v && setStatus(v as RequirementStatus)}>
                <ComboboxInput />
                <ComboboxContent>
                  <ComboboxList>
                    {Object.values(RequirementStatus).map((s) => (
                      <ComboboxItem key={s} value={s}>{s.replace('_', ' ').toUpperCase()}</ComboboxItem>
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
              rows={5}
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button onClick={() => mutation.mutate()} disabled={mutation.isPending || !title}>
            {mutation.isPending ? "Save Changes" : "Save Changes"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ── Delete Dialog ────────────────────────────────────────────────────────────

interface DeleteRequirementDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectId: string
  requirementId: string
  onSuccess?: () => void
}

export function DeleteRequirementDialog({
  open,
  onOpenChange,
  projectId,
  requirementId,
  onSuccess
}: DeleteRequirementDialogProps) {
  const queryClient = useQueryClient()

  const mutation = useMutation({
    mutationFn: () => collaborationApi.deleteRequirement(projectId, requirementId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["backlog", projectId] })
      queryClient.invalidateQueries({ queryKey: ["requirements", projectId] })
      toast.success("Requirement deleted")
      onOpenChange(false)
      onSuccess?.()
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: string } } }
      toast.error("Failed to delete requirement", {
        description: err.response?.data?.error || "Unknown error occurred"
      })
    }
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete Requirement</DialogTitle>
          <DialogDescription>
            Are you sure you want to delete this requirement? This action cannot be undone.
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
