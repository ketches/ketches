import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { ChevronDown, InfoIcon, Loader2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { appsApi, type App, type GatewaySpec } from "@/api/apps"
import { certificatesApi } from "@/api/certificates"
import { clustersApi } from "@/api/clusters"
import { domainsApi } from "@/api/domains"
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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Field, FieldContent, FieldError, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { InputGroup, InputGroupAddon, InputGroupInput, InputGroupText } from "@/components/ui/input-group"
import { Item, ItemContent, ItemDescription, ItemTitle } from "@/components/ui/item"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { isPatternDomain, isValidDomainValue, normalizeDomainValue, seedDomainInputFromSelection } from "@/lib/domain"
import { getErrorMessage } from "@/lib/utils"

interface GatewayEditorProps {
  app: App
  gateway?: GatewaySpec | null
  open?: boolean
  onOpenChange?: (open: boolean) => void
  onSuccess?: () => void
}

const PROTOCOL_OPTIONS = [
  { value: "http", label: "HTTP", description: "Plaintext HTTP routing" },
  { value: "https", label: "HTTPS", description: "TLS-terminated HTTPS routing" },
  { value: "tcp", label: "TCP", description: "Raw TCP passthrough" },
  { value: "udp", label: "UDP", description: "Raw UDP passthrough" },
] as const
const PUBLIC_ACCESS_CHECKBOX_ID = "gateway-public-access"

const SERVICE_TYPE_OPTIONS = [
  { value: "ClusterIP", label: "ClusterIP", description: "Internal only, accessible within the cluster" },
  { value: "NodePort", label: "NodePort", description: "Exposed on a static port on every cluster node" },
] as const

function inferSelectedDomainOption(domain: string, options: Array<{ value: string; domain: string }>) {
  const normalizedDomain = normalizeDomainValue(domain)
  const matched = options.find((option) => {
    const normalizedOptionDomain = normalizeDomainValue(option.domain)
    if (!isPatternDomain(normalizedOptionDomain)) {
      return normalizedOptionDomain === normalizedDomain
    }

    const suffix = normalizedOptionDomain.slice(1)
    return normalizedDomain.endsWith(suffix) && normalizedDomain !== suffix.slice(1)
  })
  return matched?.value ?? "custom"
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
    cert_id?: string
    gateway_port?: string
    node_port?: string
  }>({})
  const [selectedDomainOption, setSelectedDomainOption] = React.useState("custom")
  const [domainInput, setDomainInput] = React.useState("")

  const [formData, setFormData] = React.useState<GatewaySpec>({
    port: 80,
    protocol: 'http',
    domain: '',
    path: '/',
    gateway_port: undefined,
    service_type: 'ClusterIP',
    node_port: undefined,
    exposed: false,
  })

  const env = app.env

  const { data: clusterCertsResponse } = useQuery({
    queryKey: ['cluster-certificates', env?.cluster_id],
    queryFn: () => certificatesApi.listByCluster(env!.cluster_id, undefined, env!.project_id),
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
    queryFn: () => clustersApi.getGatewayAPIStatus(env!.cluster_id, env!.project_id),
    enabled: !!env?.cluster_id && open,
  })
  const gatewayAPIInstalled = gatewayAPIStatus?.installed !== false

  const { data: clusterDomainsResponse } = useQuery({
    queryKey: ["cluster-domains", env?.cluster_id],
    queryFn: () => domainsApi.listByCluster(env!.cluster_id, undefined, env!.project_id),
    enabled: !!env?.cluster_id && open,
  })

  const { data: envDomainsResponse } = useQuery({
    queryKey: ["env-domains", app.env_id],
    queryFn: () => domainsApi.listByEnv(app.env_id),
    enabled: !!app.env_id && open,
  })

  // Combine cluster and env certificates for selection
  const certificates = React.useMemo(() => {
    const clusterCerts = (clusterCertsResponse?.items ?? []).map(c => ({ ...c, label: `[Cluster] ${c.name}` }))
    const envCerts = (envCertsResponse?.items ?? []).map(c => ({ ...c, label: `[Env] ${c.name}` }))
    return [...clusterCerts, ...envCerts]
  }, [clusterCertsResponse, envCertsResponse])

  const domainOptions = React.useMemo(() => {
    const envOptions = (envDomainsResponse?.items ?? [])
      .map((item) => ({
        value: `env:${item.id}`,
        label: `[Env] ${item.name}`,
        description: item.domain,
        domain: item.domain,
      }))

    const clusterOptions = (clusterDomainsResponse?.items ?? [])
      .map((item) => ({
        value: `cluster:${item.id}`,
        label: `[Cluster] ${item.name}`,
        description: item.domain,
        domain: item.domain,
      }))

    return [
      ...envOptions,
      ...clusterOptions,
      {
        value: "custom",
        label: "Custom Domain",
        description: "Enter a fully qualified domain manually",
        domain: "",
      },
    ]
  }, [app.slug, clusterDomainsResponse?.items, envDomainsResponse?.items])
  const normalizedDomainInput = normalizeDomainValue(domainInput)
  const selectedDomainLabel = domainOptions.find((option) => option.value === selectedDomainOption)?.label ?? "Select"

  React.useEffect(() => {
    if (open) {
      if (isEditing && gateway) {
        setFormData(gateway)
        setDomainInput(gateway.domain)
        setSelectedDomainOption("custom")
      } else {
        setFormData({
          port: 80,
          protocol: 'http',
          domain: '',
          path: '/',
          gateway_port: undefined,
          service_type: 'ClusterIP',
          node_port: undefined,
          exposed: false,
        })
        setDomainInput("")
        setSelectedDomainOption("custom")
      }
      setErrors({})
    }
  }, [gateway, isEditing, open])

  const isHttpProtocol = formData.protocol === 'http' || formData.protocol === 'https'
  const isHttpsProtocol = formData.protocol === 'https'
  const supportsPublicExposure = isHttpProtocol
  const publicAccessDisabledReason = !supportsPublicExposure
    ? 'Public access is currently available only for HTTP/HTTPS gateways. TCP/UDP public exposure is not supported yet.'
    : !gatewayAPIInstalled
      ? 'Gateway API is not installed on this cluster, so HTTP/HTTPS public access is currently unavailable.'
      : null

  React.useEffect(() => {
    if (selectedDomainOption !== "custom" && !domainOptions.some((option) => option.value === selectedDomainOption)) {
      setSelectedDomainOption("custom")
    }
  }, [selectedDomainOption, domainOptions])

  React.useEffect(() => {
    if (!open || !isEditing || !gateway) {
      return
    }

    const nextOption = inferSelectedDomainOption(gateway.domain, domainOptions)
    setSelectedDomainOption((current) => current === nextOption ? current : nextOption)
  }, [gateway, isEditing, open, domainOptions])

  React.useEffect(() => {
    if (!supportsPublicExposure && formData.exposed) {
      setFormData((prev) => ({
        ...prev,
        exposed: false,
        gateway_port: undefined,
      }))
    }
  }, [formData.exposed, supportsPublicExposure])

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
      toast.error("Error", {
        description: getErrorMessage(error, `Failed to ${isEditing ? 'update' : 'create'} gateway`),
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

    // Validate NodePort range when ServiceType is NodePort and a custom value is provided
    if (formData.service_type === 'NodePort' && formData.node_port) {
      if (formData.node_port < 30000 || formData.node_port > 32767) {
        newErrors.node_port = "NodePort must be between 30000 and 32767"
      }
    }

    // Only validate domain/path/gateway_port when exposed is true
    if (formData.exposed) {
      if (isHttpProtocol) {
        if (!normalizedDomainInput) {
          newErrors.domain = "Domain is required for HTTP/HTTPS protocols"
        } else if (!isValidDomainValue(normalizedDomainInput)) {
          newErrors.domain = "Must be a valid domain such as example.com or *.example.com"
        }
        if (!formData.path?.trim()) {
          newErrors.path = "Path is required"
        } else if (!formData.path.startsWith('/')) {
          newErrors.path = "Path must start with /"
        }
        if (isHttpsProtocol && !formData.cert_id) {
          newErrors.cert_id = "TLS certificate is required for HTTPS"
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

  const handleSubmit = async (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault()

    if (!validateForm()) {
      return
    }

    // Clean up fields based on protocol and exposed state
    const cleanedData = { ...formData }

    // Clean up ServiceType / NodePort
    if (cleanedData.service_type !== 'NodePort') {
      cleanedData.node_port = undefined
    }

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
        cleanedData.domain = normalizedDomainInput
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
              <Field className="col-span-2">
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
                    <TooltipContent side="top" align="start" className="max-w-100 flex-col items-start">
                      <p className="text-xs">- The port your application listens on inside the container (1-65535).</p>
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

              <Field>
                <FieldLabel>Protocol *</FieldLabel>
                <FieldContent>
                  <Combobox
                    value={formData.protocol}
                    onValueChange={(value: string | null) => {
                      if (!value) {
                        return
                      }

                      const nextSupportsPublicExposure = value === "http" || value === "https"
                      setFormData((prev) => ({
                        ...prev,
                        protocol: value,
                        exposed: nextSupportsPublicExposure ? prev.exposed : false,
                        gateway_port: nextSupportsPublicExposure ? prev.gateway_port : undefined,
                      }))
                    }}
                    itemToStringLabel={(v) => PROTOCOL_OPTIONS.find((opt) => opt.value === v)?.label ?? v ?? ""}
                  >
                    <ComboboxInput />
                    <ComboboxContent>
                      <ComboboxList>
                        {PROTOCOL_OPTIONS.map((option) => (
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
                {errors.protocol && (
                  <FieldError>
                    <span className="text-destructive text-xs">{errors.protocol}</span>
                  </FieldError>
                )}
              </Field>
            </div>

            {/* Service Type selector */}
            <div className="grid grid-cols-3 gap-4">
              <div className={formData.service_type === 'NodePort' ? 'col-span-2' : 'col-span-3'}>
                <Field>
                  <FieldLabel>Service Type</FieldLabel>
                  <FieldContent>
                    <Combobox
                      value={formData.service_type || 'ClusterIP'}
                      onValueChange={(value: string | null) => {
                        const st = value || 'ClusterIP'
                        setFormData((prev) => ({
                          ...prev,
                          service_type: st,
                          node_port: st !== 'NodePort' ? undefined : prev.node_port,
                        }))
                      }}
                      itemToStringLabel={(v) => SERVICE_TYPE_OPTIONS.find((o) => o.value === v)?.label ?? v ?? ""}
                    >
                      <ComboboxInput />
                      <ComboboxContent>
                        <ComboboxList>
                          {SERVICE_TYPE_OPTIONS.map((option) => (
                            <ComboboxItem key={option.value} value={option.value}>
                              <Item size="xs" className="p-0">
                                <ItemContent>
                                  <ItemTitle className="whitespace-nowrap">{option.label}</ItemTitle>
                                  <ItemDescription>{option.description}</ItemDescription>
                                </ItemContent>
                              </Item>
                            </ComboboxItem>
                          ))}
                        </ComboboxList>
                      </ComboboxContent>
                    </Combobox>
                  </FieldContent>
                </Field>
              </div>

              {formData.service_type === 'NodePort' && (
                <Field>
                  <FieldLabel>NodePort</FieldLabel>
                  <FieldContent>
                    <Input
                      type="number"
                      placeholder="Auto"
                      min={30000}
                      max={32767}
                      value={formData.node_port || ''}
                      onChange={(e) => setFormData((prev) => ({ ...prev, node_port: parseInt(e.target.value) || undefined }))}
                      aria-invalid={!!errors.node_port}
                    />
                  </FieldContent>
                  {errors.node_port && (
                    <FieldError>
                      <span className="text-destructive text-xs">{errors.node_port}</span>
                    </FieldError>
                  )}
                </Field>
              )}
            </div>

            {/* Public Access Checkbox */}
            <Field orientation="horizontal" className="flex items-center gap-2">
              <FieldContent>
                <div className="space-y-2">
                  <div className="flex items-center gap-2">
                    <Checkbox
                      id={PUBLIC_ACCESS_CHECKBOX_ID}
                      checked={supportsPublicExposure ? formData.exposed : false}
                      onCheckedChange={(checked) => {
                        if (!supportsPublicExposure) {
                          return
                        }
                        setFormData((prev) => ({ ...prev, exposed: !!checked }))
                      }}
                      disabled={!gatewayAPIInstalled || !supportsPublicExposure}
                    />
                    <label
                      htmlFor={PUBLIC_ACCESS_CHECKBOX_ID}
                      className={`cursor-pointer ${!gatewayAPIInstalled || !supportsPublicExposure ? 'text-muted-foreground' : ''}`}
                    >
                      Enable public access
                    </label>
                    {publicAccessDisabledReason && (
                      <Tooltip>
                        <TooltipTrigger
                          tabIndex={-1}
                          render={
                            <button type="button" className="text-muted-foreground hover:text-foreground transition-colors outline-none">
                              <InfoIcon className="h-3.5 w-3.5 text-orange-600" />
                            </button>
                          }
                        />
                        <TooltipContent side="top" align="start" className="max-w-100 flex-col items-start">
                          <p className="text-xs">- {publicAccessDisabledReason}</p>
                        </TooltipContent>
                      </Tooltip>
                    )}
                  </div>
                </div>
              </FieldContent>
            </Field>

            {/* HTTP/HTTPS specific fields - Only show when exposed */}
            {formData.exposed && isHttpProtocol && (
              <>
                <div className="grid grid-cols-3 gap-4">
                  <Field className="col-span-2">
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
                        <TooltipContent side="top" align="start" className="max-w-100 flex-col items-start">
                          <p className="text-xs">- Choose a saved domain or type one directly.</p>
                          <p className="text-xs mt-1">- If the selected domain starts with `*.` the input removes the `*` so you can enter the hostname prefix yourself.</p>
                        </TooltipContent>
                      </Tooltip>
                    </FieldLabel>
                    <FieldContent>
                      <InputGroup>
                        <InputGroupAddon align="inline-start">
                          <InputGroupText>{formData.protocol}://</InputGroupText>
                        </InputGroupAddon>
                        <InputGroupInput
                          placeholder="app.example.com"
                          value={domainInput}
                          onChange={(e) => {
                            setSelectedDomainOption("custom")
                            setDomainInput(e.target.value)
                          }}
                          aria-invalid={!!errors.domain}
                        />
                        <InputGroupAddon>
                          <DropdownMenu>
                            <DropdownMenuTrigger
                              render={
                                <Button type="button" variant="ghost" size="sm">
                                  {selectedDomainLabel}
                                  <ChevronDown />
                                </Button>
                              }
                            />
                            <DropdownMenuContent align="end" className="w-64">
                              {domainOptions.map((option) => (
                                <DropdownMenuItem
                                  key={option.value}
                                  onClick={() => {
                                    setSelectedDomainOption(option.value)
                                    if (option.value === "custom") {
                                      return
                                    }
                                    setDomainInput(seedDomainInputFromSelection(option.domain))
                                  }}
                                >
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
                                </DropdownMenuItem>
                              ))}
                            </DropdownMenuContent>
                          </DropdownMenu>
                        </InputGroupAddon>
                      </InputGroup>
                    </FieldContent>
                    {errors.domain && (
                      <FieldError>
                        <span className="text-destructive text-xs">{errors.domain}</span>
                      </FieldError>
                    )}
                  </Field>

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
                        <TooltipContent side="top" align="start" className="max-w-100 flex-col items-start">
                          <p className="text-xs">- URL path prefix (must start with /).</p>
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
                      TLS Certificate *
                      <Tooltip>
                        <TooltipTrigger
                          tabIndex={-1}
                          render={
                            <button type="button" className="text-muted-foreground hover:text-foreground transition-colors outline-none">
                              <InfoIcon className="h-3.5 w-3.5" />
                            </button>
                          }
                        />
                        <TooltipContent side="top" align="start" className="max-w-100 flex-col items-start">
                          <p className="text-xs">- Select a TLS certificate for HTTPS. Certificates are managed in the environment settings.</p>
                        </TooltipContent>
                      </Tooltip>
                    </FieldLabel>
                    <FieldContent>
                      <Combobox
                        value={formData.cert_id || null}
                        onValueChange={(v: string | null) => setFormData((prev) => ({ ...prev, cert_id: v || undefined }))}
                        itemToStringLabel={(v) => certificates.find((c) => c.id === v)?.label ?? v ?? ""}
                      >
                        <ComboboxInput placeholder="Select a certificate" aria-invalid={!!errors.cert_id} />
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
                    {errors.cert_id && (
                      <FieldError>
                        <span className="text-destructive text-xs">{errors.cert_id}</span>
                      </FieldError>
                    )}
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
                    <TooltipContent side="top" align="start" className="max-w-100 flex-col items-start">
                      <p className="text-xs">- The external port exposed by the gateway for TCP/UDP protocols.</p>
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
