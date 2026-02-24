import { useMutation, useQueryClient } from "@tanstack/react-query"
import { Database, Layers } from "lucide-react"
import * as React from "react"
import { toast as sonnerToast } from "sonner"

import { appsApi } from "@/api/apps"
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
import { cn } from "@/lib/utils"
import { useProjectStore } from "@/stores/project"

interface CreateAppDialogProps {
  open?: boolean
  onOpenChange?: (open: boolean) => void
  onSuccess?: (app: any) => void
  onClose?: () => void
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

  const [errors, setErrors] = React.useState<{
    name?: string
    slug?: string
    container_image?: string
    global?: string
  }>({})

  const [formData, setFormData] = React.useState({
    name: "",
    slug: "",
    app_type: "Deployment",
    container_image: "",
    description: "",
    deploy: true,
    registry_username: "",
    registry_password: "",
  })

  const mutation = useMutation({
    mutationFn: (data: any) => appsApi.create(activeEnvId!, {
      name: data.name,
      slug: data.slug,
      app_type: data.app_type,
      container_image: data.container_image,
      registry_username: data.registry_username,
      registry_password: data.registry_password,
      replicas: 1,
      request_cpu: 100,
      request_memory: 128,
      limit_cpu: 1000,
      limit_memory: 512,
      description: data.description,
      deploy: data.deploy,
    }),
    onSuccess: (app) => {
      queryClient.invalidateQueries({ queryKey: ['apps', activeEnvId] })
      sonnerToast.success("Application deployed successfully")
      onSuccess?.(app)
      setOpen(false)
      onClose?.()
      setFormData({ name: "", slug: "", app_type: "Deployment", container_image: "", description: "", deploy: true, registry_username: "", registry_password: "" })
      setErrors({})
    },
    onError: (err: any) => {
      const errMsg = err.response?.data?.error || "Failed to create application"
      setErrors({ global: errMsg })
      sonnerToast.error("Error", { description: errMsg })
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

    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!validateForm()) return
    mutation.mutate(formData)
  }

  const appTypes = [
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
            {errors.global && (
              <div className="text-sm font-medium text-destructive text-center">
                {errors.global}
              </div>
            )}

            <div className="grid grid-cols-2 gap-4">
              <Field>
                <FieldLabel>Name *</FieldLabel>
                <FieldContent>
                  <Input
                    placeholder="My App"
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
                    placeholder="my-app"
                    value={formData.slug}
                    onChange={(e) => setFormData((prev) => ({ ...prev, slug: e.target.value }))}
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
                    <div
                      key={type.id}
                      onClick={() => setFormData(prev => ({ ...prev, app_type: type.id }))}
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
                    </div>
                  ))}
                </div>
              </FieldContent>
            </Field>

            <Field>
              <FieldLabel>Container Image *</FieldLabel>
              <FieldContent>
                <Input
                  placeholder="nginx:latest"
                  value={formData.container_image}
                  onChange={(e) => setFormData((prev) => ({ ...prev, container_image: e.target.value }))}
                  aria-invalid={!!errors.container_image}
                />
              </FieldContent>
              {errors.container_image && <FieldError><span className="text-destructive text-xs">{errors.container_image}</span></FieldError>}
            </Field>

            <div className="grid grid-cols-2 gap-4">
              <Field>
                <FieldLabel>Registry Username</FieldLabel>
                <FieldContent>
                  <Input
                    placeholder="Registry Username"
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
                    placeholder="Registry Password"
                    value={formData.registry_password}
                    onChange={(e) => setFormData((prev) => ({ ...prev, registry_password: e.target.value }))}
                  />
                </FieldContent>
              </Field>
            </div>

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
                {mutation.isPending ? "Deploying..." : "Deploy"}
              </Button>
            </div>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

export default CreateAppDialog
