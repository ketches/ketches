import { useMutation, useQueryClient } from "@tanstack/react-query"
import { type AxiosError } from "axios"
import { InfoIcon, Loader2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { envsApi, type Env } from "@/api/envs"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from "@/components/ui/dialog"
import { Field, FieldContent, FieldError, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { useProjectStore } from "@/stores/project"

interface EditEnvironmentDialogProps {
  open?: boolean
  onOpenChange?: (open: boolean) => void
  env: Env | null
  onSuccess?: () => void
}

export function EditEnvironmentDialog({
  open: controlledOpen,
  onOpenChange: setControlledOpen,
  env,
  onSuccess,
}: EditEnvironmentDialogProps) {
  const [internalOpen, setInternalOpen] = React.useState(false)
  const open = controlledOpen !== undefined ? controlledOpen : internalOpen
  const setOpen = setControlledOpen || setInternalOpen
  const queryClient = useQueryClient()
  const { activeProjectId } = useProjectStore()

  const [errors, setErrors] = React.useState<{
    name?: string
    description?: string
  }>({})

  const [formData, setFormData] = React.useState({
    name: "",
    description: "",
  })

  React.useEffect(() => {
    if (env && open) {
      setFormData({
        name: env.name,
        description: env.description || "",
      })
      setErrors({})
    }
  }, [env, open])

  const updateMutation = useMutation<Env, AxiosError<{ error: string }>, { name: string; description: string }>({
    mutationFn: (data: { name: string; description: string }) => {
      if (!env) throw new Error("No environment selected")
      return envsApi.update(env.id, data)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['envs', activeProjectId] })
      toast.success("Environment updated successfully")
      setOpen(false)
      onSuccess?.()
    },
    onError: (error) => {
      toast.error("Error", {
        description: error.response?.data?.error || "Failed to update environment",
      })
    },
  })

  const validateForm = () => {
    const newErrors: typeof errors = {}

    if (!formData.name.trim()) {
      newErrors.name = "Name is required"
    } else if (formData.name.length < 2) {
      newErrors.name = "Name must be at least 2 characters"
    } else if (formData.name.length > 50) {
      newErrors.name = "Name must be less than 50 characters"
    }

    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const handleSubmit = async (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault()

    if (!validateForm()) {
      return
    }

    updateMutation.mutate({
      name: formData.name,
      description: formData.description,
    })
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="sm:max-w-140 max-h-[90vh] overflow-y-auto">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Edit Environment</DialogTitle>
            <DialogDescription>
              Update the environment name and description.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            <Field>
              <FieldLabel>
                Name *
                <Tooltip>
                  <TooltipTrigger
                    tabIndex={-1}
                    render={
                      <button type="button" className="text-muted-foreground hover:text-foreground transition-colors outline-none">
                        <InfoIcon className="h-3.5 w-3.5" />
                      </button>
                    }
                  />
                  <TooltipContent side="top" align="start" className="max-w-64">
                    <p className="text-xs">2-50 characters.</p>
                  </TooltipContent>
                </Tooltip>
              </FieldLabel>
              <FieldContent>
                <Input
                  placeholder="Production"
                  value={formData.name}
                  onChange={(e) => setFormData((prev) => ({ ...prev, name: e.target.value }))}
                  aria-invalid={!!errors.name}
                />
              </FieldContent>
              {errors.name && (
                <FieldError>
                  <span className="text-destructive text-xs">{errors.name}</span>
                </FieldError>
              )}
            </Field>

            <Field>
              <FieldLabel>Slug</FieldLabel>
              <FieldContent>
                <Input
                  value={env?.slug || ""}
                  disabled
                  className="bg-muted font-mono"
                />
              </FieldContent>
            </Field>

            <Field>
              <FieldLabel>Description</FieldLabel>
              <FieldContent>
                <Textarea
                  placeholder="Brief description..."
                  value={formData.description}
                  onChange={(e) => setFormData((prev) => ({ ...prev, description: e.target.value }))}
                  className="min-h-20 max-h-48 resize-y break-all whitespace-pre-wrap"
                />
              </FieldContent>
            </Field>

          </div>

          <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-end">
            <div className="flex gap-2 sm:justify-end">
              <Button type="button" variant="outline" onClick={() => setOpen(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={updateMutation.isPending}>
                {updateMutation.isPending ? (
                  <>
                    <Loader2 className="h-4 w-4 animate-spin mr-2" />
                    Saving...
                  </>
                ) : (
                  "Save Changes"
                )}
              </Button>
            </div>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

export default EditEnvironmentDialog
