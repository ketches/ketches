import { useMutation, useQueryClient } from "@tanstack/react-query"
import { Loader2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import type { Template } from "@/api/templates"
import { templatesApi, type CreateTemplateRequest } from "@/api/templates"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Field, FieldContent, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"

interface CreateTemplateDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectId: string
  onSuccess?: (template: Template) => void
}

const defaultForm: CreateTemplateRequest = {
  name: "",
  slug: "",
  description: "",
  type: "application",
  content: "",
  status: "draft",
  enabled: true,
}

// Generate slug from name
function generateSlug(name: string): string {
  return name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-|-$/g, "")
    .slice(0, 128)
}

export function CreateTemplateDialog({
  open,
  onOpenChange,
  projectId,
  onSuccess,
}: CreateTemplateDialogProps) {
  const queryClient = useQueryClient()
  const [form, setForm] = React.useState<CreateTemplateRequest>(defaultForm)
  const [autoSlug, setAutoSlug] = React.useState(true)

  const createMutation = useMutation({
    mutationFn: () => templatesApi.create(projectId, form),
    onSuccess: (template) => {
      queryClient.invalidateQueries({ queryKey: ["templates", projectId] })
      toast.success("Template created")
      onOpenChange(false)
      setForm(defaultForm)
      setAutoSlug(true)
      onSuccess?.(template)
    },
    onError: (err: unknown) => {
      const msg =
        err && typeof err === "object" && "response" in err
          ? (err as { response?: { data?: { error?: string } } }).response?.data
            ?.error
          : null
      toast.error(msg || "Failed to create template")
    },
  })

  const handleSubmit = () => {
    if (!form.name?.trim()) {
      toast.error("Name is required")
      return
    }
    if (!form.slug?.trim()) {
      toast.error("Slug is required")
      return
    }
    createMutation.mutate()
  }

  const handleNameChange = (name: string) => {
    const newForm = { ...form, name }
    if (autoSlug) {
      newForm.slug = generateSlug(name)
    }
    setForm(newForm)
  }

  const handleSlugChange = (slug: string) => {
    setAutoSlug(false)
    setForm({ ...form, slug })
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-160 max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Create Template</DialogTitle>
          <DialogDescription>
            Create a new template. Templates define reusable configurations for
            applications, services, and other resources.
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-4 py-4">
          <div className="grid grid-cols-2 gap-4">
            <Field>
              <FieldLabel>Name *</FieldLabel>
              <FieldContent>
                <Input
                  placeholder="My Template"
                  value={form.name}
                  onChange={(e) => handleNameChange(e.target.value)}
                  required
                />
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel>Slug *</FieldLabel>
              <FieldContent>
                <Input
                  placeholder="my-template"
                  value={form.slug}
                  onChange={(e) => handleSlugChange(e.target.value)}
                  required
                />
              </FieldContent>
            </Field>
          </div>
          <Field>
            <FieldLabel>Description</FieldLabel>
            <FieldContent>
              <Textarea
                placeholder="A brief description of what this template does..."
                value={form.description ?? ""}
                onChange={(e) =>
                  setForm({ ...form, description: e.target.value })
                }
                rows={3}
              />
            </FieldContent>
          </Field>
          <div className="grid grid-cols-2 gap-4">
            <Field>
              <FieldLabel>Type</FieldLabel>
              <FieldContent>
                <Select
                  value={form.type || "application"}
                  onValueChange={(v) => setForm({ ...form, type: v })}
                  items={[
                    { value: "application", label: "Application" },
                    { value: "service", label: "Service" },
                    { value: "job", label: "Job" },
                    { value: "cronjob", label: "CronJob" },
                  ]}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder="Select type" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="application">Application</SelectItem>
                    <SelectItem value="service">Service</SelectItem>
                    <SelectItem value="job">Job</SelectItem>
                    <SelectItem value="cronjob">CronJob</SelectItem>
                  </SelectContent>
                </Select>
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel>Status</FieldLabel>
              <FieldContent>
                <Select
                  value={form.status || "draft"}
                  onValueChange={(v) => setForm({ ...form, status: v })}
                  items={[
                    { value: "draft", label: "Draft" },
                    { value: "reviewing", label: "Reviewing" },
                    { value: "published", label: "Published" },
                    { value: "deprecated", label: "Deprecated" },
                  ]}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder="Select status" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="draft">Draft</SelectItem>
                    <SelectItem value="reviewing">Reviewing</SelectItem>
                    <SelectItem value="published">Published</SelectItem>
                    <SelectItem value="deprecated">Deprecated</SelectItem>
                  </SelectContent>
                </Select>
              </FieldContent>
            </Field>
          </div>
          <Field>
            <FieldLabel>Content</FieldLabel>
            <FieldContent>
              <Textarea
                placeholder="Template content (YAML, JSON, etc.)..."
                value={form.content ?? ""}
                onChange={(e) => setForm({ ...form, content: e.target.value })}
                rows={8}
                className="font-mono text-sm"
              />
            </FieldContent>
          </Field>
          <div className="flex items-center gap-2">
            <Checkbox
              id="enabled-checkbox"
              checked={form.enabled ?? true}
              onCheckedChange={(v) => setForm({ ...form, enabled: v === true })}
            />
            <label htmlFor="enabled-checkbox" className="cursor-pointer">
              Enabled
            </label>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={createMutation.isPending}>
            {createMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : null}
            Create
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
