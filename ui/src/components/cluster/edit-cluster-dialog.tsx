import { useMutation, useQueryClient } from "@tanstack/react-query"
import { CheckCircle2, InfoIcon, Link2, Loader2, Upload } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { clustersApi, type Cluster } from "@/api/clusters"
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

interface EditClusterDialogProps {
  open?: boolean
  onOpenChange?: (open: boolean) => void
  cluster: Cluster | null
  onSuccess?: () => void
}

type TabType = 'basic' | 'credentials'

export function EditClusterDialog({
  open: controlledOpen,
  onOpenChange: setControlledOpen,
  cluster,
  onSuccess,
}: EditClusterDialogProps) {
  const [internalOpen, setInternalOpen] = React.useState(false)
  const open = controlledOpen !== undefined ? controlledOpen : internalOpen
  const setOpen = setControlledOpen || setInternalOpen
  const queryClient = useQueryClient()
  const [activeTab, setActiveTab] = React.useState<TabType>('basic')

  const [errors, setErrors] = React.useState<{
    name?: string
    description?: string
    kubeConfig?: string
    gateway_ip?: string
  }>({})

  const [isTestingConnection, setIsTestingConnection] = React.useState(false)
  const [connectionStatus, setConnectionStatus] = React.useState<"idle" | "success" | "error">("idle")

  const [formData, setFormData] = React.useState({
    name: "",
    description: "",
    kubeConfig: "",
    gateway_ip: "",
  })

  React.useEffect(() => {
    if (cluster && open) {
      setFormData({
        name: cluster.name,
        description: cluster.description || "",
        kubeConfig: cluster.kube_config || "",
        gateway_ip: cluster.gateway_ip || "",
      })
      setErrors({})
      setConnectionStatus("idle")
      setActiveTab('basic')
    }
  }, [cluster, open])

  const updateBasicMutation = useMutation({
    mutationFn: (data: { name: string; description: string }) => {
      if (!cluster) throw new Error("No cluster selected")
      return clustersApi.update(cluster.id, data)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['clusters'] })
      toast.success("Cluster updated successfully")
      setOpen(false)
      onSuccess?.()
    },
    onError: (error: any) => {
      toast.error("Error", {
        description: error.response?.data?.error || "Failed to update cluster",
      })
    },
  })

  const updateCredentialsMutation = useMutation({
    mutationFn: (data: { kube_config: string; gateway_ip?: string }) => {
      if (!cluster) throw new Error("No cluster selected")
      return clustersApi.updateCredentials(cluster.id, data)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['clusters'] })
      queryClient.invalidateQueries({ queryKey: ['cluster', cluster?.id] })
      toast.success("Cluster credentials updated successfully")
      setOpen(false)
      onSuccess?.()
    },
    onError: (error: any) => {
      toast.error("Error", {
        description: error.response?.data?.error || "Failed to update cluster credentials",
      })
    },
  })

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

  const validateBasicForm = () => {
    const newErrors: typeof errors = {}

    if (!formData.name.trim()) {
      newErrors.name = "Name is required"
    } else if (formData.name.length < 2) {
      newErrors.name = "Name must be at least 2 characters"
    } else if (formData.name.length > 50) {
      newErrors.name = "Name must be less than 50 characters"
    }

    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const validateCredentialsForm = () => {
    const newErrors: typeof errors = {}

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

    if (activeTab === 'basic') {
      if (!validateBasicForm()) {
        return
      }
      updateBasicMutation.mutate({
        name: formData.name,
        description: formData.description,
      })
    } else {
      if (!validateCredentialsForm()) {
        return
      }

      updateCredentialsMutation.mutate({
        kube_config: formData.kubeConfig,
        gateway_ip: formData.gateway_ip || undefined,
      })
    }
  }

  const isPending = updateBasicMutation.isPending || updateCredentialsMutation.isPending

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="sm:max-w-140 max-h-[90vh] overflow-y-auto">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Edit Cluster</DialogTitle>
            <DialogDescription>
              Update cluster information and connection settings.
            </DialogDescription>
          </DialogHeader>

          <div className="flex gap-2 mt-4 border-b">
            <button
              type="button"
              onClick={() => setActiveTab('basic')}
              className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${activeTab === 'basic'
                ? 'border-primary text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground'
                }`}
            >
              Basic Info
            </button>
            <button
              type="button"
              onClick={() => setActiveTab('credentials')}
              className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${activeTab === 'credentials'
                ? 'border-primary text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground'
                }`}
            >
              Credentials
            </button>
          </div>

          <div className="grid gap-4 py-4">
            {activeTab === 'basic' ? (
              <>
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
                      onChange={(e) => setFormData((prev) => ({ ...prev, name: e.target.value }))}
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
                  <FieldLabel>Slug</FieldLabel>
                  <FieldContent>
                    <Input
                      value={cluster?.slug || ""}
                      disabled
                      className="bg-muted font-mono"
                    />
                  </FieldContent>
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
              </>
            ) : (
              <>
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
                      placeholder='Paste or upload your new KubeConfig here...'
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
              </>
            )}
          </div>

          <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-between">
            {activeTab === 'credentials' && (
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
                    <CheckCircle2 className=" text-green-500" />
                    Connected
                  </>
                ) : (
                  <>
                    <Link2 />
                    Test Connection
                  </>
                )}
              </Button>
            )}
            {activeTab === 'basic' && <div />}
            <div className="flex gap-2 sm:justify-end">
              <Button type="button" variant="outline" onClick={() => setOpen(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={isPending}>
                {isPending ? (
                  <>
                    <Loader2 className="h-4 w-4 animate-spin mr-2" />
                    Saving...
                  </>
                ) : (
                  "Save Changes"
                )}
              </Button>
            </div>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

export default EditClusterDialog
