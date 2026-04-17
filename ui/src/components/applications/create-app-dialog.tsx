import { useMutation, useQueryClient } from "@tanstack/react-query"
import { Database, HardDriveDownload, Key, Layers } from "lucide-react"
import * as React from "react"
import { toast as sonnerToast } from "sonner"

import { appsApi, type App, type AppCreateRequest } from "@/api/apps"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Combobox, ComboboxContent, ComboboxInput, ComboboxItem, ComboboxList } from "@/components/ui/combobox"
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
import { getImagePullPolicyLabel, IMAGE_PULL_POLICY_OPTIONS } from "@/lib/image-pull-policy-options"
import { cn } from "@/lib/utils"
import { useProjectStore } from "@/stores/project"
import type { AxiosError } from "axios"

import { InputGroup, InputGroupAddon, InputGroupInput } from "@/components/ui/input-group"
import { Item, ItemContent, ItemDescription, ItemTitle } from "@/components/ui/item"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { deriveImageDefaults, toNameSlug } from "./create-app-dialog.utils"

interface CreateAppDialogProps {
  open?: boolean
  onOpenChange?: (open: boolean) => void
  onSuccess?: (app: App) => void
  onClose?: () => void
}

type CreateAppFormData = {
  name: string
  slug: string
  app_type: "Deployment" | "StatefulSet"
  container_image: string
  image_pull_policy: string
  description: string
  deploy: boolean
  registry_username: string
  registry_password: string
}

const INITIAL_FORM_DATA: CreateAppFormData = {
  name: "",
  slug: "",
  app_type: "Deployment",
  container_image: "",
  image_pull_policy: "IfNotPresent",
  description: "",
  deploy: false,
  registry_username: "",
  registry_password: "",
}

export function CreateAppDialog({
  open: controlledOpen,
  onOpenChange: setControlledOpen,
  onSuccess,
  onClose,
}: CreateAppDialogProps) {
  const [internalOpen, setInternalOpen] = React.useState(false)
  const open = controlledOpen !== undefined ? controlledOpen : internalOpen
  const setOpen = setControlledOpen || setInternalOpen

  const queryClient = useQueryClient()
  const { activeEnvId } = useProjectStore()
  const [showRegistryCredentials, setShowRegistryCredentials] = React.useState(false)
  const [showPullPolicy, setShowPullPolicy] = React.useState(false)
  const [hasEditedName, setHasEditedName] = React.useState(false)
  const [hasEditedSlug, setHasEditedSlug] = React.useState(false)
  const [hasEditedAppType, setHasEditedAppType] = React.useState(false)
  const [hasEditedPullPolicy, setHasEditedPullPolicy] = React.useState(false)

  const [errors, setErrors] = React.useState<{
    name?: string
    slug?: string
    container_image?: string
  }>({})

  const [formData, setFormData] = React.useState<CreateAppFormData>({ ...INITIAL_FORM_DATA })

  const resetForm = React.useCallback(() => {
    setFormData({ ...INITIAL_FORM_DATA })
    setErrors({})
    setShowRegistryCredentials(false)
    setShowPullPolicy(false)
    setHasEditedName(false)
    setHasEditedSlug(false)
    setHasEditedAppType(false)
    setHasEditedPullPolicy(false)
  }, [])

  const mutation = useMutation<App, AxiosError<{ error: string }>, CreateAppFormData>({
    mutationFn: (data) => {
      const payload: AppCreateRequest = {
        name: data.name,
        slug: data.slug,
        app_type: data.app_type,
        container_image: data.container_image,
        image_pull_policy: data.image_pull_policy,
        registry_username: data.registry_username,
        registry_password: data.registry_password,
        replicas: 1,
        request_cpu: 100,
        request_memory: 128,
        limit_cpu: 1000,
        limit_memory: 512,
        description: data.description,
        deploy: data.deploy,
        seed_image_metadata: true,
      }

      return appsApi.create(activeEnvId!, payload)
    },
    onSuccess: (app) => {
      queryClient.invalidateQueries({ queryKey: ['apps', activeEnvId] })
      sonnerToast.success("Application deployed successfully")
      onSuccess?.(app)
      setOpen(false)
      onClose?.()
      resetForm()
    },
    onError: (err) => {
      const errMsg = err.response?.data?.error || "Failed to create application"
      sonnerToast.error("Error", { description: errMsg })
    }
  })

  const handleImageChange = (container_image: string) => {
    const defaults = deriveImageDefaults(container_image)

    setFormData((prev) => ({
      ...prev,
      container_image,
      name: hasEditedName ? prev.name : defaults.name,
      slug: hasEditedSlug ? prev.slug : defaults.slug,
      app_type: hasEditedAppType
        ? prev.app_type
        : (defaults.isStateful ? "StatefulSet" : "Deployment"),
      image_pull_policy: hasEditedPullPolicy
        ? prev.image_pull_policy
        : derivePullPolicy(container_image),
    }))
  }

  const derivePullPolicy = (image: string): string => {
    const tag = image.split(":").pop()?.split("@")[0] ?? ""
    return tag === "latest" || !image.includes(":") ? "Always" : "IfNotPresent"
  }

  const validateForm = () => {
    const newErrors: typeof errors = {}

    if (!formData.name.trim()) {
      newErrors.name = "Name is required"
    }

    if (!formData.slug.trim()) {
      newErrors.slug = "Slug is required"
    }

    if (!formData.container_image.trim()) {
      newErrors.container_image = "Container image is required"
    }

    if (formData.registry_password.trim() && !formData.registry_username.trim()) {
      const errMsg = "Registry username is required when password is provided"
      sonnerToast.error("Error", { description: errMsg })
    }

    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const handleSubmit = async (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault()
    if (!validateForm()) return
    mutation.mutate(formData)
  }

  const appTypes: Array<{
    id: CreateAppFormData["app_type"]
    title: string
    description: string
    icon: typeof Layers
  }> = [
      {
        id: "Deployment",
        title: "Deployment",
        description: "Best for stateless applications that can be easily scaled and updated.",
        icon: Layers,
      },
      {
        id: "StatefulSet",
        title: "StatefulSet",
        description: "Best for stateful applications like databases that require stable storage and network identity.",
        icon: Database,
      },
    ]

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="sm:max-w-160 max-h-[90vh] overflow-y-auto">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Create Application</DialogTitle>
            <DialogDescription>
              Deploy a new containerized application to this environment.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            <Field>
              <FieldLabel htmlFor="container-image">Container Image *</FieldLabel>
              <FieldContent>
                <InputGroup>
                  <InputGroupInput id="container-image"
                    name="container_image"
                    placeholder="Enter or paste an image URL, e.g. nginx:latest or ghcr.io/org/app:1.0.0"
                    value={formData.container_image}
                    onChange={(e) => handleImageChange(e.target.value)}
                    aria-invalid={!!errors.container_image} />
                  <InputGroupAddon align="inline-end">
                    <Tooltip>
                      <TooltipTrigger
                        delay={200}
                        render={
                          <Button
                            type="button"
                            variant={showPullPolicy ? "default" : "ghost"}
                            size="icon-sm"
                            aria-label="Pull policy"
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
              {errors.container_image && <FieldError><span className="text-destructive text-xs">{errors.container_image}</span></FieldError>}
            </Field>

            {showPullPolicy && (<Field>
              <FieldLabel htmlFor="image-pull-policy">Pull Policy</FieldLabel>
              <FieldContent>
                <Combobox
                  value={formData.image_pull_policy}
                  onValueChange={(value) => {
                    setHasEditedPullPolicy(true)
                    setFormData((prev) => ({ ...prev, image_pull_policy: value ?? "IfNotPresent" }))
                  }}
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
                  <FieldLabel htmlFor="registry-username">Registry Username</FieldLabel>
                  <FieldContent>
                    <Input
                      id="registry-username"
                      name="registry_username"
                      placeholder="Registry Username"
                      value={formData.registry_username}
                      onChange={(e) => setFormData((prev) => ({ ...prev, registry_username: e.target.value }))}
                      autoComplete="off"
                    />
                  </FieldContent>
                </Field>
                <Field>
                  <FieldLabel htmlFor="registry-password">Registry Password</FieldLabel>
                  <FieldContent>
                    <Input
                      id="registry-password"
                      name="registry_password"
                      type="password"
                      autoComplete="new-password"
                      placeholder="Registry Password"
                      value={formData.registry_password}
                      onChange={(e) => setFormData((prev) => ({ ...prev, registry_password: e.target.value }))}
                    />
                  </FieldContent>
                </Field>
              </div>
            )}

            <div className="grid grid-cols-2 gap-4">
              <Field>
                <FieldLabel htmlFor="name">Name *</FieldLabel>
                <FieldContent>
                  <Input
                    id="name"
                    name="name"
                    placeholder="My App"
                    value={formData.name}
                    onChange={(e) => {
                      setHasEditedName(true)
                      const name = e.target.value
                      setFormData((prev) => ({
                        ...prev,
                        name,
                        slug: hasEditedSlug ? prev.slug : toNameSlug(name),
                      }))
                    }}
                    aria-invalid={!!errors.name}
                  />
                </FieldContent>
                {errors.name && <FieldError><span className="text-destructive text-xs">{errors.name}</span></FieldError>}
              </Field>

              <Field>
                <FieldLabel htmlFor="slug">Slug *</FieldLabel>
                <FieldContent>
                  <Input
                    id="slug"
                    name="slug"
                    placeholder="my-app"
                    value={formData.slug}
                    onChange={(e) => {
                      setHasEditedSlug(true)
                      setFormData((prev) => ({ ...prev, slug: e.target.value }))
                    }}
                    aria-invalid={!!errors.slug}
                  />
                </FieldContent>
                {errors.slug && <FieldError><span className="text-destructive text-xs">{errors.slug}</span></FieldError>}
              </Field>
            </div>

            <Field>
              <FieldLabel>Application Type *</FieldLabel>
              <FieldContent>
                <div className="grid grid-cols-2 gap-3">
                  {appTypes.map((type) => (
                    <button
                      type="button"
                      key={type.id}
                      data-app-type={type.id}
                      aria-pressed={formData.app_type === type.id}
                      onClick={() => {
                        setHasEditedAppType(true)
                        setFormData(prev => ({ ...prev, app_type: type.id }))
                      }}
                      className={cn(
                        "relative flex flex-col gap-2 p-3 rounded-lg border-2 cursor-pointer transition-all hover:bg-muted/50",
                        formData.app_type === type.id
                          ? "border-primary bg-primary/5"
                          : "border-muted"
                      )}
                    >
                      <div className="flex items-center gap-2">
                        <div className={cn(
                          "p-1.5 rounded-md",
                          formData.app_type === type.id ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground"
                        )}>
                          <type.icon className="h-4 w-4" />
                        </div>
                        <span className="font-semibold text-sm">{type.title}</span>
                      </div>
                      <p className="text-[11px] text-muted-foreground leading-tight">
                        {type.description}
                      </p>
                    </button>
                  ))}
                </div>
              </FieldContent>
            </Field>

            <Field>
              <FieldLabel htmlFor="description">Description</FieldLabel>
              <FieldContent>
                <Textarea
                  id="description"
                  placeholder="Brief description of this application..."
                  className="min-h-20 max-h-48 resize-y break-all whitespace-pre-wrap"
                  value={formData.description}
                  onChange={(e) => setFormData((prev) => ({ ...prev, description: e.target.value }))}
                />
              </FieldContent>
            </Field>
          </div>

          <DialogFooter className="sm:justify-between">
            <div className="flex items-center gap-2">
              <Checkbox
                id="deploy"
                checked={formData.deploy}
                onCheckedChange={(checked) => setFormData((prev) => ({ ...prev, deploy: checked === true }))}
              />
              <label htmlFor="deploy" className="cursor-pointer">
                Create and deploy application
              </label>
            </div>
            <div className="flex gap-2">
              <Button type="button" variant="outline" onClick={() => setOpen(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={mutation.isPending}>
                {mutation.isPending ? "Creating..." : "Create"}
              </Button>
            </div>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

export default CreateAppDialog
