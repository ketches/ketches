import { useMutation, useQueryClient } from "@tanstack/react-query"
import { type AxiosError } from "axios"
import { Loader2 } from "lucide-react"
import * as React from "react"
import { toast as sonnerToast } from "sonner"

import { appsApi, type App } from "@/api/apps"
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
import { useProjectStore } from "@/stores/project"

interface EditAppDialogProps {
  open?: boolean
  onOpenChange?: (open: boolean) => void
  app: App | null
  onSuccess?: () => void
}

export function EditAppDialog({
  open: controlledOpen,
  onOpenChange: setControlledOpen,
  app,
  onSuccess,
}: EditAppDialogProps) {
  const [internalOpen, setInternalOpen] = React.useState(false)
  const open = controlledOpen !== undefined ? controlledOpen : internalOpen
  const setOpen = setControlledOpen || setInternalOpen
  const queryClient = useQueryClient()
  const { activeEnvId } = useProjectStore()

  const [errors, setErrors] = React.useState<{
    name?: string
    description?: string
  }>({})

  const [formData, setFormData] = React.useState({
    name: "",
    description: "",
  })

  React.useEffect(() => {
    if (app && open) {
      setFormData({
        name: app.name,
        description: app.description || "",
      })
      setErrors({})
    }
  }, [app, open])

  const updateMutation = useMutation<App, AxiosError<{ error: string }>, { name: string; description: string }>({
    mutationFn: (data: { name: string; description: string }) => {
      if (!app) throw new Error("No application selected")
      return appsApi.updateBasic(app.id, data)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['apps', activeEnvId] })
      sonnerToast.success("Application updated successfully")
      setOpen(false)
      onSuccess?.()
    },
    onError: (error) => {
      sonnerToast.error("Error", {
        description: error.response?.data?.error || "Failed to update application",
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
            <DialogTitle>Edit Application</DialogTitle>
            <DialogDescription>
              Update the application name and description.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            <Field>
              <FieldLabel>Name *</FieldLabel>
              <FieldContent>
                <Input
                  placeholder="My App"
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
                  value={app?.slug || ""}
                  disabled
                  className="bg-muted font-mono"
                />
              </FieldContent>
            </Field>

            <Field>
              <FieldLabel>Description</FieldLabel>
              <FieldContent>
                <Textarea
                  placeholder="Brief description of this application..."
                  className="min-h-20 max-h-48 resize-y break-all whitespace-pre-wrap"
                  value={formData.description}
                  onChange={(e) => setFormData((prev) => ({ ...prev, description: e.target.value }))}
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

export default EditAppDialog
