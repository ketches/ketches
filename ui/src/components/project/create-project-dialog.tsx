import { useMutation, useQueryClient } from "@tanstack/react-query"
import * as React from "react"
import { toast } from "sonner"

import { projectsApi } from "@/api/projects"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
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
import type { AxiosError } from "axios"

interface CreateProjectFormProps {
  open?: boolean
  onOpenChange?: (open: boolean) => void
  onSuccess?: (project: { id: string; name: string; slug: string }) => void
  onClose?: () => void
}

export function CreateProjectDialog({
  open: controlledOpen,
  onOpenChange: setControlledOpen,
  onSuccess,
  onClose,
}: CreateProjectFormProps) {
  const [internalOpen, setInternalOpen] = React.useState(false)
  const open = controlledOpen !== undefined ? controlledOpen : internalOpen
  const setOpen = setControlledOpen || setInternalOpen

  const queryClient = useQueryClient()
  const [errors, setErrors] = React.useState<{ name?: string; slug?: string }>({})

  const [formData, setFormData] = React.useState({
    name: "",
    slug: "",
    description: "",
    collaborationEnabled: false,
  })

  const mutation = useMutation({
    mutationFn: (data: typeof formData) => projectsApi.create({
      name: data.name,
      slug: data.slug,
      description: data.description,
      collaboration_enabled: data.collaborationEnabled,
    }),
    onSuccess: (project) => {
      queryClient.invalidateQueries({ queryKey: ['projects'] })
      queryClient.invalidateQueries({ queryKey: ['projects-simple'] })
      toast.success("Project created successfully")
      onSuccess?.(project)
      setOpen(false)
      onClose?.()
      setFormData({ name: "", slug: "", description: "", collaborationEnabled: false })
      setErrors({})
    },
    onError: (err: AxiosError<{ error: string }>) => {
      const errMsg = err.response?.data?.error || "Failed to create project"
      toast.error("Error", { description: errMsg })
    }
  })

  const handleNameChange = (name: string) => {
    setFormData((prev) => ({ ...prev, name }))
    const slug = name
      .toLowerCase()
      .replace(/[^a-z0-9\s-]/g, "")
      .replace(/\s+/g, "-")
      .replace(/-+/g, "-")
      .replace(/^-|-$/g, "")
    setFormData((prev) => ({ ...prev, slug }))
  }

  const validateForm = () => {
    const newErrors: { name?: string; slug?: string } = {}

    if (!formData.name.trim()) {
      newErrors.name = "Name is required"
    }

    if (!formData.slug.trim()) {
      newErrors.slug = "Slug is required"
    }

    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const handleSubmit = async (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault()
    if (!validateForm()) return
    mutation.mutate(formData)
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="sm:max-w-160">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Create Project</DialogTitle>
            <DialogDescription>
              Create a new project to organize your environments and applications.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            <div className="grid grid-cols-2 gap-4">
              <Field>
                <FieldLabel>Name *</FieldLabel>
                <FieldContent>
                  <Input
                    placeholder="My Project"
                    value={formData.name}
                    onChange={(e) => handleNameChange(e.target.value)}
                    aria-invalid={!!errors.name}
                  />
                </FieldContent>
                {errors.name && <FieldError><span className="text-destructive text-xs">{errors.name}</span></FieldError>}
              </Field>

              <Field>
                <FieldLabel>Slug *</FieldLabel>
                <FieldContent>
                  <Input
                    placeholder="my-project"
                    value={formData.slug}
                    onChange={(e) => setFormData((prev) => ({ ...prev, slug: e.target.value }))}
                    aria-invalid={!!errors.slug}
                  />
                </FieldContent>
                {errors.slug && <FieldError><span className="text-destructive text-xs">{errors.slug}</span></FieldError>}
              </Field>
            </div>

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

            <div className="flex items-center gap-2">
              <Checkbox
                id="project-collaboration-enabled"
                checked={formData.collaborationEnabled}
                onCheckedChange={(v) => setFormData((prev) => ({ ...prev, collaborationEnabled: v === true }))}
              />
              <label htmlFor="project-collaboration-enabled" className="cursor-pointer">
                Enable collaboration module for this project
              </label>
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={mutation.isPending}>
              {mutation.isPending ? "Creating..." : "Create"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

export default CreateProjectDialog
