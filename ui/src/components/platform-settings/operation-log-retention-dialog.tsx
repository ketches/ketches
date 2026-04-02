import { operationLogsApi } from "@/api/operation-logs"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldContent, FieldDescription, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { Loader2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

interface OperationLogRetentionDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  retentionDays: number
}

export function OperationLogRetentionDialog({
  open,
  onOpenChange,
  retentionDays,
}: OperationLogRetentionDialogProps) {
  const queryClient = useQueryClient()
  const [value, setValue] = React.useState(String(retentionDays))
  const mutation = useMutation({
    mutationFn: (days: number) => operationLogsApi.updateOperationLogSettings(days),
  })

  React.useEffect(() => {
    if (open) {
      setValue(String(retentionDays))
    }
  }, [open, retentionDays])

  const handleSubmit = (event: React.SubmitEvent<HTMLFormElement>) => {
    event.preventDefault()

    const parsedDays = Number(value)
    if (!Number.isInteger(parsedDays) || parsedDays < 1) {
      toast.error("Retention days must be a positive integer")
      return
    }

    mutation.mutate(parsedDays, {
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ["operation-log-settings"] })
        toast.success("Retention settings updated")
        onOpenChange(false)
      },
      onError: (error) => {
        toast.error("Failed to update retention settings", {
          description: error instanceof Error ? error.message : String(error),
        })
      },
    })
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <form onSubmit={handleSubmit} className="space-y-4" noValidate>
          <DialogHeader>
            <DialogTitle>Edit Operation Log Retention</DialogTitle>
            <DialogDescription>
              Choose how many days operation logs should remain available before cleanup.
            </DialogDescription>
          </DialogHeader>

          <Field>
            <FieldLabel htmlFor="retention-days">Retention Days</FieldLabel>
            <FieldContent>
              <Input
                id="retention-days"
                name="retention-days"
                type="number"
                min={1}
                value={value}
                onChange={(event) => setValue(event.target.value)}
                disabled={mutation.isPending}
                required
              />
              <FieldDescription>
                Enter a positive integer. Older operation logs beyond this window may be removed.
              </FieldDescription>
            </FieldContent>
          </Field>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={mutation.isPending}>
              {mutation.isPending ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Saving...
                </>
              ) : (
                "Save"
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
