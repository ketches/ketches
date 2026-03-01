import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import * as React from "react"
import { toast } from "sonner"

import { clustersApi, type Cluster } from "@/api/clusters"
import { envsApi } from "@/api/envs"
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
import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox"


import { Textarea } from "@/components/ui/textarea"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { useProjectStore } from "@/stores/project"
import type { AxiosError } from "axios"
import { InfoIcon } from "lucide-react"

interface CreateEnvironmentFormProps {
  open?: boolean
  onOpenChange?: (open: boolean) => void
  onSuccess?: (environment: any) => void
  onClose?: () => void
}

export function CreateEnvironmentDialog({
  open: controlledOpen,
  onOpenChange: setControlledOpen,
  onSuccess,
  onClose,
}: CreateEnvironmentFormProps) {
  const [internalOpen, setInternalOpen] = React.useState(false)
  const open = controlledOpen !== undefined ? controlledOpen : internalOpen
  const setOpen = setControlledOpen || setInternalOpen

  const queryClient = useQueryClient()
  const { activeProjectId } = useProjectStore()

  const [errors, setErrors] = React.useState<{
    name?: string
    slug?: string
    cluster_id?: string
    global?: string
  }>({})

  const [formData, setFormData] = React.useState({
    name: "",
    slug: "",
    cluster_id: "",
    description: "",
  })

  const { data: clusters = [] } = useQuery<Cluster[]>({
    queryKey: ['clusters-public'],
    queryFn: clustersApi.listPublic,
    enabled: open,
  })

  const mutation = useMutation({
    mutationFn: (data: any) => envsApi.create(activeProjectId!, {
      name: data.name,
      slug: data.slug,
      project_id: activeProjectId!,
      cluster_id: data.cluster_id,
      description: data.description,
      cluster_namespace: data.slug,
    }),
    onSuccess: (env) => {
      queryClient.invalidateQueries({ queryKey: ['envs', activeProjectId] })
      toast.success("Environment created successfully")
      onSuccess?.(env)
      setOpen(false)
      onClose?.()
      setFormData({ name: "", slug: "", cluster_id: "", description: "" })
      setErrors({})
    },
    onError: (err: AxiosError<{ error: string }>) => {
      const errMsg = err.response?.data?.error || "Failed to create environment"
      setErrors({ global: errMsg })
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
    const newErrors: typeof errors = {}

    if (!formData.name.trim()) {
      newErrors.name = "Name is required"
    }

    if (!formData.slug.trim()) {
      newErrors.slug = "Slug is required"
    }

    if (!formData.cluster_id.trim()) {
      newErrors.cluster_id = "Please select a cluster"
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
      <DialogContent className="sm:max-w-140">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Create Environment</DialogTitle>
            <DialogDescription>
              Create a new deployment environment for your project.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            {errors.global && (
              <div className="text-sm font-medium text-destructive text-center">
                {errors.global}
              </div>
            )}
            <Field>
              <FieldLabel>
                Name *
                <Tooltip>
                  <TooltipTrigger tabIndex={-1} type="button">
                    <InfoIcon className="h-3.5 w-3.5" />
                  </TooltipTrigger>
                  <TooltipContent side="top" align="start" className="max-w-64">
                    <p className="text-xs">2-50 characters.</p>
                  </TooltipContent>
                </Tooltip>
              </FieldLabel>
              <FieldContent>
                <Input
                  placeholder="Production"
                  value={formData.name}
                  onChange={(e) => handleNameChange(e.target.value)}
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
              <FieldLabel>
                Slug *
                <Tooltip>
                  <TooltipTrigger tabIndex={-1} type="button">
                    <InfoIcon className="h-3.5 w-3.5" />
                  </TooltipTrigger>
                  <TooltipContent side="top" align="start" className="max-w-64">
                    <p className="text-xs">3-32 characters.</p>
                  </TooltipContent>
                </Tooltip>
              </FieldLabel>
              <FieldContent>
                <Input
                  placeholder="production"
                  value={formData.slug}
                  onChange={(e) => setFormData((prev) => ({ ...prev, slug: e.target.value }))}
                  aria-invalid={!!errors.slug}
                />
              </FieldContent>
              {errors.slug && (
                <FieldError>
                  <span className="text-destructive text-xs">{errors.slug}</span>
                </FieldError>
              )}
            </Field>

            <Field>
              <FieldLabel>Cluster *</FieldLabel>
              <FieldContent>
                <Combobox
                  value={formData.cluster_id}
                  onValueChange={(value: string | null) => value && setFormData((prev) => ({ ...prev, cluster_id: value }))}
                  itemToStringLabel={(id) => (clusters as Cluster[]).find((c) => c.id === id)?.name ?? id ?? ""}
                >
                  <ComboboxInput placeholder="Select a cluster" aria-invalid={!!errors.cluster_id} />
                  <ComboboxContent>
                    <ComboboxList>
                      {(clusters as Cluster[]).length === 0 ? (
                        <ComboboxItem value="no-clusters" disabled>
                          No clusters available
                        </ComboboxItem>
                      ) : (
                        (clusters as Cluster[]).map((cluster) => (
                          <ComboboxItem key={cluster.id} value={cluster.id}>
                            {cluster.name}
                          </ComboboxItem>
                        ))
                      )}
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
              </FieldContent>
              {errors.cluster_id && (
                <FieldError>
                  <span className="text-destructive text-xs">{errors.cluster_id}</span>
                </FieldError>
              )}
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

export default CreateEnvironmentDialog
