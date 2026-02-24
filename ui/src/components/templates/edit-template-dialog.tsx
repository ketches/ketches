import { useMutation, useQueryClient } from "@tanstack/react-query"
import { Loader2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import {
  templatesApi,
  type Template,
  type UpdateTemplateRequest,
} from "@/api/templates"
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

interface EditTemplateDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  template: Template | null
  onSuccess?: () => void
}

export function EditTemplateDialog({
  open,
  onOpenChange,
  template,
  onSuccess,
}: EditTemplateDialogProps) {
  const queryClient = useQueryClient()
  const [form, setForm] = React.useState<UpdateTemplateRequest>({})

  React.useEffect(() => {
    if (template && open) {
      setForm({
        name: template.name,
        slug: template.slug,
        description: template.description,
        type: template.type,
        content: template.content,
        status: template.status,
        enabled: template.enabled,
      })
    }
  }, [template, open])

  const updateMutation = useMutation({
    mutationFn: () => {
      const { slug, ...updateData } = form
      return templatesApi.update(template!.id, updateData)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["template", template?.id],
      })
      queryClient.invalidateQueries({ queryKey: ["templates"] })
      toast.success("Template updated")
      onOpenChange(false)
      onSuccess?.()
    },
    onError: (err: unknown) => {
      const msg =
        err && typeof err === "object" && "response" in err
          ? (err as { response?: { data?: { error?: string } } }).response?.data
            ?.error
          : null
      toast.error(msg || "Failed to update template")
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
    updateMutation.mutate()
  }

  if (!template) return null

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-160 max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Edit Template</DialogTitle>
          <DialogDescription>
            Update the template configuration, content, and status.
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-4 py-4">
          <div className="grid grid-cols-2 gap-4">
            <Field>
              <FieldLabel>Name *</FieldLabel>
              <FieldContent>
                <Input
                  value={form.name ?? ""}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                  required
                />
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel>Slug *</FieldLabel>
              <FieldContent>
                <Input
                  value={form.slug ?? ""}
                  disabled
                  className="bg-muted font-mono"
                />
              </FieldContent>
            </Field>
          </div>
          <Field>
            <FieldLabel>Description</FieldLabel>
            <FieldContent>
              <Textarea
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
                value={form.content ?? ""}
                onChange={(e) => setForm({ ...form, content: e.target.value })}
                rows={8}
                className="font-mono text-sm"
              />
            </FieldContent>
          </Field>
          <div className="flex items-center gap-2">
            <Checkbox
              id="edit-enabled-checkbox"
              checked={form.enabled ?? true}
              onCheckedChange={(v) => setForm({ ...form, enabled: v === true })}
            />
            <label htmlFor="edit-enabled-checkbox" className="cursor-pointer">
              Enabled
            </label>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={updateMutation.isPending}>
            {updateMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : null}
            Save
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
