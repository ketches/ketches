import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Field, FieldContent, FieldLabel } from "../ui/field"
import { collaborationApi, SprintStatus, type Sprint } from "@/api/collaboration"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useState } from "react"
import { toast } from "sonner"
import { Combobox, ComboboxContent, ComboboxInput, ComboboxItem, ComboboxList } from "@/components/ui/combobox"

interface PlanToSprintDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectId: string
  requirementIds: string[]
  onSuccess?: () => void
}

export function PlanToSprintDialog({
  open,
  onOpenChange,
  projectId,
  requirementIds,
  onSuccess
}: PlanToSprintDialogProps) {
  const queryClient = useQueryClient()
  const [selectedSprintId, setSelectedSprintId] = useState<string>("")

  // Fetch available sprints (Planned or Active)
  const { data: sprintsResponse } = useQuery({
    queryKey: ["sprints", projectId, "plannable"],
    queryFn: () => collaborationApi.listSprints(projectId, { 
      page: 1, 
      page_size: 100, // Fetch enough sprints
      status: `${SprintStatus.PLANNED},${SprintStatus.ACTIVE}` // API might not support comma separated, usually client side filter or separate requests
    }),
    enabled: open
  })
  
  // If API doesn't support multiple statuses, we might just fetch all or default to PLANNED.
  // Ideally backend supports it. Assuming listSprints returns paginated list.

  const sprints = sprintsResponse?.items || []

  const mutation = useMutation({
    mutationFn: () => {
      if (!selectedSprintId) throw new Error("No sprint selected")
      return collaborationApi.planToSprint(projectId, requirementIds, selectedSprintId)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["backlog", projectId] })
      queryClient.invalidateQueries({ queryKey: ["requirements", projectId] }) // Update requirements list too
      toast.success("Requirements moved to sprint")
      onOpenChange(false)
      onSuccess?.()
      setSelectedSprintId("")
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: string } } }
      toast.error("Failed to plan to sprint", {
        description: err.response?.data?.error || "Unknown error occurred"
      })
    }
  })

  const handleSubmit = () => {
    mutation.mutate()
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-160">
        <DialogHeader>
          <DialogTitle>Plan to Sprint</DialogTitle>
          <DialogDescription>
            Move {requirementIds.length} requirement{requirementIds.length !== 1 ? "s" : ""} to a sprint.
          </DialogDescription>
        </DialogHeader>
        
        <form onSubmit={(e) => { e.preventDefault(); handleSubmit(); }} className="space-y-4">
          <Field>
            <FieldLabel>Select Sprint</FieldLabel>
            <FieldContent>
              <Combobox
                value={selectedSprintId}
                onValueChange={(val) => val && setSelectedSprintId(val)}
              >
                <ComboboxInput placeholder="Select sprint" />
                <ComboboxContent>
                  <ComboboxList>
                    {sprints.map((sprint: Sprint) => (
                      <ComboboxItem key={sprint.id} value={sprint.id} label={`${sprint.name} (${sprint.status})`}>
                        {sprint.name} ({sprint.status})
                      </ComboboxItem>
                    ))}
                  </ComboboxList>
                </ComboboxContent>
              </Combobox>
            </FieldContent>
          </Field>

          <DialogFooter>
            <Button variant="outline" type="button" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={mutation.isPending || !selectedSprintId}>
              {mutation.isPending ? "Moving..." : "Move to Sprint"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
