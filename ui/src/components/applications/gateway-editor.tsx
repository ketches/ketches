import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { InfoIcon, Loader2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { appsApi, type App, type GatewaySpec } from "@/api/apps"
import { certificatesApi } from "@/api/certificates"
import { clustersApi } from "@/api/clusters"
import { envsApi } from "@/api/envs"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
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
import { InputGroup, InputGroupAddon, InputGroupInput, InputGroupText } from "@/components/ui/input-group"
import { Item, ItemContent, ItemDescription, ItemTitle } from "@/components/ui/item"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"

interface GatewayEditorProps {
  app: App
  gateway?: GatewaySpec | null
  open?: boolean
  onOpenChange?: (open: boolean) => void
  onSuccess?: () => void
}



export function GatewayEditor({
  app,
  gateway,
  open: controlledOpen,
  onOpenChange: setControlledOpen,
  onSuccess,
}: GatewayEditorProps) {
  const [internalOpen, setInternalOpen] = React.useState(false)
  const open = controlledOpen !== undefined ? controlledOpen : internalOpen
  const setOpen = setControlledOpen || setInternalOpen
  const queryClient = useQueryClient()

  const isEditing = gateway !== null && gateway !== undefined

  const [errors, setErrors] = React.useState<{
    port?: string
    protocol?: string
    domain?: string
    path?: string
    gateway_port?: string
  }>({})

  const [formData, setFormData] = React.useState<GatewaySpec>({
    port: 80,
    protocol: 'http',
    domain: '',
    path: '/',
    gateway_port: undefined,
    exposed: true,
  })

  // Fetch environment to get cluster_id, then fetch both cluster and env certificates
  const { data: env } = useQuery({
    queryKey: ['env', app.env_id],
    queryFn: () => envsApi.get(app.env_id),
    enabled: !!app.env_id && open,
  })

  const { data: clusterCertsResponse } = useQuery({
    queryKey: ['cluster-certificates', env?.cluster_id],
    queryFn: () => certificatesApi.listByCluster(env!.cluster_id),
    enabled: !!env?.cluster_id && open,
  })

  const { data: envCertsResponse } = useQuery({
    queryKey: ['env-certificates', app.env_id],
    queryFn: () => certificatesApi.listByEnv(app.env_id),
    enabled: !!app.env_id && open,
  })

  // Check whether Gateway API is installed on the cluster.
  const { data: gatewayAPIStatus } = useQuery({
    queryKey: ['cluster-gateway-api-status', env?.cluster_id],
    queryFn: () => clustersApi.getGatewayAPIStatus(env!.cluster_id),
    enabled: !!env?.cluster_id && open,
  })
  const gatewayAPIInstalled = gatewayAPIStatus?.installed !== false

  // Combine cluster and env certificates for selection
  const certificates = React.useMemo(() => {
    const clusterCerts = (clusterCertsResponse?.items ?? []).map(c => ({ ...c, label: `[Cluster] ${c.name}` }))
    const envCerts = (envCertsResponse?.items ?? []).map(c => ({ ...c, label: `[Env] ${c.name}` }))
    return [...clusterCerts, ...envCerts]
  }, [clusterCertsResponse, envCertsResponse])

  React.useEffect(() => {
    if (open) {
      if (isEditing && gateway) {
        setFormData(gateway)
      } else {
        setFormData({
          port: 80,
          protocol: 'http',
          domain: '',
          path: '/',
          gateway_port: undefined,
          exposed: true,
        })
      }
      setErrors({})
    }
  }, [gateway, isEditing, open])

  const isHttpProtocol = formData.protocol === 'http' || formData.protocol === 'https'
  const isHttpsProtocol = formData.protocol === 'https'

  const saveMutation = useMutation({
    mutationFn: (data: GatewaySpec) => {
      if (isEditing && gateway?.id) {
        // Update existing gateway
        return appsApi.updateGateway(gateway.id, data)
      } else {
        // Create new gateway
        return appsApi.addGateway(app.id, data)
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['app-gateways', app.id] })
      toast.success(isEditing ? "Gateway updated successfully" : "Gateway created successfully")
      setOpen(false)
      onSuccess?.()
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: string } } }
      toast.error("Error", {
        description: err.response?.data?.error || `Failed to ${isEditing ? 'update' : 'create'} gateway`,
      })
    },
  })

  const validateForm = () => {
    const newErrors: typeof errors = {}

    if (!formData.port || formData.port < 1 || formData.port > 65535) {
      newErrors.port = "Port must be between 1 and 65535"
    }

    if (!formData.protocol) {
      newErrors.protocol = "Protocol is required"
    }

    // Only validate domain/path/gateway_port when exposed is true
    if (formData.exposed) {
      if (isHttpProtocol) {
        if (!formData.domain?.trim()) {
          newErrors.domain = "Domain is required for HTTP/HTTPS protocols"
        }
        if (!formData.path?.trim()) {
          newErrors.path = "Path is required"
        } else if (!formData.path.startsWith('/')) {
          newErrors.path = "Path must start with /"
        }
      } else {
        // TCP/UDP
        if (!formData.gateway_port) {
          newErrors.gateway_port = "Gateway port is required when exposed"
        } else if (formData.gateway_port < 1 || formData.gateway_port > 65535) {
          newErrors.gateway_port = "Gateway port must be between 1 and 65535"
        }
      }
    }

    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!validateForm()) {
      return
    }

    // Clean up fields based on protocol and exposed state
    const cleanedData = { ...formData }

    if (!formData.exposed) {
      // When not exposed, clear all routing-related fields
      cleanedData.domain = ''
      cleanedData.path = ''
      cleanedData.gateway_port = undefined
      cleanedData.cert_id = undefined
    } else {
      // When exposed, clean up based on protocol
      if (!isHttpProtocol) {
        cleanedData.domain = ''
        cleanedData.path = ''
        cleanedData.cert_id = undefined
      } else {
        cleanedData.gateway_port = undefined
      }
    }

    saveMutation.mutate(cleanedData)
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="sm:max-w-160 max-h-[90vh] overflow-y-auto">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>{isEditing ? 'Edit Gateway' : 'Add Gateway'}</DialogTitle>
            <DialogDescription>
              Configure network gateway to expose your application.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            {/* Port and Protocol in one row */}
            <div className="grid grid-cols-3 gap-4">
              <div className="col-span-2">
                <Field>
                  <FieldLabel>
                    Container Port *
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
                        <p className="text-xs">The port your application listens on inside the container (1-65535).</p>
                      </TooltipContent>
                    </Tooltip>
                  </FieldLabel>
                  <FieldContent>
                    <Input
                      type="number"
                      placeholder="80"
                      value={formData.port}
                      onChange={(e) => setFormData((prev) => ({ ...prev, port: parseInt(e.target.value) || 0 }))}
                      aria-invalid={!!errors.port}
                    />
                  </FieldContent>
                  {errors.port && (
                    <FieldError>
                      <span className="text-destructive text-xs">{errors.port}</span>
                    </FieldError>
                  )}
                </Field>
              </div>

              <Field>
                <FieldLabel>Protocol *</FieldLabel>
                <FieldContent>
                  <Combobox
                    value={formData.protocol}
                    onValueChange={(value: string | null) => value && setFormData((prev) => ({ ...prev, protocol: value }))}
                  >
                    <ComboboxInput />
                    <ComboboxContent>
                      <ComboboxList>
                        {[
                          { value: "http", label: "HTTP", description: "Plaintext HTTP routing" },
                          { value: "https", label: "HTTPS", description: "TLS-terminated HTTPS routing" },
                          { value: "tcp", label: "TCP", description: "Raw TCP passthrough" },
                          { value: "udp", label: "UDP", description: "Raw UDP passthrough" },
                        ].map((option) => (
                          <ComboboxItem key={option.value} value={option.value}>
                            {/* <div className="flex flex-col gap-0.5">
                              <span>{option.label}</span>
                              <span className="text-muted-foreground text-[10px] leading-relaxed">{option.description}</span>
                            </div> */}

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
                {errors.protocol && (
                  <FieldError>
                    <span className="text-destructive text-xs">{errors.protocol}</span>
                  </FieldError>
                )}
              </Field>
            </div>

            {/* Public Access Checkbox */}
            <Field orientation="horizontal" className="flex items-center gap-2">
              <FieldContent>
                <Tooltip>
                  <TooltipTrigger
                    tabIndex={-1}
                    render={
                      <div className="flex items-center gap-2">
                        <Checkbox
                          checked={formData.exposed}
                          onCheckedChange={(checked) => setFormData((prev) => ({ ...prev, exposed: !!checked }))}
                          disabled={!gatewayAPIInstalled}
                        />
                        <label htmlFor="exposed" className={`cursor-pointer ${!gatewayAPIInstalled ? 'text-muted-foreground' : ''}`}>Enable public access</label>
                      </div>
                    }
                  />
                  <TooltipContent side="top" align="start" className="max-w-64">
                    {gatewayAPIInstalled
                      ? <p className="text-xs">When enabled, creates routes to expose this service externally.</p>
                      : <p className="text-xs">Gateway API is not installed on this cluster. Contact your administrator to install the gateway extension.</p>
                    }
                  </TooltipContent>
                </Tooltip>
              </FieldContent>
            </Field>

            {/* HTTP/HTTPS specific fields - Only show when exposed */}
            {formData.exposed && isHttpProtocol && (
              <>
                <div className="grid grid-cols-3 gap-4">
                  <div className="col-span-2">
                    <Field>
                      <FieldLabel>
                        Domain *
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
                            <p className="text-xs">The domain name for accessing your application.</p>
                          </TooltipContent>
                        </Tooltip>
                      </FieldLabel>
                      <FieldContent>
                        <InputGroup>
                          <InputGroupInput placeholder="app.example.com" value={formData.domain}
                            onChange={(e) => setFormData((prev) => ({ ...prev, domain: e.target.value }))}
                            aria-invalid={!!errors.domain} />
                          <InputGroupAddon>
                            <InputGroupText>{formData.protocol}://</InputGroupText>
                          </InputGroupAddon>
                          <InputGroupAddon align="inline-end">
                            <InfoIcon />
                          </InputGroupAddon>
                        </InputGroup>
                      </FieldContent>
                      {errors.domain && (
                        <FieldError>
                          <span className="text-destructive text-xs">{errors.domain}</span>
                        </FieldError>
                      )}
                    </Field>
                  </div>

                  <Field>
                    <FieldLabel>
                      Path *
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
                          <p className="text-xs">URL path prefix (must start with /).</p>
                        </TooltipContent>
                      </Tooltip>
                    </FieldLabel>
                    <FieldContent>
                      <Input
                        placeholder="/"
                        value={formData.path}
                        onChange={(e) => setFormData((prev) => ({ ...prev, path: e.target.value }))}
                        aria-invalid={!!errors.path}
                      />
                    </FieldContent>
                    {errors.path && (
                      <FieldError>
                        <span className="text-destructive text-xs">{errors.path}</span>
                      </FieldError>
                    )}
                  </Field>
                </div>

                {/* HTTPS Certificate Selection */}
                {isHttpsProtocol && (
                  <Field>
                    <FieldLabel>
                      TLS Certificate
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
                          <p className="text-xs">Select a TLS certificate for HTTPS. Certificates are managed in the environment settings.</p>
                        </TooltipContent>
                      </Tooltip>
                    </FieldLabel>
                    <FieldContent>
                      <Combobox
                        value={formData.cert_id || null}
                        onValueChange={(v: string | null) => setFormData((prev) => ({ ...prev, cert_id: v || undefined }))}
                        itemToStringLabel={(v) => certificates.find((c) => c.id === v)?.label ?? v ?? ""}
                      >
                        <ComboboxInput placeholder="Select a certificate" />
                        <ComboboxContent>
                          <ComboboxList>
                            {certificates.map((cert) => (
                              <ComboboxItem key={cert.id} value={cert.id}>
                                {cert.label}
                              </ComboboxItem>
                            ))}
                          </ComboboxList>
                        </ComboboxContent>
                      </Combobox>
                    </FieldContent>
                  </Field>
                )}
              </>
            )}

            {/* TCP/UDP specific fields - Only show when exposed */}
            {formData.exposed && !isHttpProtocol && (
              <Field>
                <FieldLabel>
                  Gateway Port {formData.exposed ? '*' : '(Optional)'}
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
                      <p className="text-xs">The external port exposed by the gateway for TCP/UDP protocols.</p>
                    </TooltipContent>
                  </Tooltip>
                </FieldLabel>
                <FieldContent>
                  <Input
                    type="number"
                    placeholder="30000"
                    value={formData.gateway_port || ''}
                    onChange={(e) => setFormData((prev) => ({ ...prev, gateway_port: parseInt(e.target.value) || undefined }))}
                    aria-invalid={!!errors.gateway_port}
                  />
                </FieldContent>
                {errors.gateway_port && (
                  <FieldError>
                    <span className="text-destructive text-xs">{errors.gateway_port}</span>
                  </FieldError>
                )}
              </Field>
            )}
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={saveMutation.isPending}>
              {saveMutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
              {isEditing ? 'Update' : 'Create'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog >
  )
}
