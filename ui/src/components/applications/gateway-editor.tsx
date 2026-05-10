import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { ChevronDown, ChevronRight, Copy, InfoIcon, Loader2, Plus, Trash2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { appsApi, type App, type GatewayRouteBackendSpec, type GatewayRouteExtension, type GatewayRouteFilters, type GatewayRouteMatches, type GatewayRouteSpec, type GatewaySpec } from "@/api/apps"
import { certificatesApi } from "@/api/certificates"
import { clustersApi } from "@/api/clusters"
import { domainsApi } from "@/api/domains"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Combobox,
  ComboboxContent,
  ComboboxGroup,
  ComboboxInput,
  ComboboxItem,
  ComboboxLabel,
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
import { InputGroupAddon, InputGroupText } from "@/components/ui/input-group"
import { Item, ItemContent, ItemDescription, ItemTitle } from "@/components/ui/item"
import { Separator } from "@/components/ui/separator"
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

const GATEWAY_PROTOCOL_OPTIONS = [
  { value: "http", label: "HTTP", description: "Application HTTP port with optional HTTPRoutes" },
  { value: "tcp", label: "TCP", description: "Internal TCP service port" },
  { value: "udp", label: "UDP", description: "Internal UDP service port" },
] as const

const ROUTE_PROTOCOL_OPTIONS = [
  { value: "http", label: "HTTP", description: "Plaintext listener" },
  { value: "https", label: "HTTPS", description: "TLS listener with certificate" },
] as const

const SERVICE_TYPE_OPTIONS = [
  { value: "ClusterIP", label: "ClusterIP", description: "Internal only, accessible within the cluster" },
  { value: "NodePort", label: "NodePort", description: "Exposed on a static port on every cluster node" },
] as const

const PATH_MATCH_OPTIONS = [
  { value: "PathPrefix", label: "Prefix", description: "Match this path and child paths" },
  { value: "Exact", label: "Exact", description: "Match this path only" },
] as const

const METHOD_OPTIONS = [
  { value: "", label: "Any", description: "Match any HTTP method" },
  { value: "GET", label: "GET", description: "GET requests" },
  { value: "POST", label: "POST", description: "POST requests" },
  { value: "PUT", label: "PUT", description: "PUT requests" },
  { value: "PATCH", label: "PATCH", description: "PATCH requests" },
  { value: "DELETE", label: "DELETE", description: "DELETE requests" },
] as const

const CLUSTER_SCOPE_LABEL = "Cluster Scope"
const ENVIRONMENT_SCOPE_LABEL = "Environment Scope"
const DEFAULT_WEIGHT = 100

type DomainOption = {
  value: string
  label: string
  domain: string
}

type CertificateOption = {
  id: string
  label: string
  description: string
}

type ComboboxOptionGroup<T> = {
  label: string
  items: T[]
}

type GatewayRouteFormState = GatewayRouteSpec & {
  form_id: string
}

type GatewayFormState = Omit<GatewaySpec, "routes"> & {
  routes: GatewayRouteFormState[]
}

type RouteError = {
  host?: string
  path?: string
  cert_id?: string
  timeout?: string
  backend?: string
}

type FormErrors = {
  port?: string
  protocol?: string
  node_port?: string
  form?: string
  routes: Record<string, RouteError>
}

let routeFormID = 0

function nextRouteFormID() {
  routeFormID += 1
  return `route-form-${routeFormID}`
}

function inferSelectedDomainOption(domain: string | undefined, options: Array<{ value: string; domain: string }>) {
  const normalizedDomain = normalizeDomainValue(domain ?? "")
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

function defaultBackend(appID: string, port: number): GatewayRouteBackendSpec {
  return {
    backend_app_id: appID,
    backend_port: port,
    weight: DEFAULT_WEIGHT,
  }
}

function defaultRoute(appID: string, port: number): GatewayRouteFormState {
  return {
    form_id: nextRouteFormID(),
    host: "",
    listener_protocol: "http",
    path: "/",
    path_match_type: "PathPrefix",
    enabled: true,
    backends: [defaultBackend(appID, port)],
  }
}

function toFormRoute(route: GatewayRouteSpec, appID: string, port: number): GatewayRouteFormState {
  return {
    ...route,
    form_id: nextRouteFormID(),
    listener_protocol: route.listener_protocol || "http",
    path: route.path || "/",
    path_match_type: route.path_match_type || "PathPrefix",
    enabled: route.enabled ?? true,
    backends: route.backends && route.backends.length > 0 ? route.backends : [defaultBackend(appID, port)],
  }
}

function buildInitialFormState(app: App, gateway?: GatewaySpec | null): GatewayFormState {
  if (gateway) {
    const protocol = gateway.protocol || "http"
    const routes = (gateway.routes ?? []).map((route) => toFormRoute(route, app.id, gateway.port || 80))
    return {
      id: gateway.id,
      app_id: gateway.app_id,
      port: gateway.port || 80,
      protocol,
      gateway_port: gateway.gateway_port,
      service_type: gateway.service_type || "ClusterIP",
      node_port: gateway.node_port,
      gateway_host: gateway.gateway_host,
      internal_address: gateway.internal_address,
      routes,
    }
  }

  return {
    port: 80,
    protocol: "http",
    service_type: "ClusterIP",
    routes: [defaultRoute(app.id, 80)],
  }
}

function parseDurationMillis(value: string | undefined) {
  const trimmed = value?.trim()
  if (!trimmed) {
    return null
  }

  const match = trimmed.match(/^(\d+(?:\.\d+)?)(ms|s|m|h)$/)
  if (!match) {
    return Number.NaN
  }

  const amount = Number(match[1])
  const unit = match[2]
  if (unit === "ms") return amount
  if (unit === "s") return amount * 1000
  if (unit === "m") return amount * 60 * 1000
  return amount * 60 * 60 * 1000
}

function compactRoute(route: GatewayRouteFormState): GatewayRouteSpec {
  const requestTimeout = route.timeouts?.request?.trim()
  const backendTimeout = route.timeouts?.backend_request?.trim()
  const timeouts = requestTimeout || backendTimeout
    ? {
      ...(requestTimeout ? { request: requestTimeout } : {}),
      ...(backendTimeout ? { backend_request: backendTimeout } : {}),
    }
    : undefined

  const method = route.matches?.method?.trim()
  const matches: GatewayRouteMatches | undefined = method ? { method } : undefined

  const headerName = route.filters?.request_headers?.set?.[0]?.name?.trim()
  const headerValue = route.filters?.request_headers?.set?.[0]?.value?.trim()
  const filters: GatewayRouteFilters | undefined = headerName && headerValue
    ? { request_headers: { set: [{ name: headerName, value: headerValue }] } }
    : undefined

  const sessionName = route.session_persistence?.session_name?.trim()
  const sessionPersistence = sessionName
    ? {
      type: "Cookie",
      session_name: sessionName,
      cookie_lifetime_type: route.session_persistence?.cookie_lifetime_type || "Session",
    }
    : undefined

  const requestBodySize = route.extension?.request_body_size?.trim()
  const extension: GatewayRouteExtension = {}
  if (requestBodySize) {
    extension.request_body_size = requestBodySize
  }
  if (route.extension?.keep_alive) {
    extension.keep_alive = true
  }
  if (route.extension?.websocket) {
    extension.websocket = true
  }

  const hasExtension = Object.keys(extension).length > 0
  const backend = route.backends?.[0]

  return {
    ...(route.id ? { id: route.id } : {}),
    ...(route.gateway_id ? { gateway_id: route.gateway_id } : {}),
    host: normalizeDomainValue(route.host ?? ""),
    listener_protocol: route.listener_protocol || "http",
    path: route.path?.trim() || "/",
    path_match_type: route.path_match_type || "PathPrefix",
    enabled: route.enabled ?? true,
    ...(route.listener_protocol === "https" && route.cert_id ? { cert_id: route.cert_id } : {}),
    ...(matches ? { matches } : {}),
    ...(filters ? { filters } : {}),
    ...(timeouts ? { timeouts } : {}),
    ...(sessionPersistence ? { session_persistence: sessionPersistence } : {}),
    ...(hasExtension ? { extension } : {}),
    backends: [{
      ...(backend?.id ? { id: backend.id } : {}),
      ...(backend?.route_id ? { route_id: backend.route_id } : {}),
      backend_app_id: backend?.backend_app_id || "",
      backend_app_slug: backend?.backend_app_slug,
      backend_port: backend?.backend_port || 0,
      weight: backend?.weight ?? DEFAULT_WEIGHT,
    }],
    ...(route.sort_order ? { sort_order: route.sort_order } : {}),
  }
}

function getRouteSummary(route: GatewayRouteFormState) {
  const host = route.host?.trim() || "No host"
  const path = route.path?.trim() || "/"
  return path === "/" ? host : `${host}${path}`
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
  const [formData, setFormData] = React.useState<GatewayFormState>(() => buildInitialFormState(app, gateway))
  const [errors, setErrors] = React.useState<FormErrors>({ routes: {} })
  const [expandedRoutes, setExpandedRoutes] = React.useState<Record<string, boolean>>({})
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

  const certificateGroups = React.useMemo<ComboboxOptionGroup<CertificateOption>[]>(() => {
    const clusterCerts = (clusterCertsResponse?.items ?? []).map((item) => ({
      id: item.id,
      label: item.name,
      description: item.description,
    }))
    const envCerts = (envCertsResponse?.items ?? []).map((item) => ({
      id: item.id,
      label: item.name,
      description: item.description,
    }))

    return [
      {
        label: CLUSTER_SCOPE_LABEL,
        items: clusterCerts,
      },
      {
        label: ENVIRONMENT_SCOPE_LABEL,
        items: envCerts,
      },
    ].filter((group) => group.items.length > 0)
  }, [clusterCertsResponse, envCertsResponse])

  const certificates = React.useMemo(() => {
    return certificateGroups.flatMap((group) => group.items)
  }, [certificateGroups])

  const domainOptionGroups = React.useMemo<ComboboxOptionGroup<DomainOption>[]>(() => {
    const clusterOptions = (clusterDomainsResponse?.items ?? [])
      .map((item) => ({
        value: `cluster:${item.id}`,
        label: `${item.name}`,
        domain: item.domain,
      }))

    const envOptions = (envDomainsResponse?.items ?? [])
      .map((item) => ({
        value: `env:${item.id}`,
        label: `${item.name}`,
        domain: item.domain,
      }))

    return [
      {
        label: CLUSTER_SCOPE_LABEL,
        items: clusterOptions,
      },
      {
        label: ENVIRONMENT_SCOPE_LABEL,
        items: envOptions,
      },
    ]
      .filter((group) => group.items.length > 0)
  }, [clusterDomainsResponse?.items, envDomainsResponse?.items])

  const domainOptions = React.useMemo(() => {
    return domainOptionGroups.flatMap((group) => group.items)
  }, [domainOptionGroups])

  React.useEffect(() => {
    if (!open) {
      return
    }

    const nextFormData = buildInitialFormState(app, gateway)
    setFormData(nextFormData)
    setExpandedRoutes(Object.fromEntries(nextFormData.routes.map((route) => [route.form_id, true])))
    setErrors({ routes: {} })
  }, [app, gateway, open])

  const saveMutation = useMutation({
    mutationFn: (data: GatewaySpec) => {
      if (isEditing && gateway?.id) {
        return appsApi.updateGateway(gateway.id, data)
      }

      return appsApi.addGateway(app.id, data)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['app-gateways', app.id] })
      toast.success(isEditing ? "Gateway updated successfully" : "Gateway created successfully")
      setOpen(false)
      onSuccess?.()
    },
    onError: (error: unknown) => {
      const message = getErrorMessage(error, `Failed to ${isEditing ? 'update' : 'create'} gateway`)
      const nextErrors: FormErrors = { routes: {}, form: message }
      const firstHTTPSRoute = formData.routes.find((route) => route.listener_protocol === "https")
      if (firstHTTPSRoute && message.toLowerCase().includes("certificate")) {
        nextErrors.routes[firstHTTPSRoute.form_id] = { cert_id: message }
      }
      if (message.toLowerCase().includes("duplicate")) {
        nextErrors.form = "Duplicate route host, listener protocol, path match type, and path."
      }
      setErrors(nextErrors)
      toast.error("Error", {
        description: message,
      })
    },
  })

  const updateRoute = (formID: string, updater: (route: GatewayRouteFormState) => GatewayRouteFormState) => {
    setFormData((prev) => ({
      ...prev,
      routes: prev.routes.map((route) => route.form_id === formID ? updater(route) : route),
    }))
  }

  const addRoute = () => {
    const route = defaultRoute(app.id, formData.port)
    setFormData((prev) => ({
      ...prev,
      routes: [...prev.routes, route],
    }))
    setExpandedRoutes((prev) => ({ ...prev, [route.form_id]: true }))
  }

  const duplicateRoute = (route: GatewayRouteFormState) => {
    const copy: GatewayRouteFormState = {
      ...route,
      id: undefined,
      gateway_id: undefined,
      form_id: nextRouteFormID(),
      host: route.host ? `copy-${route.host}` : "",
      backends: route.backends?.map((backend) => ({
        ...backend,
        id: undefined,
        route_id: undefined,
      })),
    }
    setFormData((prev) => ({
      ...prev,
      routes: [...prev.routes, copy],
    }))
    setExpandedRoutes((prev) => ({ ...prev, [copy.form_id]: true }))
  }

  const deleteRoute = (formID: string) => {
    setFormData((prev) => ({
      ...prev,
      routes: prev.routes.filter((route) => route.form_id !== formID),
    }))
  }

  const toggleRouteExpanded = (formID: string) => {
    setExpandedRoutes((prev) => ({
      ...prev,
      [formID]: !prev[formID],
    }))
  }

  const validateForm = () => {
    const nextErrors: FormErrors = { routes: {} }

    if (!formData.port || formData.port < 1 || formData.port > 65535) {
      nextErrors.port = "Port must be between 1 and 65535"
    }

    if (!formData.protocol) {
      nextErrors.protocol = "Protocol is required"
    }

    if (formData.service_type === "NodePort" && formData.node_port) {
      if (formData.node_port < 30000 || formData.node_port > 32767) {
        nextErrors.node_port = "NodePort must be between 30000 and 32767"
      }
    }

    if (formData.protocol === "http") {
      for (const route of formData.routes) {
        const routeErrors: RouteError = {}
        const host = normalizeDomainValue(route.host ?? "")
        if (route.enabled && !host) {
          routeErrors.host = "Host is required for enabled routes"
        } else if (route.enabled && !isValidDomainValue(host)) {
          routeErrors.host = "Must be a valid domain such as example.com or *.example.com"
        }

        if (route.enabled && !route.path?.trim()) {
          routeErrors.path = "Path is required"
        } else if (route.path?.trim() && !route.path.trim().startsWith("/")) {
          routeErrors.path = "Path must start with /"
        }

        if (route.enabled && route.listener_protocol === "https" && !route.cert_id) {
          routeErrors.cert_id = "TLS certificate is required for HTTPS"
        }

        const requestTimeout = parseDurationMillis(route.timeouts?.request)
        const backendTimeout = parseDurationMillis(route.timeouts?.backend_request)
        if (Number.isNaN(requestTimeout) || Number.isNaN(backendTimeout)) {
          routeErrors.timeout = "Timeouts must use ms, s, m, or h units"
        } else if (requestTimeout !== null && backendTimeout !== null && backendTimeout > requestTimeout) {
          routeErrors.timeout = "Backend timeout must not exceed request timeout"
        }

        const backend = route.backends?.[0]
        if (!backend || backend.weight <= 0) {
          routeErrors.backend = "At least one backend weight must be positive"
        } else if (backend.weight > 1000000) {
          routeErrors.backend = "Backend weight must be between 1 and 1000000"
        }

        if (Object.keys(routeErrors).length > 0) {
          nextErrors.routes[route.form_id] = routeErrors
        }
      }
    }

    setErrors(nextErrors)
    return !nextErrors.port && !nextErrors.protocol && !nextErrors.node_port && Object.keys(nextErrors.routes).length === 0
  }

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    if (!validateForm()) {
      return
    }

    const payload: GatewaySpec = {
      ...(formData.id ? { id: formData.id } : {}),
      port: formData.port,
      protocol: formData.protocol,
      service_type: formData.service_type || "ClusterIP",
      ...(formData.service_type === "NodePort" && formData.node_port ? { node_port: formData.node_port } : {}),
    }

    if (formData.protocol === "http") {
      payload.routes = formData.routes.map(compactRoute)
    }

    saveMutation.mutate(payload)
  }

  const renderCertificateCombobox = (route: GatewayRouteFormState, index: number, routeErrors: RouteError) => (
    <Field>
      <FieldLabel>TLS Certificate *</FieldLabel>
      <FieldContent>
        <Combobox
          value={route.cert_id || null}
          onValueChange={(value: string | null) => updateRoute(route.form_id, (current) => ({ ...current, cert_id: value || undefined }))}
          itemToStringLabel={(value) => certificates.find((cert) => cert.id === value)?.label ?? value ?? ""}
        >
          <ComboboxInput placeholder="Select a certificate" aria-label={`Route certificate ${index + 1}`} aria-invalid={!!routeErrors.cert_id} />
          <ComboboxContent>
            <ComboboxList>
              {certificateGroups.map((group) => (
                <ComboboxGroup key={group.label}>
                  <ComboboxLabel>{group.label}</ComboboxLabel>
                  {group.items.map((cert) => (
                    <ComboboxItem key={cert.id} value={cert.id}>
                      <Item size="xs" className="p-0">
                        <ItemContent>
                          <ItemTitle className="whitespace-nowrap">
                            {cert.label}
                          </ItemTitle>
                          <ItemDescription>
                            {cert.description}
                          </ItemDescription>
                        </ItemContent>
                      </Item>
                    </ComboboxItem>
                  ))}
                </ComboboxGroup>
              ))}
            </ComboboxList>
          </ComboboxContent>
        </Combobox>
      </FieldContent>
      {routeErrors.cert_id && (
        <FieldError>
          <span className="text-destructive text-xs">{routeErrors.cert_id}</span>
        </FieldError>
      )}
    </Field>
  )

  const renderRouteEditor = (route: GatewayRouteFormState, index: number) => {
    const routeErrors = errors.routes[route.form_id] ?? {}
    const selectedDomainOption = inferSelectedDomainOption(route.host, domainOptions)
    const expanded = expandedRoutes[route.form_id] ?? true
    const backend = route.backends?.[0] ?? defaultBackend(app.id, formData.port)
    const certificateLabel = route.cert_id
      ? certificates.find((cert) => cert.id === route.cert_id)?.label ?? route.cert_id
      : "No certificate"

    return (
      <div key={route.form_id} className="rounded-md border bg-background">
        <div className="flex items-center justify-between gap-3 px-3 py-2">
          <div className="flex min-w-0 items-center gap-2">
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              aria-label={expanded ? `Collapse route ${index + 1}` : `Expand route ${index + 1}`}
              onClick={() => toggleRouteExpanded(route.form_id)}
            >
              {expanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
            </Button>
            <span className="rounded-sm border px-1.5 py-0.5 text-[0.625rem] font-medium uppercase text-muted-foreground">
              {route.listener_protocol || "http"}
            </span>
            <div className="min-w-0">
              <p className="truncate font-mono text-xs">{getRouteSummary(route)}</p>
              <p className="truncate text-xs text-muted-foreground">
                {route.listener_protocol === "https" ? certificateLabel : "Plain HTTP"} · {route.path_match_type || "PathPrefix"}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-1">
            <Checkbox
              aria-label={`Enable route ${index + 1}`}
              checked={route.enabled}
              onCheckedChange={(checked) => updateRoute(route.form_id, (current) => ({ ...current, enabled: !!checked }))}
            />
            <Button type="button" variant="ghost" size="icon-sm" aria-label={`Duplicate route ${index + 1}`} onClick={() => duplicateRoute(route)}>
              <Copy className="h-4 w-4" />
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              className="text-destructive hover:bg-destructive/10 hover:text-destructive"
              aria-label={`Delete route ${index + 1}`}
              onClick={() => deleteRoute(route.form_id)}
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          </div>
        </div>

        {expanded && (
          <div className="space-y-4 border-t px-3 py-3">
            <div className="grid grid-cols-3 gap-3">
              <Field className="col-span-2">
                <FieldLabel>Host *</FieldLabel>
                <FieldContent>
                  <Combobox
                    value={selectedDomainOption === "custom" ? null : selectedDomainOption}
                    onValueChange={(value: string | null) => {
                      if (!value) {
                        return
                      }

                      const nextOption = domainOptions.find((option) => option.value === value)
                      if (!nextOption) {
                        return
                      }

                      updateRoute(route.form_id, (current) => ({
                        ...current,
                        host: seedDomainInputFromSelection(nextOption.domain),
                      }))
                    }}
                    itemToStringLabel={(value) => domainOptions.find((option) => option.value === value)?.label ?? value ?? ""}
                  >
                    <ComboboxInput
                      aria-label={`Route host ${index + 1}`}
                      placeholder="app.example.com"
                      value={route.host}
                      onChange={(event) => updateRoute(route.form_id, (current) => ({ ...current, host: event.target.value }))}
                      aria-invalid={!!routeErrors.host}
                    >
                      <InputGroupAddon align="inline-start">
                        <InputGroupText>{route.listener_protocol || "http"}://</InputGroupText>
                      </InputGroupAddon>
                    </ComboboxInput>
                    <ComboboxContent>
                      <ComboboxList>
                        {domainOptionGroups.map((group) => (
                          <ComboboxGroup key={group.label}>
                            <ComboboxLabel>{group.label}</ComboboxLabel>
                            {group.items.map((option) => (
                              <ComboboxItem
                                key={option.value}
                                value={option.value}
                              >
                                <Item size="xs" className="p-0">
                                  <ItemContent>
                                    <ItemTitle className="whitespace-nowrap">
                                      {option.label}
                                    </ItemTitle>
                                    <ItemDescription>
                                      {option.domain}
                                    </ItemDescription>
                                  </ItemContent>
                                </Item>
                              </ComboboxItem>
                            ))}
                          </ComboboxGroup>
                        ))}
                      </ComboboxList>
                    </ComboboxContent>
                  </Combobox>
                </FieldContent>
                {routeErrors.host && (
                  <FieldError>
                    <span className="text-destructive text-xs">{routeErrors.host}</span>
                  </FieldError>
                )}
              </Field>

              <Field>
                <FieldLabel>Listener</FieldLabel>
                <FieldContent>
                  <Combobox
                    value={route.listener_protocol}
                    onValueChange={(value: string | null) => {
                      if (!value) return
                      updateRoute(route.form_id, (current) => ({
                        ...current,
                        listener_protocol: value,
                        cert_id: value === "https" ? current.cert_id : undefined,
                      }))
                    }}
                    itemToStringLabel={(value) => ROUTE_PROTOCOL_OPTIONS.find((option) => option.value === value)?.label ?? value ?? ""}
                  >
                    <ComboboxInput aria-label={`Route listener protocol ${index + 1}`} />
                    <ComboboxContent>
                      <ComboboxList>
                        {ROUTE_PROTOCOL_OPTIONS.map((option) => (
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

            <div className="grid grid-cols-3 gap-3">
              <Field>
                <FieldLabel>Path *</FieldLabel>
                <FieldContent>
                  <Input
                    aria-label={`Route path ${index + 1}`}
                    placeholder="/"
                    value={route.path}
                    onChange={(event) => updateRoute(route.form_id, (current) => ({ ...current, path: event.target.value }))}
                    aria-invalid={!!routeErrors.path}
                  />
                </FieldContent>
                {routeErrors.path && (
                  <FieldError>
                    <span className="text-destructive text-xs">{routeErrors.path}</span>
                  </FieldError>
                )}
              </Field>

              <Field>
                <FieldLabel>Path Match</FieldLabel>
                <FieldContent>
                  <Combobox
                    value={route.path_match_type}
                    onValueChange={(value: string | null) => {
                      if (!value) return
                      updateRoute(route.form_id, (current) => ({ ...current, path_match_type: value }))
                    }}
                    itemToStringLabel={(value) => PATH_MATCH_OPTIONS.find((option) => option.value === value)?.label ?? value ?? ""}
                  >
                    <ComboboxInput aria-label={`Route path match ${index + 1}`} />
                    <ComboboxContent>
                      <ComboboxList>
                        {PATH_MATCH_OPTIONS.map((option) => (
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

              {route.listener_protocol === "https" && renderCertificateCombobox(route, index, routeErrors)}
            </div>

            <Separator />

            <div className="space-y-3">
              <div>
                <h4 className="text-xs font-medium">Matching</h4>
                <p className="text-xs text-muted-foreground">Optional method and header match settings.</p>
              </div>
              <div className="grid grid-cols-3 gap-3">
                <Field>
                  <FieldLabel>Method</FieldLabel>
                  <FieldContent>
                    <Combobox
                      value={route.matches?.method || ""}
                      onValueChange={(value: string | null) => updateRoute(route.form_id, (current) => ({
                        ...current,
                        matches: {
                          ...current.matches,
                          method: value || undefined,
                        },
                      }))}
                      itemToStringLabel={(value) => METHOD_OPTIONS.find((option) => option.value === value)?.label ?? value ?? ""}
                    >
                      <ComboboxInput aria-label={`Route method ${index + 1}`} />
                      <ComboboxContent>
                        <ComboboxList>
                          {METHOD_OPTIONS.map((option) => (
                            <ComboboxItem key={option.value || "any"} value={option.value}>
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
                <Field>
                  <FieldLabel>Request Header</FieldLabel>
                  <FieldContent>
                    <Input
                      aria-label={`Route request header name ${index + 1}`}
                      placeholder="X-Feature"
                      value={route.filters?.request_headers?.set?.[0]?.name ?? ""}
                      onChange={(event) => updateRoute(route.form_id, (current) => ({
                        ...current,
                        filters: {
                          ...current.filters,
                          request_headers: {
                            ...current.filters?.request_headers,
                            set: [{
                              name: event.target.value,
                              value: current.filters?.request_headers?.set?.[0]?.value ?? "",
                            }],
                          },
                        },
                      }))}
                    />
                  </FieldContent>
                </Field>
                <Field>
                  <FieldLabel>Header Value</FieldLabel>
                  <FieldContent>
                    <Input
                      aria-label={`Route request header value ${index + 1}`}
                      placeholder="enabled"
                      value={route.filters?.request_headers?.set?.[0]?.value ?? ""}
                      onChange={(event) => updateRoute(route.form_id, (current) => ({
                        ...current,
                        filters: {
                          ...current.filters,
                          request_headers: {
                            ...current.filters?.request_headers,
                            set: [{
                              name: current.filters?.request_headers?.set?.[0]?.name ?? "",
                              value: event.target.value,
                            }],
                          },
                        },
                      }))}
                    />
                  </FieldContent>
                </Field>
              </div>
            </div>

            <Separator />

            <div className="space-y-3">
              <div>
                <h4 className="text-xs font-medium">Timeouts, Sessions, And Traffic</h4>
                <p className="text-xs text-muted-foreground">Request limits, cookie stickiness, and backend weighting.</p>
              </div>
              <div className="grid grid-cols-4 gap-3">
                <Field>
                  <FieldLabel>Request Timeout</FieldLabel>
                  <FieldContent>
                    <Input
                      aria-label={`Route request timeout ${index + 1}`}
                      placeholder="30s"
                      value={route.timeouts?.request ?? ""}
                      onChange={(event) => updateRoute(route.form_id, (current) => ({
                        ...current,
                        timeouts: {
                          ...current.timeouts,
                          request: event.target.value,
                        },
                      }))}
                      aria-invalid={!!routeErrors.timeout}
                    />
                  </FieldContent>
                </Field>
                <Field>
                  <FieldLabel>Backend Timeout</FieldLabel>
                  <FieldContent>
                    <Input
                      aria-label={`Route backend timeout ${index + 1}`}
                      placeholder="25s"
                      value={route.timeouts?.backend_request ?? ""}
                      onChange={(event) => updateRoute(route.form_id, (current) => ({
                        ...current,
                        timeouts: {
                          ...current.timeouts,
                          backend_request: event.target.value,
                        },
                      }))}
                      aria-invalid={!!routeErrors.timeout}
                    />
                  </FieldContent>
                </Field>
                <Field>
                  <FieldLabel>Cookie Session</FieldLabel>
                  <FieldContent>
                    <Input
                      aria-label={`Route cookie session ${index + 1}`}
                      placeholder="ketches_route"
                      value={route.session_persistence?.session_name ?? ""}
                      onChange={(event) => updateRoute(route.form_id, (current) => ({
                        ...current,
                        session_persistence: {
                          ...current.session_persistence,
                          type: "Cookie",
                          cookie_lifetime_type: "Session",
                          session_name: event.target.value,
                        },
                      }))}
                    />
                  </FieldContent>
                </Field>
                <Field>
                  <FieldLabel>Weight</FieldLabel>
                  <FieldContent>
                    <Input
                      aria-label={`Route backend weight ${index + 1}`}
                      type="number"
                      min={0}
                      max={1000000}
                      value={backend.weight}
                      onChange={(event) => {
                        const weight = Number.parseInt(event.target.value, 10)
                        updateRoute(route.form_id, (current) => ({
                          ...current,
                          backends: [{
                            ...backend,
                            backend_app_id: backend.backend_app_id || app.id,
                            backend_port: backend.backend_port || formData.port,
                            weight: Number.isNaN(weight) ? 0 : weight,
                          }],
                        }))
                      }}
                      aria-invalid={!!routeErrors.backend}
                    />
                  </FieldContent>
                </Field>
              </div>
              {(routeErrors.timeout || routeErrors.backend) && (
                <FieldError>
                  <span className="text-destructive text-xs">{routeErrors.timeout || routeErrors.backend}</span>
                </FieldError>
              )}
            </div>

            <Separator />

            <div className="space-y-3">
              <div>
                <h4 className="text-xs font-medium">Provider Extensions</h4>
                <p className="text-xs text-muted-foreground">Controller-specific options are stored with the route and validated by the gateway provider.</p>
              </div>
              <div className="grid grid-cols-3 gap-3">
                <Field>
                  <FieldLabel>Request Body Size</FieldLabel>
                  <FieldContent>
                    <Input
                      aria-label={`Route request body size ${index + 1}`}
                      placeholder="10Mi"
                      value={route.extension?.request_body_size ?? ""}
                      onChange={(event) => updateRoute(route.form_id, (current) => ({
                        ...current,
                        extension: {
                          ...current.extension,
                          request_body_size: event.target.value,
                        },
                      }))}
                    />
                  </FieldContent>
                </Field>
                <Field className="justify-end">
                  <FieldLabel>KeepAlive</FieldLabel>
                  <FieldContent>
                    <label className="flex items-center gap-2 text-xs">
                      <Checkbox
                        aria-label={`Route keep alive ${index + 1}`}
                        checked={!!route.extension?.keep_alive}
                        onCheckedChange={(checked) => updateRoute(route.form_id, (current) => ({
                          ...current,
                          extension: {
                            ...current.extension,
                            keep_alive: !!checked,
                          },
                        }))}
                      />
                      Enabled
                    </label>
                  </FieldContent>
                </Field>
                <Field className="justify-end">
                  <FieldLabel>WebSocket</FieldLabel>
                  <FieldContent>
                    <label className="flex items-center gap-2 text-xs">
                      <Checkbox
                        aria-label={`Route websocket ${index + 1}`}
                        checked={!!route.extension?.websocket}
                        onCheckedChange={(checked) => updateRoute(route.form_id, (current) => ({
                          ...current,
                          extension: {
                            ...current.extension,
                            websocket: !!checked,
                          },
                        }))}
                      />
                      Enabled
                    </label>
                  </FieldContent>
                </Field>
              </div>
            </div>
          </div>
        )}
      </div>
    )
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="sm:max-w-200 max-h-[90vh] overflow-y-auto">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>{isEditing ? 'Edit Gateway' : 'Add Gateway'}</DialogTitle>
            <DialogDescription>
              Configure an application port and its HTTP routes.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-5 py-4">
            <div className="space-y-3">
              <div className="flex items-center justify-between gap-3">
                <div>
                  <h3 className="text-sm font-medium">Port Settings</h3>
                  <p className="text-xs text-muted-foreground">The service port exposed by this application gateway.</p>
                </div>
                {formData.internal_address && (
                  <span className="rounded-sm border px-2 py-1 font-mono text-xs text-muted-foreground">
                    {formData.internal_address}
                  </span>
                )}
              </div>
              <div className="grid grid-cols-3 gap-4">
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
                      <TooltipContent side="top" align="start" className="max-w-100 flex-col items-start">
                        <p className="text-xs">- The port your application listens on inside the container.</p>
                      </TooltipContent>
                    </Tooltip>
                  </FieldLabel>
                  <FieldContent>
                    <Input
                      aria-label="Gateway container port"
                      type="number"
                      placeholder="80"
                      value={formData.port}
                      onChange={(event) => {
                        const nextPort = Number.parseInt(event.target.value, 10) || 0
                        setFormData((prev) => ({
                          ...prev,
                          port: nextPort,
                          routes: prev.routes.map((route) => ({
                            ...route,
                            backends: (route.backends && route.backends.length > 0 ? route.backends : [defaultBackend(app.id, nextPort)])
                              .map((backend) => ({
                                ...backend,
                                backend_port: backend.backend_port === prev.port ? nextPort : backend.backend_port,
                              })),
                          })),
                        }))
                      }}
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
                        if (!value) return
                        setFormData((prev) => ({
                          ...prev,
                          protocol: value,
                          gateway_port: value === "http" ? undefined : prev.gateway_port,
                        }))
                      }}
                      itemToStringLabel={(value) => GATEWAY_PROTOCOL_OPTIONS.find((option) => option.value === value)?.label ?? value ?? ""}
                    >
                      <ComboboxInput aria-label="Gateway protocol" />
                      <ComboboxContent>
                        <ComboboxList>
                          {GATEWAY_PROTOCOL_OPTIONS.map((option) => (
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
                  {errors.protocol && (
                    <FieldError>
                      <span className="text-destructive text-xs">{errors.protocol}</span>
                    </FieldError>
                  )}
                </Field>

                <Field>
                  <FieldLabel>Service Type</FieldLabel>
                  <FieldContent>
                    <Combobox
                      value={formData.service_type || "ClusterIP"}
                      onValueChange={(value: string | null) => {
                        const serviceType = value || "ClusterIP"
                        setFormData((prev) => ({
                          ...prev,
                          service_type: serviceType,
                          node_port: serviceType === "NodePort" ? prev.node_port : undefined,
                        }))
                      }}
                      itemToStringLabel={(value) => SERVICE_TYPE_OPTIONS.find((option) => option.value === value)?.label ?? value ?? ""}
                    >
                      <ComboboxInput aria-label="Gateway service type" />
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

              {formData.service_type === "NodePort" && (
                <Field className="max-w-60">
                  <FieldLabel>NodePort</FieldLabel>
                  <FieldContent>
                    <Input
                      aria-label="Gateway node port"
                      type="number"
                      placeholder="Auto"
                      min={30000}
                      max={32767}
                      value={formData.node_port || ""}
                      onChange={(event) => setFormData((prev) => ({ ...prev, node_port: Number.parseInt(event.target.value, 10) || undefined }))}
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

            <Separator />

            {formData.protocol === "http" ? (
              <div className="space-y-3">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <h3 className="text-sm font-medium">HTTP Routes</h3>
                    <p className="text-xs text-muted-foreground">Each route can choose HTTP or HTTPS, certificate, matching, timeouts, and backend weight.</p>
                    {!gatewayAPIInstalled && (
                      <p className="mt-1 text-xs text-orange-600">Gateway API is not installed on this cluster, so saved routes may not sync until it is installed.</p>
                    )}
                  </div>
                  <Button type="button" variant="secondary" onClick={addRoute}>
                    <Plus className="h-4 w-4" />
                    Add Route
                  </Button>
                </div>

                {formData.routes.length === 0 ? (
                  <div className="rounded-md border border-dashed p-4 text-xs text-muted-foreground">
                    No HTTP routes configured.
                  </div>
                ) : (
                  <div className="space-y-3">
                    {formData.routes.map((route, index) => renderRouteEditor(route, index))}
                  </div>
                )}
              </div>
            ) : (
              <div className="rounded-md border border-dashed p-4 text-xs text-muted-foreground">
                HTTP routes are kept as a draft and are omitted from submit while the gateway protocol is {formData.protocol?.toUpperCase()}.
              </div>
            )}

            {errors.form && (
              <div className="rounded-md border border-destructive/40 bg-destructive/5 px-3 py-2 text-xs text-destructive">
                {errors.form}
              </div>
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
    </Dialog>
  )
}
