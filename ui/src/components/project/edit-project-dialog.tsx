import { useMutation, useQueryClient } from "@tanstack/react-query"
import * as React from "react"
import { toast } from "sonner"

import { projectsApi, type Project } from "@/api/projects"
import { ProjectAiProvidersPanel } from "@/components/project/project-ai-providers-panel"
import { ProjectSettingsShell, type ProjectSettingsSection } from "@/components/project/project-settings-shell"
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

interface EditProjectDialogProps {
  open?: boolean
  onOpenChange?: (open: boolean) => void
  project?: Project | null
  onSuccess?: () => void
}

export function EditProjectDialog({
  open: controlledOpen,
  onOpenChange: setControlledOpen,
  project,
  onSuccess,
}: EditProjectDialogProps) {
  const [internalOpen, setInternalOpen] = React.useState(false)
  const open = controlledOpen !== undefined ? controlledOpen : internalOpen
  const setOpen = setControlledOpen || setInternalOpen

  const queryClient = useQueryClient()
  const [errors, setErrors] = React.useState<{ name?: string; global?: string }>({})
  const [activeSection, setActiveSection] = React.useState<ProjectSettingsSection>("general")

  const [formData, setFormData] = React.useState({
    name: "",
    description: "",
    collaborationEnabled: false,
  })

  // Reset form when project changes
  React.useEffect(() => {
    if (project) {
      setFormData({
        name: project.name || "",
        description: project.description || "",
        collaborationEnabled: !!project.collaboration_enabled,
      })
      setErrors({})
    }
  }, [project])

  React.useEffect(() => {
    if (open) {
      setActiveSection("general")
    }
  }, [open])

  const mutation = useMutation({
    mutationFn: (data: typeof formData) => {
      if (!project) throw new Error("No project selected")
      return projectsApi.update(project.id, {
        name: data.name,
        description: data.description,
        collaboration_enabled: data.collaborationEnabled,
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] })
      toast.success("Project updated successfully")
      onSuccess?.()
      setOpen(false)
      setErrors({})
    },
    onError: (err: AxiosError<{ error: string }>) => {
      const errMsg = err.response?.data?.error || "Failed to update project"
      setErrors({ global: errMsg })
      toast.error("Error", { description: errMsg })
    }
  })

  const validateForm = () => {
    const newErrors: { name?: string } = {}

    if (!formData.name.trim()) {
      newErrors.name = "Name is required"
    }

    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!validateForm()) return
    mutation.mutate(formData)
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="sm:max-w-[50vw] w-full min-h-[75vh]" showCloseButton>
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Edit Project</DialogTitle>
            <DialogDescription>
              Update the project name and description.
            </DialogDescription>
          </DialogHeader>

          <ProjectSettingsShell activeSection={activeSection} onSectionChange={setActiveSection}>
            {activeSection === "general" ? (
              <div className="grid gap-4 py-4">
                {errors.global && (
                  <div className="text-sm font-medium text-destructive text-center">
                    {errors.global}
                  </div>
                )}
                <Field>
                  <FieldLabel>Name *</FieldLabel>
                  <FieldContent>
                    <Input
                      placeholder="My Project"
                      value={formData.name}
                      onChange={(e) => setFormData((prev) => ({ ...prev, name: e.target.value }))}
                      aria-invalid={!!errors.name}
                    />
                  </FieldContent>
                  {errors.name && <FieldError><span className="text-destructive text-xs">{errors.name}</span></FieldError>}
                </Field>

                <Field>
                  <FieldLabel>Slug</FieldLabel>
                  <FieldContent>
                    <Input
                      value={project?.slug || ""}
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

                <div className="flex items-center gap-2">
                  <Checkbox
                    id="project-edit-collaboration-enabled"
                    checked={formData.collaborationEnabled}
                    onCheckedChange={(v) => setFormData((prev) => ({ ...prev, collaborationEnabled: v === true }))}
                  />
                  <label htmlFor="project-edit-collaboration-enabled" className="cursor-pointer">
                    Enable collaboration module for this project
                  </label>
                </div>
              </div>
            ) : null}

            {activeSection === "ai-providers" ? (
              project ? <ProjectAiProvidersPanel projectId={project.id} /> : null
            ) : null}
          </ProjectSettingsShell>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={mutation.isPending}>
              {mutation.isPending ? "Updating..." : "Update"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

export default EditProjectDialog
