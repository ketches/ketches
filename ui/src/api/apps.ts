import client from "./client"
import type { Env } from "./envs"
import type {
  OperationRequestBody,
  OperationResponseData,
  WithRequired,
} from "./generated/helpers"
import type { PaginationParams, PaginationResponse } from "./pagination"

type GeneratedSimpleApp = OperationResponseData<"/api/v1/envs/{envID}/apps/simple", "get">[number]
type GeneratedApp = OperationResponseData<"/api/v1/apps/{appID}", "get">
type GeneratedAppInstance = OperationResponseData<"/api/v1/apps/{appID}/instances", "get">[number]
type GeneratedAppEvent = OperationResponseData<
  "/api/v1/apps/{appID}/instances/{instanceName}/events",
  "get"
>[number]
type GeneratedAppVolume = OperationResponseData<"/api/v1/apps/{appID}/volumes", "get">[number]
type GeneratedAppEnvVar = OperationResponseData<"/api/v1/apps/{appID}/env-vars", "get">[number]
type GeneratedAppConfigFile = OperationResponseData<"/api/v1/apps/{appID}/config-files", "get">[number]
type GeneratedActionMetadata = NonNullable<
  OperationResponseData<"/api/v1/apps/{appID}/available-actions", "get">["actions"]
>[number]

export type SimpleApp = WithRequired<
  GeneratedSimpleApp,
  "id" | "name" | "slug" | "description" | "status"
>

export interface AutoScalingSpec {
  min_replicas: number
  max_replicas: number
  target_cpu_utilization?: number
  target_memory_utilization?: number
}

export interface SchedulingSpec {
  rule_type: string
  node_name?: string
  node_selector?: string
  node_affinity?: string
  tolerations?: string
}

export interface ProbeSpec {
  type: "liveness" | "readiness" | "startup"
  probe_mode: "httpGet" | "tcpSocket" | "exec"
  enabled: boolean
  http_get_path?: string
  http_get_port?: number
  tcp_socket_port?: number
  exec_command?: string
  initial_delay_seconds: number
  period_seconds: number
  timeout_seconds: number
  success_threshold: number
  failure_threshold: number
}

export interface GatewayHTTPMatch {
  name: string
  type: string
  value: string
}

export interface GatewayHeaderValue {
  name: string
  value: string
}

export interface GatewayHeaderModifier {
  set?: GatewayHeaderValue[]
  add?: GatewayHeaderValue[]
  remove?: string[]
}

export interface GatewayRouteFilters {
  request_headers?: GatewayHeaderModifier
  response_headers?: GatewayHeaderModifier
}

export interface GatewayRouteMatches {
  method?: string
  headers?: GatewayHTTPMatch[]
  query_params?: GatewayHTTPMatch[]
}

export interface GatewayRouteTimeouts {
  request?: string
  backend_request?: string
}

export interface GatewayRouteRetry {
  attempts?: number
  backoff?: string
  codes?: number[]
}

export interface GatewaySessionPersistence {
  type?: string
  session_name?: string
  cookie_lifetime_type?: string
  absolute_timeout?: string
  idle_timeout?: string
}

export interface GatewayRouteExtension {
  request_body_size?: string
  keep_alive?: boolean
  websocket?: boolean
}

export interface GatewayRouteBackendSpec {
  id?: string
  route_id?: string
  backend_app_id?: string
  backend_app_slug?: string
  backend_port: number
  weight: number
}

export interface GatewayRouteSpec {
  id?: string
  gateway_id?: string
  host: string
  listener_protocol: "http" | "https" | string
  path: string
  path_match_type: "PathPrefix" | "Exact" | string
  enabled: boolean
  cert_id?: string | null
  matches?: GatewayRouteMatches
  filters?: GatewayRouteFilters
  timeouts?: GatewayRouteTimeouts
  retry?: GatewayRouteRetry
  session_persistence?: GatewaySessionPersistence
  extension?: GatewayRouteExtension
  backends?: GatewayRouteBackendSpec[]
  sort_order?: number
}

export interface GatewaySpec {
  id?: string
  app_id?: string
  port: number
  protocol: "http" | "tcp" | "udp" | string
  gateway_port?: number
  service_type?: "ClusterIP" | "NodePort" | string
  node_port?: number
  gateway_host?: string
  internal_address?: string
  routes?: GatewayRouteSpec[]
}

export type AppCreateRequest = OperationRequestBody<"/api/v1/envs/{envID}/apps", "post">

export type AppVolume = WithRequired<
  GeneratedAppVolume,
  "slug" | "volume_type" | "mount_path"
>
export interface AppVolumeRequest {
  id?: string
  slug: string
  volume_type: string
  status?: string
  mount_path: string
  sub_path?: string
  storage_class?: string
  capacity?: number
  access_modes?: string
  volume_mode?: string
}

export type AppEnvVar = WithRequired<GeneratedAppEnvVar, "key" | "value">
export type AppEnvVarRequest = OperationRequestBody<"/api/v1/apps/{appID}/env-vars", "post">

export type AppConfigFile = WithRequired<
  GeneratedAppConfigFile,
  "slug" | "mount_path" | "content"
>
export type AppConfigFileRequest = OperationRequestBody<"/api/v1/apps/{appID}/config-files", "post">

export type AppImportResponse = OperationResponseData<"/api/v1/envs/{envID}/apps/import", "post">
export type AppImportConflict = NonNullable<AppImportResponse["conflicts"]>[number]
export type AppExportResponse = OperationResponseData<"/api/v1/apps/{appID}/export", "get">

export type App = WithRequired<
  GeneratedApp,
  | "id"
  | "slug"
  | "name"
  | "description"
  | "env_id"
  | "app_type"
  | "container_image"
  | "replicas"
  | "request_cpu"
  | "request_memory"
  | "limit_cpu"
  | "limit_memory"
  | "status"
  | "created_at"
> & {
  available_actions: ActionMetadata[]
  auto_scaling?: AutoScalingSpec | null
  scheduling_rule?: SchedulingSpec | null
  probes?: ProbeSpec[]
  gateways?: GatewaySpec[]
  env?: Env
  registry_password?: string
}

export type AppInstance = WithRequired<
  GeneratedAppInstance,
  | "instance_name"
  | "status"
  | "ip"
  | "init_container_count"
  | "init_containers"
  | "container_count"
  | "containers"
  | "node_name"
  | "node_ip"
  | "restart_count"
  | "running_duration"
  | "created_at"
>

export type AppEvent = WithRequired<
  GeneratedAppEvent,
  "type" | "reason" | "message" | "from" | "count" | "created_at"
>

export type ActionMetadata = WithRequired<
  GeneratedActionMetadata,
  "action" | "label" | "icon" | "category" | "variant"
> & {
  category: "primary" | "secondary"
  variant: "default" | "destructive" | "outline"
}

export interface AvailableActionsResponse {
  actions: ActionMetadata[]
}

type AppsListResponse = {
  items: App[]
  pagination: PaginationResponse
}

type AppImageTagsResponse = WithRequired<
  OperationResponseData<"/api/v1/apps/{appID}/image-tags", "get">,
  "repository" | "current_tag" | "tags"
>
interface AppTopologyNode {
  id: string
  type: string
  name: string
  status?: string
  metadata?: Record<string, string>
}

interface AppTopologyEdge {
  source: string
  target: string
  type?: string
}

interface AppTopologyResponse {
  nodes: AppTopologyNode[]
  edges: AppTopologyEdge[]
}

interface AppTopologyResourceYamlResponse {
  yaml: string
}

export const appsApi = {
  list: async (envId: string, params?: PaginationParams) => {
    return client.get(`/v1/envs/${envId}/apps`, {
      params,
    }) as Promise<AppsListResponse>
  },
  listSimple: async (envId: string) => {
    return client.get(`/v1/envs/${envId}/apps/simple`) as Promise<SimpleApp[]>
  },
  create: async (envId: string, data: AppCreateRequest) => {
    return client.post(`/v1/envs/${envId}/apps`, data) as Promise<App>
  },
  get: async (id: string) => {
    return client.get(`/v1/apps/${id}`) as Promise<App>
  },
  delete: async (id: string) => {
    return client.delete(`/v1/apps/${id}`)
  },
  batchDelete: async (ids: string[]) => {
    return client.post("/v1/apps/batch-delete", { ids })
  },
  updateBasic: async (id: string, data: Partial<App>) => {
    return client.patch(`/v1/apps/${id}/basic`, data) as Promise<App>
  },
  updateImage: async (
    id: string,
    data: OperationRequestBody<"/api/v1/apps/{appID}/image", "patch">
  ) => {
    return client.patch(`/v1/apps/${id}/image`, data) as Promise<App>
  },
  listImageTags: async (id: string) => {
    return client.get(`/v1/apps/${id}/image-tags`) as Promise<AppImageTagsResponse>
  },
  updateReplicas: async (id: string, replicas: number) => {
    return client.patch(`/v1/apps/${id}/replicas`, { replicas }) as Promise<App>
  },
  updateResources: async (
    id: string,
    data: OperationRequestBody<"/api/v1/apps/{appID}/resources", "patch">
  ) => {
    return client.patch(`/v1/apps/${id}/resources`, data) as Promise<App>
  },
  updateAutoScaling: async (id: string, auto_scaling: AutoScalingSpec | null) => {
    return client.patch(`/v1/apps/${id}/auto-scaling`, { auto_scaling }) as Promise<App>
  },
  updateHealth: async (id: string, probes: ProbeSpec[]) => {
    return client.patch(`/v1/apps/${id}/health`, { probes }) as Promise<App>
  },
  updateScheduling: async (id: string, scheduling_rule: SchedulingSpec | null) => {
    return client.patch(`/v1/apps/${id}/scheduling`, { scheduling_rule }) as Promise<App>
  },
  updateCommand: async (id: string, container_command: string) => {
    return client.patch(`/v1/apps/${id}/command`, { container_command }) as Promise<App>
  },
  getAvailableActions: async (id: string) => {
    return client.get(`/v1/apps/${id}/available-actions`) as Promise<AvailableActionsResponse>
  },
  executeAction: async (id: string, action: string) => {
    return client.post(`/v1/apps/${id}/action`, { action })
  },
  listInstances: async (id: string) => {
    return client.get(`/v1/apps/${id}/instances`) as Promise<AppInstance[]>
  },
  deleteInstance: async (appId: string, instanceName: string) => {
    return client.delete(`/v1/apps/${appId}/instances/${instanceName}`)
  },
  listInstanceEvents: async (appId: string, instanceName: string) => {
    return client.get(`/v1/apps/${appId}/instances/${instanceName}/events`) as Promise<AppEvent[]>
  },
  listVolumes: async (id: string) => {
    return client.get(`/v1/apps/${id}/volumes`) as Promise<AppVolume[]>
  },
  addVolume: async (id: string, data: AppVolumeRequest) => {
    return client.post(`/v1/apps/${id}/volumes`, data)
  },
  updateVolume: async (volumeId: string, data: AppVolumeRequest) => {
    return client.put(`/v1/volumes/${volumeId}`, data)
  },
  deleteVolume: async (volumeId: string) => {
    return client.delete(`/v1/volumes/${volumeId}`)
  },
  listEnvVars: async (id: string) => {
    return client.get(`/v1/apps/${id}/env-vars`) as Promise<AppEnvVar[]>
  },
  addEnvVar: async (id: string, data: AppEnvVarRequest) => {
    return client.post(`/v1/apps/${id}/env-vars`, data)
  },
  updateEnvVar: async (varId: string, data: AppEnvVarRequest) => {
    return client.put(`/v1/env-vars/${varId}`, data)
  },
  deleteEnvVar: async (varId: string) => {
    return client.delete(`/v1/env-vars/${varId}`)
  },
  listConfigFiles: async (id: string) => {
    return client.get(`/v1/apps/${id}/config-files`) as Promise<AppConfigFile[]>
  },
  addConfigFile: async (id: string, data: AppConfigFileRequest) => {
    return client.post(`/v1/apps/${id}/config-files`, data)
  },
  updateConfigFile: async (id: string, data: AppConfigFileRequest) => {
    return client.put(`/v1/config-files/${id}`, data)
  },
  deleteConfigFile: async (id: string) => {
    return client.delete(`/v1/config-files/${id}`)
  },
  listGateways: async (id: string) => {
    return client.get(`/v1/apps/${id}/gateways`) as Promise<GatewaySpec[]>
  },
  addGateway: async (id: string, data: GatewaySpec) => {
    return client.post(`/v1/apps/${id}/gateways`, data)
  },
  updateGateway: async (gatewayId: string, data: GatewaySpec) => {
    return client.put(`/v1/gateways/${gatewayId}`, data)
  },
  deleteGateway: async (gatewayId: string) => {
    return client.delete(`/v1/gateways/${gatewayId}`)
  },
  getTopology: async (id: string) => {
    return client.get(`/v1/apps/${id}/topology`) as Promise<AppTopologyResponse>
  },
  getTopologyResourceYaml: async (appId: string, nodeId: string) => {
    return client.get(`/v1/apps/${appId}/topology/nodes/${nodeId}/resource-yaml`) as Promise<AppTopologyResourceYamlResponse>
  },
  importApps: async (
    envId: string,
    data: OperationRequestBody<"/api/v1/envs/{envID}/apps/import", "post">
  ) => {
    return client.post(`/v1/envs/${envId}/apps/import`, data) as Promise<AppImportResponse>
  },
  exportApps: async (
    appId: string,
    format: "kubernetes" | "ketches" | "helm" | "dockercompose"
  ) => {
    return client.get(`/v1/apps/${appId}/export`, {
      params: { format },
    }) as Promise<AppExportResponse>
  },
  exportEnvApps: async (
    envId: string,
    format: "kubernetes" | "ketches" | "helm" | "dockercompose",
    appIds?: string[]
  ) => {
    return client.get(`/v1/envs/${envId}/apps/export`, {
      params: { format, app_ids: appIds?.join(",") },
    }) as Promise<AppExportResponse>
  },
}
