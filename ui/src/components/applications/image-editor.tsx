import { useMutation, useQueryClient } from "@tanstack/react-query"
import { Loader2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

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
import { Field, FieldContent, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"

interface ImageEditorProps {
  open?: boolean
  onOpenChange?: (open: boolean) => void
  app: App | null
  onSuccess?: () => void
}

export function ImageEditor({
  open: controlledOpen,
  onOpenChange: setControlledOpen,
  app,
  onSuccess,
}: ImageEditorProps) {
  const [internalOpen, setInternalOpen] = React.useState(false)
  const open = controlledOpen !== undefined ? controlledOpen : internalOpen
  const setOpen = setControlledOpen || setInternalOpen
  const queryClient = useQueryClient()

  const [formData, setFormData] = React.useState({
    container_image: "",
    registry_username: "",
    registry_password: "",
  })

  React.useEffect(() => {
    if (app && open) {
      setFormData({
        container_image: app.container_image,
        registry_username: app.registry_username || "",
        registry_password: app.registry_password || "",
      })
    }
  }, [app, open])

  const mutation = useMutation({
    mutationFn: (data: { container_image: string; registry_username?: string; registry_password?: string }) => {
      if (!app) throw new Error("No application selected")
      return appsApi.update(app.id, {
        name: app.name,
        description: app.description,
        ...data
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['app', app?.id] })
      toast.success("Application image updated successfully")
      setOpen(false)
      onSuccess?.()
    },
    onError: (error: any) => {
      toast.error("Error", {
        description: error.response?.data?.error || "Failed to update application image",
      })
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    mutation.mutate(formData)
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="sm:max-w-140">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Update Application Image</DialogTitle>
            <DialogDescription>
              Update the container image and optional registry credentials.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            <Field>
              <FieldLabel>Container Image *</FieldLabel>
              <FieldContent>
                <Input
                  placeholder="e.g. nginx:latest"
                  value={formData.container_image}
                  onChange={(e) => setFormData((prev) => ({ ...prev, container_image: e.target.value }))}
                  required
                />
              </FieldContent>
            </Field>

            <div className="grid gap-4">
              <Field>
                <FieldLabel>Registry Username</FieldLabel>
                <FieldContent>
                  <Input
                    placeholder="Registry username"
                    value={formData.registry_username}
                    onChange={(e) => setFormData((prev) => ({ ...prev, registry_username: e.target.value }))}
                  />
                </FieldContent>
              </Field>
              <Field>
                <FieldLabel>Registry Password</FieldLabel>
                <FieldContent>
                  <Input
                    type="password"
                    autoComplete="new-password"
                    placeholder="Registry password"
                    value={formData.registry_password}
                    onChange={(e) => setFormData((prev) => ({ ...prev, registry_password: e.target.value }))}
                  />
                </FieldContent>
              </Field>
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={mutation.isPending}>
              {mutation.isPending ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin mr-2" />
                  Saving...
                </>
              ) : (
                "Save Changes"
              )}
            </Button>
          </DialogFooter>
        </form >
      </DialogContent >
    </Dialog >
  )
}
