import { useMutation, useQueryClient } from "@tanstack/react-query"
import { Layers2, Zap } from "lucide-react"
import { useEffect, useState } from "react"
import { toast } from "sonner"

import { pluginsApi } from "@/api/plugins"
import { KeyValueInput, type KeyValuePair } from "@/components/shared/key-value-input"
import { Button } from "@/components/ui/button"
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
import { Textarea } from "@/components/ui/textarea"
import { cn } from "@/lib/utils"
import type { AxiosError } from "axios"

interface EditPluginDialogProps {
  plugin: any
  open: boolean
  onOpenChange: (open: boolean) => void
}

const pluginTypes = [
  {
    id: "init" as const,
    title: "Init Container",
    description: "Runs before app starts. Perfect for setup tasks like database migrations, configuration, or dependency checks.",
    icon: Zap,
  },
  {
    id: "sidecar" as const,
    title: "Sidecar Container",
    description: "Runs alongside app. Ideal for logging agents, proxies, monitoring tools, or service mesh components.",
    icon: Layers2,
  },
]

export function EditPluginDialog({ plugin, open, onOpenChange }: EditPluginDialogProps) {
  const queryClient = useQueryClient()
  const [formData, setFormData] = useState({
    name: "",
    slug: "",
    description: "",
    image: "",
    registry_username: "",
    registry_password: "",
    command: "",
    plugin_type: "init" as "init" | "sidecar"
  })
  const [envVars, setEnvVars] = useState<KeyValuePair[]>([])

  useEffect(() => {
    if (plugin) {
      setFormData({
        name: plugin.name || "",
        slug: plugin.slug || "",
        description: plugin.description || "",
        image: plugin.image || "",
        registry_username: plugin.registry_username || "",
        registry_password: "",
        command: plugin.command || "",
        plugin_type: plugin.plugin_type || "init"
      })
      setEnvVars(Array.isArray(plugin.env_vars) ? plugin.env_vars : [])
    }
  }, [plugin])

  const updateMutation = useMutation({
    mutationFn: (data: any) => pluginsApi.updatePlugin(plugin.id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['plugins'] })
      toast.success("Plugin updated successfully")
      onOpenChange(false)
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error("Failed to update plugin", {
        description: err.response?.data?.error || "An unknown error occurred"
      })
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

  const validateRequiredFields = () => {
    if (!formData.name || !formData.slug || !formData.image) {
      toast.error("Please fill in all required fields")
      return false
    }
    return true
  }


  const handleSubmit = (e: React.SubmitEvent) => {
    e.preventDefault()

    if (!validateRequiredFields()) return

    const payload: any = {
      name: formData.name,
      description: formData.description,
      image: formData.image,
      command: formData.command || undefined,
      env_vars: envVars,
      plugin_type: formData.plugin_type
    }

    if (formData.registry_username) {
      payload.registry_username = formData.registry_username
    }
    if (formData.registry_password) {
      payload.registry_password = formData.registry_password
    }

    updateMutation.mutate(payload)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-160 max-h-[90vh] overflow-y-auto">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Edit Plugin</DialogTitle>
            <DialogDescription>
              Update plugin configuration. Leave password blank to keep existing credentials.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            <div className="grid grid-cols-2 gap-4">
              <Field>
                <FieldLabel htmlFor="edit-name">Name *</FieldLabel>
                <FieldContent>
                  <Input
                    id="edit-name"
                    value={formData.name}
                    onChange={(e) => handleNameChange(e.target.value)}
                    required
                  />
                </FieldContent>
              </Field>

              <Field>
                <FieldLabel htmlFor="edit-slug">Slug *</FieldLabel>
                <FieldContent>
                  <Input
                    id="edit-slug"
                    value={formData.slug}
                    disabled
                    className="bg-muted font-mono"
                  />
                </FieldContent>
              </Field>
            </div>

            <Field>
              <FieldLabel htmlFor="edit-description">Description</FieldLabel>
              <FieldContent>
                <Textarea
                  id="edit-description"
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  rows={2}
                  className="min-h-20 max-h-48 resize-y break-all whitespace-pre-wrap"
                />
              </FieldContent>
            </Field>

            <Field>
              <FieldLabel>Plugin Type *</FieldLabel>
              <FieldContent>
                <div className="grid grid-cols-2 gap-3">
                  {pluginTypes.map((type) => (
                    <div
                      key={type.id}
                      onClick={() => setFormData((prev) => ({ ...prev, plugin_type: type.id }))}
                      className={cn(
                        "relative flex flex-col gap-2 p-3 rounded-lg border-2 cursor-pointer transition-all hover:bg-muted/50",
                        formData.plugin_type === type.id
                          ? "border-primary bg-primary/5"
                          : "border-muted"
                      )}
                    >
                      <div className="flex items-center gap-2">
                        <div
                          className={cn(
                            "p-1.5 rounded-md",
                            formData.plugin_type === type.id
                              ? "bg-primary text-primary-foreground"
                              : "bg-muted text-muted-foreground"
                          )}
                        >
                          <type.icon className="h-4 w-4" />
                        </div>
                        <span className="font-semibold text-sm">{type.title}</span>
                      </div>
                      <p className="text-[11px] text-muted-foreground leading-tight">
                        {type.description}
                      </p>
                    </div>
                  ))}
                </div>
              </FieldContent>
            </Field>

            <Field>
              <FieldLabel htmlFor="edit-image">Container Image *</FieldLabel>
              <FieldContent>
                <Input
                  id="edit-image"
                  value={formData.image}
                  onChange={(e) => setFormData({ ...formData, image: e.target.value })}
                  required
                />
              </FieldContent>
            </Field>

            <div className="grid grid-cols-2 gap-4">
              <Field>
                <FieldLabel htmlFor="edit-registry_username">Registry Username</FieldLabel>
                <FieldContent>
                  <Input
                    id="edit-registry_username"
                    value={formData.registry_username}
                    onChange={(e) => setFormData({ ...formData, registry_username: e.target.value })}
                  />
                </FieldContent>
              </Field>

              <Field>
                <FieldLabel htmlFor="edit-registry_password">Registry Password</FieldLabel>
                <FieldContent>
                  <Input
                    id="edit-registry_password"
                    type="password"
                    autoComplete="new-password"
                    placeholder="(leave blank to keep existing)"
                    value={formData.registry_password}
                    onChange={(e) => setFormData({ ...formData, registry_password: e.target.value })}
                  />
                </FieldContent>
              </Field>
            </div>

            <Field>
              <FieldLabel htmlFor="edit-command">Command</FieldLabel>
              <FieldContent>
                <Textarea
                  id="edit-command"
                  placeholder="echo hello"
                  value={formData.command}
                  onChange={(e) => setFormData({ ...formData, command: e.target.value })}
                  rows={3}
                  className="min-h-20 max-h-48 resize-y break-all whitespace-pre-wrap"
                />
              </FieldContent>
            </Field>

            <Field>
              <FieldLabel>Environment Variables</FieldLabel>
              <FieldContent>
                <KeyValueInput
                  value={envVars}
                  onChange={setEnvVars}
                />
              </FieldContent>
            </Field>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={updateMutation.isPending}>
              {updateMutation.isPending ? "Updating..." : "Update Plugin"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
