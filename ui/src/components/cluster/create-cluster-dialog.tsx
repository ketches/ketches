import { useMutation, useQueryClient } from "@tanstack/react-query"
import { CheckCircle2, InfoIcon, Link2, Loader2, Upload } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { clustersApi } from "@/api/clusters"
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
import { Textarea } from "@/components/ui/textarea"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"

interface CreateClusterFormProps {
  open?: boolean
  onOpenChange?: (open: boolean) => void
  onSuccess?: () => void
  onClose?: () => void
}

export function CreateClusterDialog({
  open: controlledOpen,
  onOpenChange: setControlledOpen,
  onSuccess,
  onClose,
}: CreateClusterFormProps) {
  const [internalOpen, setInternalOpen] = React.useState(false)
  const open = controlledOpen !== undefined ? controlledOpen : internalOpen
  const setOpen = setControlledOpen || setInternalOpen
  const queryClient = useQueryClient()

  const [errors, setErrors] = React.useState<{
    name?: string
    slug?: string
    kubeConfig?: string
    gateway_ip?: string
  }>({})
  const [isTestingConnection, setIsTestingConnection] = React.useState(false)
  const [connectionStatus, setConnectionStatus] = React.useState<"idle" | "success" | "error">("idle")

  const [formData, setFormData] = React.useState({
    name: "",
    slug: "",
    kubeConfig: "",
    gateway_ip: "",
    description: "",
  })

  const createMutation = useMutation({
    mutationFn: (data: any) => clustersApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['clusters'] })
      toast.success("Cluster added successfully")
      setOpen(false)
      onSuccess?.()
      onClose?.()
      setFormData({ name: "", slug: "", kubeConfig: "", gateway_ip: "", description: "" })
      setErrors({})
      setConnectionStatus("idle")
    },
    onError: (error: any) => {
      toast.error("Error", {
        description: error.response?.data?.error || "Failed to add cluster",
      })
    },
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

  const extractGatewayIp = (kubeConfig: string) => {
    try {
      const config = JSON.parse(kubeConfig)
      const clusters = config.clusters || []
      if (clusters.length > 0) {
        const cluster = clusters[0]
        const server = cluster.cluster?.server || ""
        const match = server.match(/https?:\/\/([^:]+):/)
        if (match) {
          return match[1]
        }
      }
    } catch {
      const lines = kubeConfig.split("\n")
      for (const line of lines) {
        const serverMatch = line.match(/server:\s*https?:\/\/([^:]+):/)
        if (serverMatch) {
          return serverMatch[1]
        }
      }
    }
    return ""
  }

  const handleKubeConfigChange = (value: string) => {
    setFormData((prev) => ({ ...prev, kubeConfig: value }))
    const gatewayIp = extractGatewayIp(value)
    if (gatewayIp) {
      setFormData((prev) => ({ ...prev, gateway_ip: gatewayIp }))
    }
  }

  const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      const reader = new FileReader()
      reader.onload = (event) => {
        const content = event.target?.result as string
        handleKubeConfigChange(content)
      }
      reader.readAsText(file)
    }
  }

  const validateForm = () => {
    const newErrors: typeof errors = {}

    if (!formData.name.trim()) {
      newErrors.name = "Name is required"
    } else if (formData.name.length < 2) {
      newErrors.name = "Name must be at least 2 characters"
    } else if (formData.name.length > 50) {
      newErrors.name = "Name must be less than 50 characters"
    }

    if (!formData.slug.trim()) {
      newErrors.slug = "Slug is required"
    } else if (formData.slug.length < 3) {
      newErrors.slug = "Slug must be at least 3 characters"
    } else if (formData.slug.length > 32) {
      newErrors.slug = "Slug must be less than 32 characters"
    } else if (!/^[a-z][a-z0-9-]*[a-z0-9]$/.test(formData.slug)) {
      newErrors.slug = "Must start and end with a letter, and cannot contain consecutive hyphens"
    }

    if (!formData.kubeConfig.trim()) {
      newErrors.kubeConfig = "KubeConfig is required"
    }

    if (formData.gateway_ip && !/^[\d.]+$/.test(formData.gateway_ip)) {
      newErrors.gateway_ip = "Gateway IP format is incorrect"
    }

    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const testConnection = async () => {
    if (!formData.kubeConfig.trim()) {
      setErrors((prev) => ({ ...prev, kubeConfig: "Please fill in the KubeConfig" }))
      return
    }

    setIsTestingConnection(true)
    setConnectionStatus("idle")

    try {
      await clustersApi.ping({ kube_config: formData.kubeConfig })
      setConnectionStatus("success")
      toast.success("Connection test successful")
    } catch (error: any) {
      setConnectionStatus("error")
      toast.error("Connection Failed", {
        description: error.response?.data?.error || "Unable to connect to cluster",
      })
    } finally {
      setIsTestingConnection(false)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!validateForm()) {
      return
    }

    createMutation.mutate({
      slug: formData.slug,
      name: formData.name,
      description: formData.description,
      kube_config: formData.kubeConfig,
      gateway_ip: formData.gateway_ip,
    })
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="sm:max-w-140 max-h-[90vh] overflow-y-auto">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Add Cluster</DialogTitle>
            <DialogDescription>
              Connect a new Kubernetes cluster to your platform.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            <Field>
              <FieldLabel>
                Name *
                <Tooltip>
                  <TooltipTrigger
                    tabIndex={-1}
                    render={
                      <button type="button" className="text-muted-foreground hover:text-foreground transition-colors outline-none">
                        <InfoIcon className="h-3.5 w-3.5" />
                      </button>
                    }
                  />
                  <TooltipContent side="top" align="start" className="max-w-64">
                    <p className="text-xs">2-50 characters.</p>
                  </TooltipContent>
                </Tooltip>
              </FieldLabel>
              <FieldContent>
                <Input
                  placeholder="My Cluster"
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
                  <TooltipTrigger tabIndex={-1}>
                    <InfoIcon className="h-3.5 w-3.5" />
                  </TooltipTrigger>
                  <TooltipContent side="top" align="start" className="max-w-64">
                    <p className="text-xs">3-32 characters.</p>
                    <p className="text-xs mt-1">Only lowercase letters, numbers, and hyphens.</p>
                  </TooltipContent>
                </Tooltip>
              </FieldLabel>
              <FieldContent>
                <Input
                  placeholder="my-cluster"
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
              <div className="flex items-center justify-between">
                <FieldLabel>
                  KubeConfig *
                  <Tooltip>
                    <TooltipTrigger tabIndex={-1}>
                      <InfoIcon className="h-3.5 w-3.5" />
                    </TooltipTrigger>
                    <TooltipContent side="top" align="start" className="max-w-64">
                      <p className="text-xs">Kubernetes config file content (JSON or YAML).</p>
                      <p className="text-xs mt-1">Supports upload JSON or YAML format file.</p>
                      <p className="text-xs mt-1">Auto-extracts Gateway IP from apiServer.</p>
                    </TooltipContent>
                  </Tooltip>
                </FieldLabel>
                <label className="cursor-pointer inline-flex items-center gap-1 px-2 py-1 bg-muted hover:bg-muted/80 rounded text-xs transition-colors shrink-0">
                  <Upload className="h-3 w-3" />
                  Upload
                  <input
                    type="file"
                    accept=".json,.yaml,.yml"
                    className="hidden"
                    onChange={handleFileUpload}
                  />
                </label>
              </div>
              <FieldContent>
                <Textarea
                  placeholder='Paste or upload your KubeConfig here...'
                  value={formData.kubeConfig}
                  onChange={(e) => handleKubeConfigChange(e.target.value)}
                  className="min-h-20 max-h-48 resize-y break-all whitespace-pre-wrap font-mono"
                  aria-invalid={!!errors.kubeConfig}
                />
              </FieldContent>
              {errors.kubeConfig && (
                <FieldError>
                  <span className="text-destructive text-xs">{errors.kubeConfig}</span>
                </FieldError>
              )}
            </Field>

            <Field>
              <FieldLabel>
                Gateway IP
                <Tooltip>
                  <TooltipTrigger
                    tabIndex={-1}
                    render={
                      <button type="button" className="text-muted-foreground hover:text-foreground transition-colors outline-none">
                        <InfoIcon className="h-3.5 w-3.5" />
                      </button>
                    }
                  />
                  <TooltipContent side="top" align="start" className="max-w-64">
                    <p className="text-xs">Auto-extracted from KubeConfig apiServer.</p>
                    <p className="text-xs mt-1">Can be manually edited.</p>
                  </TooltipContent>
                </Tooltip>
              </FieldLabel>
              <FieldContent>
                <Input
                  placeholder="192.168.1.1"
                  value={formData.gateway_ip}
                  onChange={(e) => setFormData((prev) => ({ ...prev, gateway_ip: e.target.value }))}
                  aria-invalid={!!errors.gateway_ip}
                />
              </FieldContent>
              {errors.gateway_ip && (
                <FieldError>
                  <span className="text-destructive text-xs">{errors.gateway_ip}</span>
                </FieldError>
              )}
            </Field>

            <Field>
              <FieldLabel>
                Description
              </FieldLabel>
              <FieldContent>
                <Textarea
                  placeholder="Brief description of the cluster."
                  value={formData.description}
                  onChange={(e) => setFormData((prev) => ({ ...prev, description: e.target.value }))}
                  className="min-h-20 max-h-48 resize-y break-all whitespace-pre-wrap"
                />
              </FieldContent>
            </Field>
          </div>

          <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-between">
            <Button
              type="button"
              variant="outline"
              onClick={testConnection}
              disabled={isTestingConnection || !formData.kubeConfig.trim()}
              className="gap-2 justify-start"
            >
              {isTestingConnection ? (
                <>
                  <Loader2 className="animate-spin" />
                  Testing...
                </>
              ) : connectionStatus === "success" ? (
                <>
                  <CheckCircle2 className="text-green-500" />
                  Connected
                </>
              ) : (
                <>
                  <Link2 />
                  Test Connection
                </>
              )}
            </Button>
            <div className="flex gap-2 sm:justify-end">
              <Button type="button" variant="outline" onClick={() => setOpen(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={createMutation.isPending}>
                {createMutation.isPending ? "Adding..." : "Add Cluster"}
              </Button>
            </div>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

export default CreateClusterDialog
