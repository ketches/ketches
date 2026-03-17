import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import * as React from "react"
import { toast } from "sonner"

import { clustersApi, type Cluster } from "@/api/clusters"
import { envsApi, type EnvNamespaceAvailabilityResponse } from "@/api/envs"
import { projectsApi, type Project } from "@/api/projects"
import { Button } from "@/components/ui/button"
import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox"
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


import { Item, ItemContent, ItemDescription, ItemTitle } from "@/components/ui/item"
import { Textarea } from "@/components/ui/textarea"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { useProjectStore } from "@/stores/project"
import type { AxiosError } from "axios"
import { InfoIcon } from "lucide-react"

const KUBERNETES_NAMESPACE_MAX_LENGTH = 63
const KUBERNETES_NAMESPACE_PATTERN = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/

function buildDefaultNamespace(projectSlug: string, envSlug: string) {
  if (!projectSlug || !envSlug) {
    return ""
  }

  const raw = `${projectSlug}-${envSlug}`
    .toLowerCase()
    .replace(/[^a-z0-9-]/g, "-")
    .replace(/-+/g, "-")
    .replace(/^-|-$/g, "")

  if (!raw) {
    return ""
  }

  return raw.slice(0, KUBERNETES_NAMESPACE_MAX_LENGTH).replace(/-+$/g, "")
}

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
    namespace?: string
    cluster_id?: string
    global?: string
  }>({})
  const [namespaceEdited, setNamespaceEdited] = React.useState(false)

  const [formData, setFormData] = React.useState({
    name: "",
    slug: "",
    namespace: "",
    cluster_id: "",
    description: "",
  })

  const { data: clusters = [] } = useQuery<Cluster[]>({
    queryKey: ['clusters-public'],
    queryFn: clustersApi.listPublic,
    enabled: open,
  })

  const { data: project } = useQuery<Project>({
    queryKey: ["project", activeProjectId],
    queryFn: () => projectsApi.get(activeProjectId!),
    enabled: open && !!activeProjectId,
  })

  const defaultNamespace = React.useMemo(
    () => buildDefaultNamespace(project?.slug || "", formData.slug),
    [project?.slug, formData.slug],
  )

  const shouldShowNamespaceField = !!formData.cluster_id.trim() && !!formData.slug.trim()

  React.useEffect(() => {
    if (!open) {
      setFormData({
        name: "",
        slug: "",
        namespace: "",
        cluster_id: "",
        description: "",
      })
      setErrors({})
      setNamespaceEdited(false)
      return
    }

    if (!shouldShowNamespaceField) {
      return
    }

    if (namespaceEdited) {
      return
    }

    setFormData((prev) => {
      if (prev.namespace === defaultNamespace) {
        return prev
      }

      return {
        ...prev,
        namespace: defaultNamespace,
      }
    })
  }, [defaultNamespace, namespaceEdited, open, shouldShowNamespaceField])

  const localNamespaceValidationMessage = React.useMemo(() => {
    if (!shouldShowNamespaceField) {
      return undefined
    }

    const namespace = formData.namespace.trim()
    if (!namespace) {
      return undefined
    }

    if (namespace.length > KUBERNETES_NAMESPACE_MAX_LENGTH) {
      return `Namespace must be ${KUBERNETES_NAMESPACE_MAX_LENGTH} characters or fewer`
    }

    if (!KUBERNETES_NAMESPACE_PATTERN.test(namespace)) {
      return "Namespace must use lowercase letters, numbers, or hyphens, and start and end with a letter or number"
    }

    return undefined
  }, [formData.namespace, shouldShowNamespaceField])

  const namespaceAvailabilityQuery = useQuery<EnvNamespaceAvailabilityResponse>({
    queryKey: ["env-namespace-availability", activeProjectId, formData.cluster_id, formData.namespace.trim()],
    queryFn: () => envsApi.checkNamespaceAvailability(activeProjectId!, {
      cluster_id: formData.cluster_id,
      cluster_namespace: formData.namespace.trim(),
    }),
    enabled: open && !!activeProjectId && shouldShowNamespaceField && !!formData.namespace.trim() && !localNamespaceValidationMessage,
  })

  const mutation = useMutation({
    mutationFn: (data: any) => envsApi.create(activeProjectId!, {
      name: data.name,
      slug: data.slug,
      project_id: activeProjectId!,
      cluster_id: data.cluster_id,
      description: data.description,
      cluster_namespace: data.namespace.trim(),
    }),
    onSuccess: (env) => {
      queryClient.invalidateQueries({ queryKey: ['envs', activeProjectId] })
      toast.success("Environment created successfully")
      onSuccess?.(env)
      setOpen(false)
      onClose?.()
      setFormData({ name: "", slug: "", namespace: "", cluster_id: "", description: "" })
      setErrors({})
      setNamespaceEdited(false)
    },
    onError: (err: AxiosError<{ error: string }>) => {
      const errMsg = err.response?.data?.error || "Failed to create environment"
      setErrors({ global: errMsg })
      toast.error("Error", { description: errMsg })
    }
  })

  const handleClusterChange = (clusterId: string) => {
    setNamespaceEdited(false)
    setFormData((prev) => ({
      ...prev,
      cluster_id: clusterId,
      namespace: prev.slug.trim()
        ? buildDefaultNamespace(project?.slug || "", prev.slug)
        : prev.namespace,
    }))
  }

  const handleNameChange = (name: string) => {
    const slug = name
      .toLowerCase()
      .replace(/[^a-z0-9\s-]/g, "")
      .replace(/\s+/g, "-")
      .replace(/-+/g, "-")
      .replace(/^-|-$/g, "")

    setFormData((prev) => ({
      ...prev,
      name,
      slug,
    }))
  }

  const validateNamespace = React.useCallback(() => {
    if (!shouldShowNamespaceField) {
      return undefined
    }

    const namespace = formData.namespace.trim()
    if (!namespace) {
      return "Namespace is required"
    }

    if (localNamespaceValidationMessage) {
      return localNamespaceValidationMessage
    }

    if (namespaceAvailabilityQuery.isFetching) {
      return "Checking namespace availability"
    }

    if (namespaceAvailabilityQuery.data && !namespaceAvailabilityQuery.data.available) {
      return namespaceAvailabilityQuery.data.message
    }

    if (namespaceAvailabilityQuery.error) {
      return "Failed to check namespace availability"
    }

    return undefined
  }, [formData.namespace, localNamespaceValidationMessage, namespaceAvailabilityQuery.data, namespaceAvailabilityQuery.error, namespaceAvailabilityQuery.isFetching, shouldShowNamespaceField])

  const namespaceStatus = React.useMemo(() => {
    if (!shouldShowNamespaceField) {
      return null
    }

    if (localNamespaceValidationMessage) {
      return {
        tone: "error" as const,
        text: localNamespaceValidationMessage,
      }
    }

    if (namespaceAvailabilityQuery.isFetching) {
      return {
        tone: "muted" as const,
        text: "Checking...",
      }
    }

    if (namespaceAvailabilityQuery.error) {
      return {
        tone: "error" as const,
        text: "Failed to check namespace availability",
      }
    }

    if (namespaceAvailabilityQuery.data) {
      return {
        tone: namespaceAvailabilityQuery.data.available ? "success" as const : "error" as const,
        text: namespaceAvailabilityQuery.data.message,
      }
    }

    return null
  }, [localNamespaceValidationMessage, namespaceAvailabilityQuery.data, namespaceAvailabilityQuery.error, namespaceAvailabilityQuery.isFetching, shouldShowNamespaceField])

  React.useEffect(() => {
    setErrors((prev) => {
      if (!prev.namespace) {
        return prev
      }

      return {
        ...prev,
        namespace: undefined,
      }
    })
  }, [formData.cluster_id, formData.namespace, formData.slug, shouldShowNamespaceField])

  const isNamespaceCreatable = React.useMemo(() => {
    if (!shouldShowNamespaceField) {
      return true
    }

    if (!formData.namespace.trim()) {
      return false
    }

    if (localNamespaceValidationMessage) {
      return false
    }

    return namespaceAvailabilityQuery.data?.available === true
  }, [formData.namespace, localNamespaceValidationMessage, namespaceAvailabilityQuery.data?.available, shouldShowNamespaceField])

  const validateForm = () => {
    const newErrors: typeof errors = {}

    if (!formData.name.trim()) {
      newErrors.name = "Name is required"
    }

    if (!formData.slug.trim()) {
      newErrors.slug = "Slug is required"
    }

    const namespaceError = validateNamespace()
    if (namespaceError) {
      newErrors.namespace = namespaceError
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
      <DialogContent className="sm:max-w-160">
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
            <div className="grid grid-cols-2 gap-4">
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
                    id="name"
                    name="name"
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
                    id="slug"
                    name="slug"
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

            </div>
            <Field>
              <FieldLabel>Cluster *</FieldLabel>
              <FieldContent>
                <Combobox
                  value={formData.cluster_id}
                  onValueChange={(value: string | null) => value && handleClusterChange(value)}
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
                          <ComboboxItem key={cluster.id} value={cluster.id} disabled={cluster.connection_status !== "connected"}>
                            <Item size="xs" className="p-0">
                              <ItemContent>
                                <ItemTitle className="whitespace-nowrap">
                                  {cluster.name}
                                </ItemTitle>
                                <ItemDescription>
                                  {cluster.slug}
                                </ItemDescription>
                              </ItemContent>
                            </Item>

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
            {shouldShowNamespaceField && (
              <Field>
                <FieldLabel htmlFor="namespace" className="flex items-center justify-between gap-3">
                  <span className="inline-flex items-center gap-1.5">
                    Namespace *
                    <Tooltip>
                      <TooltipTrigger tabIndex={-1} type="button">
                        <InfoIcon className="h-3.5 w-3.5" />
                      </TooltipTrigger>
                      <TooltipContent side="top" align="start" className="max-w-72">
                        <p className="text-xs">Use a valid Kubernetes namespace: lowercase letters, numbers, or hyphens, up to 63 characters.</p>
                      </TooltipContent>
                    </Tooltip>
                  </span>
                  {namespaceStatus && (
                    <span
                      className={
                        namespaceStatus.tone === "success"
                          ? "text-xs text-emerald-600"
                          : namespaceStatus.tone === "error"
                            ? "text-xs text-destructive"
                            : "text-xs text-muted-foreground"
                      }
                    >
                      {namespaceStatus.text}
                    </span>
                  )}
                </FieldLabel>
                <FieldContent>
                  <Input
                    id="namespace"
                    name="namespace"
                    placeholder="demo-project-production"
                    value={formData.namespace}
                    onChange={(e) => {
                      setNamespaceEdited(true)
                      setFormData((prev) => ({ ...prev, namespace: e.target.value }))
                    }}
                    aria-invalid={!!errors.namespace}
                  />
                </FieldContent>
                {errors.namespace && (
                  <FieldError>
                    <span className="text-destructive text-xs">{errors.namespace}</span>
                  </FieldError>
                )}
              </Field>
            )}

            <Field>
              <FieldLabel>Description</FieldLabel>
              <FieldContent>
                <Textarea
                  id="description"
                  name="description"
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
            <Button
              type="submit"
              disabled={mutation.isPending || namespaceAvailabilityQuery.isFetching || !isNamespaceCreatable}
            >
              {mutation.isPending ? "Creating..." : "Create"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog >
  )
}

export default CreateEnvironmentDialog
