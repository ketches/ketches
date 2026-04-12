import { AxiosError } from "axios"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { CheckCircle2, InfoIcon, Link2, Loader2, Upload } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { clustersApi, type Cluster, type UpdateClusterCredentialsRequest } from "@/api/clusters"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Field, FieldContent, FieldError, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"

interface EditClusterConnectionDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  cluster: Cluster
}

function extractGatewayHost(kubeConfig: string) {
  try {
    const parsed = JSON.parse(kubeConfig)
    const clusters = parsed.clusters || []
    if (clusters.length > 0) {
      const server = clusters[0]?.cluster?.server || ""
      const match = server.match(/https?:\/\/([^:/]+)(?::\d+)?/)
      if (match) {
        return match[1]
      }
    }
  } catch {
    const lines = kubeConfig.split("\n")
    for (const line of lines) {
      const serverMatch = line.match(/server:\s*https?:\/\/([^:/]+)(?::\d+)?/)
      if (serverMatch) {
        return serverMatch[1]
      }
    }
  }

  return ""
}

export function EditClusterConnectionDialog({
  open,
  onOpenChange,
  cluster,
}: EditClusterConnectionDialogProps) {
  const queryClient = useQueryClient()

  const [errors, setErrors] = React.useState<{
    kube_config?: string
    gateway_host?: string
  }>({})
  const [isTestingConnection, setIsTestingConnection] = React.useState(false)
  const [connectionStatus, setConnectionStatus] = React.useState<"idle" | "success" | "error">("idle")

  const [formData, setFormData] = React.useState({
    kube_config: "",
    gateway_host: "",
  })

  React.useEffect(() => {
    if (open) {
      setFormData({
        kube_config: "",
        gateway_host: cluster.gateway_host || "",
      })
      setErrors({})
      setConnectionStatus("idle")
    }
  }, [cluster, open])

  const updateMutation = useMutation({
    mutationFn: (data: UpdateClusterCredentialsRequest) => clustersApi.updateCredentials(cluster.id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["cluster", cluster.id] })
      toast.success("Cluster connection updated successfully")
      onOpenChange(false)
    },
    onError: (error: AxiosError<{ error: string }>) => {
      toast.error("Failed to update connection", {
        description: error.response?.data?.error || error.message,
      })
    },
  })

  const pingMutation = useMutation({
    mutationFn: (kube_config: string) => clustersApi.ping({ kube_config }),
    onSuccess: () => {
      setConnectionStatus("success")
      toast.success("Connection test successful")
    },
    onError: (error: AxiosError<{ error: string }>) => {
      setConnectionStatus("error")
      toast.error("Connection Failed", {
        description: error.response?.data?.error || "Unable to connect to cluster",
      })
    },
    onSettled: () => {
      setIsTestingConnection(false)
    },
  })

  const handleKubeConfigChange = (value: string) => {
    setFormData((prev) => ({ ...prev, kube_config: value }))
    const gatewayHost = extractGatewayHost(value)
    if (gatewayHost) {
      setFormData((prev) => ({ ...prev, gateway_host: gatewayHost }))
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

    if (formData.kube_config && formData.kube_config.trim().length < 10) {
      newErrors.kube_config = "Provided KubeConfig appears invalid"
    }

    if (formData.gateway_host && (formData.gateway_host.includes(" ") || formData.gateway_host.includes("://"))) {
      newErrors.gateway_host = "Must be a valid hostname or IP address without protocol"
    }

    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const handlePing = () => {
    if (!formData.kube_config.trim()) {
      setErrors((prev) => ({ ...prev, kube_config: "Please fill in the KubeConfig" }))
      return
    }

    setIsTestingConnection(true)
    setConnectionStatus("idle")
    pingMutation.mutate(formData.kube_config.trim())
  }

  const handleSubmit = (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault()

    if (!validateForm()) {
      return
    }

    const payload: UpdateClusterCredentialsRequest = {
      gateway_host: formData.gateway_host,
    }

    if (formData.kube_config.trim()) {
      payload.kube_config = formData.kube_config.trim()
    }

    updateMutation.mutate(payload)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-140 max-h-[90vh] overflow-y-auto">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Update KubeConfig</DialogTitle>
            <DialogDescription>
              Update the cluster KubeConfig and gateway host.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            <Field>
              <div className="flex items-center justify-between">
                <FieldLabel>
                  KubeConfig
                  <Tooltip>
                    <TooltipTrigger tabIndex={-1}>
                      <InfoIcon className="h-3.5 w-3.5" />
                    </TooltipTrigger>
                    <TooltipContent side="top" align="start" className="max-w-64">
                      <p className="text-xs">Kubernetes config file content (JSON or YAML).</p>
                      <p className="text-xs mt-1">Supports upload JSON or YAML format file.</p>
                      <p className="text-xs mt-1">Auto-extracts Gateway Host from apiServer.</p>
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
                  placeholder={cluster.has_kube_config ? "Leave blank to keep existing configuration..." : "Paste or upload your KubeConfig here..."}
                  value={formData.kube_config}
                  onChange={(e) => handleKubeConfigChange(e.target.value)}
                  className="min-h-20 max-h-48 resize-y break-all whitespace-pre-wrap font-mono"
                  aria-invalid={!!errors.kube_config}
                />
              </FieldContent>
              {errors.kube_config && (
                <FieldError>
                  <span className="text-destructive text-xs">{errors.kube_config}</span>
                </FieldError>
              )}
            </Field>

            <Field>
              <FieldLabel>
                Gateway Host
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
                  placeholder="e.g. 10.0.0.1 or gateway.example.com"
                  value={formData.gateway_host}
                  onChange={(e) => setFormData((prev) => ({ ...prev, gateway_host: e.target.value }))}
                  aria-invalid={!!errors.gateway_host}
                />
              </FieldContent>
              {errors.gateway_host && (
                <FieldError>
                  <span className="text-destructive text-xs">{errors.gateway_host}</span>
                </FieldError>
              )}
            </Field>

          </div>

          <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-between">
            <Button
              type="button"
              variant="outline"
              onClick={handlePing}
              disabled={isTestingConnection || !formData.kube_config.trim()}
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
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={updateMutation.isPending}>
                {updateMutation.isPending ? "Updating..." : "Update KubeConfig"}
              </Button>
            </div>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
