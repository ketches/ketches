import { useMutation, useQueryClient } from "@tanstack/react-query"
import { HardDriveDownload, Key, Layers2, Zap } from "lucide-react"
import { useState } from "react"
import { toast } from "sonner"

import { pluginsApi, type CreatePluginRequest } from "@/api/plugins"
import { KeyValueInput, type KeyValuePair } from "@/components/shared/key-value-input"
import { Button } from "@/components/ui/button"
import { Combobox, ComboboxContent, ComboboxInput, ComboboxItem, ComboboxList } from "@/components/ui/combobox"
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
import { InputGroup, InputGroupAddon, InputGroupInput } from "@/components/ui/input-group"
import { Item, ItemContent, ItemDescription, ItemTitle } from "@/components/ui/item"
import { Textarea } from "@/components/ui/textarea"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { getImagePullPolicyLabel, IMAGE_PULL_POLICY_OPTIONS } from "@/lib/image-pull-policy-options"
import { cn } from "@/lib/utils"
import type { AxiosError } from "axios"

interface CreatePluginDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectId: string
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

export function CreatePluginDialog({ open, onOpenChange, projectId }: CreatePluginDialogProps) {
  const queryClient = useQueryClient()
  const [formData, setFormData] = useState({
    name: "",
    slug: "",
    description: "",
    image: "",
    image_pull_policy: "IfNotPresent",
    registry_username: "",
    registry_password: "",
    command: "",
    plugin_type: "init" as "init" | "sidecar"
  })
  const [envVars, setEnvVars] = useState<KeyValuePair[]>([])
  const [showRegistryCredentials, setShowRegistryCredentials] = useState(false)
  const [showPullPolicy, setShowPullPolicy] = useState(false)
  const createMutation = useMutation({
    mutationFn: (data: CreatePluginRequest) => pluginsApi.createPlugin(projectId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['plugins', projectId] })
      toast.success("Plugin created successfully")
      onOpenChange(false)
      resetForm()
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error("Failed to create plugin", {
        description: err.response?.data?.error || "An unknown error occurred"
      })
    }
  })

  const resetForm = () => {
    setFormData({
      name: "",
      slug: "",
      description: "",
      image: "",
      image_pull_policy: "",
      registry_username: "",
      registry_password: "",
      command: "",
      plugin_type: "init"
    })
    setEnvVars([])
    setShowRegistryCredentials(false)
    setShowPullPolicy(false)
  }

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

    if (formData.registry_password.trim() && !formData.registry_username.trim()) {
      toast.error("Registry username is required when password is provided")
      return false
    }

    return true
  }


  const handleSubmit = (e: React.SubmitEvent) => {
    e.preventDefault()

    if (!validateRequiredFields()) return

    const payload = {
      project_id: projectId,
      slug: formData.slug,
      name: formData.name,
      description: formData.description,
      image: formData.image,
      image_pull_policy: formData.image_pull_policy || undefined,
      registry_username: formData.registry_username || undefined,
      registry_password: formData.registry_password || undefined,
      command: formData.command || undefined,
      env_vars: envVars,
      plugin_type: formData.plugin_type
    }

    createMutation.mutate(payload)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-160 max-h-[90vh] overflow-y-auto">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Create Plugin</DialogTitle>
            <DialogDescription>
              Create a new plugin. Plugins can be installed as init containers or sidecars.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            <div className="grid grid-cols-2 gap-4">
              <Field>
                <FieldLabel htmlFor="name">Name *</FieldLabel>
                <FieldContent>
                  <Input
                    id="name"
                    placeholder="Database Migration"
                    value={formData.name}
                    onChange={(e) => handleNameChange(e.target.value)}
                    required
                  />
                </FieldContent>
              </Field>

              <Field>
                <FieldLabel htmlFor="slug">Slug *</FieldLabel>
                <FieldContent>
                  <Input
                    id="slug"
                    placeholder="database-migration"
                    value={formData.slug}
                    onChange={(e) => setFormData({ ...formData, slug: e.target.value })}
                    required
                  />
                </FieldContent>
              </Field>
            </div>

            <Field>
              <FieldLabel htmlFor="description">Description</FieldLabel>
              <FieldContent>
                <Textarea
                  id="description"
                  placeholder="Runs database migrations before app starts"
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
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
              <FieldLabel htmlFor="image">Container Image *</FieldLabel>
              <FieldContent>
                <InputGroup>
                  <InputGroupInput
                    id="image"
                    name="image"
                    placeholder="docker.io/library/busybox:latest"
                    value={formData.image}
                    onChange={(e) => setFormData({ ...formData, image: e.target.value })}
                    required
                  />
                  <InputGroupAddon align="inline-end">
                    <Tooltip>
                      <TooltipTrigger
                        delay={200}
                        render={
                          <Button
                            type="button"
                            variant={showPullPolicy ? "default" : "ghost"}
                            size="icon-sm"
                            aria-label="Pull Policy"
                            aria-pressed={showPullPolicy}
                            onClick={() => setShowPullPolicy((prev) => !prev)}
                            className="ml-auto"
                          />
                        }
                      >
                        <HardDriveDownload />
                      </TooltipTrigger>
                      <TooltipContent>
                        <p>Pull Policy</p>
                      </TooltipContent>
                    </Tooltip>
                    <Tooltip>
                      <TooltipTrigger
                        delay={200}
                        render={
                          <Button
                            type="button"
                            variant={showRegistryCredentials ? "default" : "ghost"}
                            size="icon-sm"
                            aria-label="Registry credentials"
                            aria-pressed={showRegistryCredentials}
                            onClick={() => setShowRegistryCredentials((prev) => !prev)}
                            className="ml-auto"
                          />
                        }
                      >
                        <Key />
                      </TooltipTrigger>
                      <TooltipContent>
                        <p>Registry Credentials</p>
                      </TooltipContent>
                    </Tooltip>
                  </InputGroupAddon>
                </InputGroup>
              </FieldContent>
            </Field>

            {showPullPolicy && (
              <Field>
                <FieldLabel htmlFor="image-pull-policy">Pull Policy</FieldLabel>
                <FieldContent>
                  <Combobox
                    value={formData.image_pull_policy}
                    onValueChange={(value) => setFormData((prev) => ({ ...prev, image_pull_policy: value ?? "IfNotPresent" }))}
                    itemToStringLabel={getImagePullPolicyLabel}
                  >
                    <ComboboxInput
                      id="image-pull-policy"
                      name="image_pull_policy"
                      value={formData.image_pull_policy}
                      readOnly
                      className="w-full cursor-pointer"
                    />
                    <ComboboxContent>
                      <ComboboxList>
                        {IMAGE_PULL_POLICY_OPTIONS.map((option) => (
                          <ComboboxItem key={option.value} value={option.value}>
                            <Item size="xs" className="p-0">
                              <ItemContent>
                                <ItemTitle className="whitespace-nowrap">
                                  {option.label}
                                </ItemTitle>
                                <ItemDescription>
                                  {option.description}
                                </ItemDescription>
                              </ItemContent>
                            </Item>
                          </ComboboxItem>
                        ))}
                      </ComboboxList>
                    </ComboboxContent>
                  </Combobox>
                </FieldContent>
              </Field>
            )}

            {showRegistryCredentials && (
              <div className="grid grid-cols-2 gap-4">
                <Field>
                  <FieldLabel htmlFor="registry_username">Registry Username</FieldLabel>
                  <FieldContent>
                    <Input
                      id="registry_username"
                      placeholder="(optional for private images)"
                      value={formData.registry_username}
                      onChange={(e) => setFormData({ ...formData, registry_username: e.target.value })}
                    />
                  </FieldContent>
                </Field>

                <Field>
                  <FieldLabel htmlFor="registry_password">Registry Password</FieldLabel>
                  <FieldContent>
                    <Input
                      id="registry_password"
                      type="password"
                      autoComplete="new-password"
                      placeholder="(optional for private images)"
                      value={formData.registry_password}
                      onChange={(e) => setFormData({ ...formData, registry_password: e.target.value })}
                    />
                  </FieldContent>
                </Field>
              </div>
            )}

            <Field>
              <FieldLabel htmlFor="command">Command</FieldLabel>
              <FieldContent>
                <Textarea
                  id="command"
                  placeholder='echo hello'
                  value={formData.command}
                  onChange={(e) => setFormData({ ...formData, command: e.target.value })}
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
            <Button type="submit" disabled={createMutation.isPending}>
              {createMutation.isPending ? "Creating..." : "Create Plugin"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
