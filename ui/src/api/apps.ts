import client from './client'
import { type PaginationParams, type PaginationResponse } from './pagination'

export interface SimpleApp {
  id: string
  name: string
  slug: string
  description: string
  status: string
  code_repository_id: string
}

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
  type: 'liveness' | 'readiness' | 'startup'
  probe_mode: 'httpGet' | 'tcpSocket' | 'exec'
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

export interface GatewaySpec {
  id?: string
  port: number
  protocol: string
  domain: string
  path: string
  gateway_port?: number
  service_type?: string
  node_port?: number
  gateway_ip?: string
  internal_address?: string
  exposed: boolean
  cert_id?: string
}

export interface App {
  id: string
  slug: string
  name: string
  description: string
  env_id: string
  app_type: string
  code_repository_id?: string
  container_image: string
  image_pull_policy?: string
  container_command?: string
  registry_username?: string
  registry_password?: string
  replicas: number
  request_cpu: number
  request_memory: number
  limit_cpu: number
  limit_memory: number
  status: string
  available_actions: ActionMetadata[]
  auto_scaling?: AutoScalingSpec
  scheduling_rule?: SchedulingSpec
  probes?: ProbeSpec[]
  gateways?: GatewaySpec[]
  created_at: string
  env?: {
    id: string
    name: string
    project_id: string
  }
}

export interface AppInstance {
  instance_name: string
  status: string
  ip: string
  init_container_count: number
  init_containers: string[]
  container_count: number
  containers: string[]
  node_name: string
  node_ip: string
  restart_count: number
  running_duration: string
  created_at: string
}

export interface AppEvent {
  type: string
  reason: string
  message: string
  from: string
  count: number
  created_at: string
}

export interface ActionMetadata {
  action: string
  label: string
  icon: string
  category: 'primary' | 'secondary'
  variant: 'default' | 'destructive' | 'outline'
}

export interface AvailableActionsResponse {
  actions: ActionMetadata[]
}

export const appsApi = {
  list: async (envId: string, params?: PaginationParams) => {
    return client.get(`/v1/envs/${envId}/apps`, {
      params
    }) as Promise<{ items: App[], pagination: PaginationResponse }>
  },
  listSimple: async (envId: string) => {
    return client.get(`/v1/envs/${envId}/apps/simple`) as Promise<SimpleApp[]>
  },
  create: async (envId: string, data: any) => {
    return client.post(`/v1/envs/${envId}/apps`, data) as Promise<App>
  },
  get: async (id: string) => {
    return client.get(`/v1/apps/${id}`) as Promise<App>
  },
  delete: async (id: string) => {
    return client.delete(`/v1/apps/${id}`)
  },
  batchDelete: async (ids: string[]) => {
    return client.post('/v1/apps/batch-delete', { ids })
  },
  updateBasic: async (id: string, data: Partial<App>) => {
    return client.patch(`/v1/apps/${id}/basic`, data) as Promise<App>
  },
  updateImage: async (id: string, data: { container_image: string, image_pull_policy?: string, registry_username?: string, registry_password?: string }) => {
    return client.patch(`/v1/apps/${id}/image`, data) as Promise<App>
  },
  listImageTags: async (id: string) => {
    return client.get(`/v1/apps/${id}/image-tags`) as Promise<{ repository: string, current_tag: string, tags: string[] }>
  },
  updateReplicas: async (id: string, replicas: number) => {
    return client.patch(`/v1/apps/${id}/replicas`, { replicas }) as Promise<App>
  },
  updateResources: async (id: string, data: { request_cpu: number, request_memory: number, limit_cpu: number, limit_memory: number }) => {
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
    return client.get(`/v1/apps/${id}/volumes`) as Promise<any[]>
  },
  addVolume: async (id: string, data: any) => {
    return client.post(`/v1/apps/${id}/volumes`, data)
  },
  updateVolume: async (volumeId: string, data: any) => {
    return client.put(`/v1/volumes/${volumeId}`, data)
  },
  deleteVolume: async (volumeId: string) => {
    return client.delete(`/v1/volumes/${volumeId}`)
  },
  listEnvVars: async (id: string) => {
    return client.get(`/v1/apps/${id}/env-vars`) as Promise<any[]>
  },
  addEnvVar: async (id: string, data: any) => {
    return client.post(`/v1/apps/${id}/env-vars`, data)
  },
  updateEnvVar: async (varId: string, data: any) => {
    return client.put(`/v1/env-vars/${varId}`, data)
  },
  deleteEnvVar: async (varId: string) => {
    return client.delete(`/v1/env-vars/${varId}`)
  },
  listConfigFiles: async (id: string) => {
    return client.get(`/v1/apps/${id}/config-files`) as Promise<any[]>
  },
  addConfigFile: async (id: string, data: any) => {
    return client.post(`/v1/apps/${id}/config-files`, data)
  },
  updateConfigFile: async (id: string, data: any) => {
    return client.put(`/v1/config-files/${id}`, data)
  },
  deleteConfigFile: async (id: string) => {
    return client.delete(`/v1/config-files/${id}`)
  },
  listGateways: async (id: string) => {
    return client.get(`/v1/apps/${id}/gateways`) as Promise<any[]>
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
    return client.get(`/v1/apps/${id}/topology`) as Promise<{
      nodes: {
        id: string
        type: string
        name: string
        status?: string
        metadata?: Record<string, string>
      }[]
      edges: {
        source: string
        target: string
        type?: string
      }[]
    }>
  },
  getTopologyResourceYaml: async (appId: string, nodeId: string) => {
    return client.get(`/v1/apps/${appId}/topology/nodes/${nodeId}/resource-yaml`) as Promise<{ yaml: string }>
  },

  importApps: async (envId: string, data: {
    type: 'dockercompose' | 'kubernetes' | 'ketches'
    content: string
    conflict_strategy?: 'rename' | 'ask' | 'error'
  }) => {
    return client.post(`/v1/envs/${envId}/apps/import`, data) as Promise<{
      imported: { name: string, slug: string, status: string }[]
      conflicts: any[]
    }>
  },

  exportApps: async (appId: string, format: 'kubernetes' | 'ketches' | 'helm' | 'dockercompose') => {
    return client.get(`/v1/apps/${appId}/export`, {
      params: { format }
    }) as Promise<{ yaml?: string, metadata?: string, chart?: string, compose?: string }>
  },

  exportEnvApps: async (envId: string, format: 'kubernetes' | 'ketches' | 'helm' | 'dockercompose', appIds?: string[]) => {
    return client.get(`/v1/envs/${envId}/apps/export`, {
      params: { format, app_ids: appIds?.join(',') }
    }) as Promise<{ yaml?: string, metadata?: string, chart?: string, compose?: string }>
  }
}
