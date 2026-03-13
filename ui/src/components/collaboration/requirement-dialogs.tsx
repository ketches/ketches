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
  CollabPriority,
  CollabPriorityOptions,
  PlanningStatus,
  RequirementStatus,
  RequirementStatusOptions,
  type CreateRequirementRequest,
  type Requirement,
  type UpdateRequirementRequest
} from "@/api/collaboration"
import { Combobox, ComboboxContent, ComboboxInput, ComboboxItem, ComboboxList } from "@/components/ui/combobox"
import { Input } from "@/components/ui/input"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useEffect, useState } from "react"
import { toast } from "sonner"
import { Field, FieldContent, FieldLabel } from "../ui/field"
import { RichTextEditor } from "./rich-text-editor"

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
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="sm:max-w-xl overflow-y-auto">
        <form onSubmit={(e) => { e.preventDefault(); mutation.mutate() }} className="space-y-4">
          <SheetHeader>
            <SheetTitle>{parentId ? "Create Child Requirement" : "Create Requirement"}{parentTitle && ` for "${parentTitle}"`}</SheetTitle>
            <SheetDescription>
              Add a new requirement to the backlog.
            </SheetDescription>
          </SheetHeader>

          <Field>
            <FieldLabel>Title</FieldLabel>
            <FieldContent>
              <Input
                id="title"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="e.g. User Authentication"
              />
            </FieldContent>
          </Field>

          <div className="grid grid-cols-2 gap-4">
            <Field>
              <FieldLabel>Priority</FieldLabel>
              <FieldContent>
                <Combobox value={priority} onValueChange={(v) => v && setPriority(v as CollabPriority)} itemToStringLabel={(item) => CollabPriorityOptions.find(opt => opt.value === item)?.label || item}>
                  <ComboboxInput />
                  <ComboboxContent>
                    <ComboboxList>
                      {CollabPriorityOptions.map((p) => (
                        <ComboboxItem key={p.value} value={p.value}>{p.label}</ComboboxItem>
                      ))}
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
              </FieldContent>
            </Field>

            <Field>
              <FieldLabel>Status</FieldLabel>
              <FieldContent>
                <Combobox value={status} onValueChange={(v) => v && setStatus(v as RequirementStatus)} itemToStringLabel={(item) => RequirementStatusOptions.find(opt => opt.value === item)?.label || item}>
                  <ComboboxInput />
                  <ComboboxContent>
                    <ComboboxList>
                      {RequirementStatusOptions.map((s) => (
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
              <RichTextEditor
                value={description}
                onChange={setDescription}
                placeholder="Detailed description..."
              />
            </FieldContent>
          </Field>

          <SheetFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
            <Button type="submit" disabled={mutation.isPending || !title}>
              {mutation.isPending ? "Creating..." : "Create"}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
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
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="sm:max-w-xl overflow-y-auto">
        <form onSubmit={(e) => { e.preventDefault(); mutation.mutate() }} className="space-y-4">
          <SheetHeader>
            <SheetTitle>Edit Requirement</SheetTitle>
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

          <div className="grid grid-cols-2 gap-4">
            <Field>
              <FieldLabel>Priority</FieldLabel>
              <FieldContent>
                <Combobox value={priority} onValueChange={(v) => v && setPriority(v as CollabPriority)} itemToStringLabel={(item) => CollabPriorityOptions.find(opt => opt.value === item)?.label || item}>
                  <ComboboxInput />
                  <ComboboxContent>
                    <ComboboxList>
                      {CollabPriorityOptions.map((p) => (
                        <ComboboxItem key={p.value} value={p.value}>{p.label}</ComboboxItem>
                      ))}
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
              </FieldContent>
            </Field>

            <Field>
              <FieldLabel>Status</FieldLabel>
              <FieldContent>
                <Combobox value={status} onValueChange={(v) => v && setStatus(v as RequirementStatus)} itemToStringLabel={(item) => RequirementStatusOptions.find(opt => opt.value === item)?.label || item}>
                  <ComboboxInput />
                  <ComboboxContent>
                    <ComboboxList>
                      {RequirementStatusOptions.map((s) => (
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
              <RichTextEditor
                value={description}
                onChange={setDescription}
              />
            </FieldContent>
          </Field>

          <SheetFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
            <Button type="submit" disabled={mutation.isPending || !title}>
              {mutation.isPending ? "Save Changes" : "Save Changes"}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
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
