import type { App } from './apps'
import client from './client'
import { type PaginationParams, type PaginationResponse, type SimpleResponse } from './pagination'

export interface Plugin {
  id: string
  slug: string
  name: string
  description: string
  image: string
  registry_username: string
  command: string
  env_vars: { key: string, value: string }[]
  plugin_type: "init" | "sidecar"
  install_count: number
  created_at: string
  updated_at: string
}

export interface AppPlugin {
  id: string
  app_id: string
  plugin_id: string
  enabled: boolean
  env_vars?: { key: string, value: string }[]
  request_cpu?: number
  limit_cpu?: number
  request_memory?: number
  limit_memory?: number
  plugin: Plugin
  created_at: string
}

export interface CreatePluginRequest {
  slug: string
  name: string
  description: string
  image: string
  registry_username?: string
  command?: string
  env_vars?: { key: string, value: string }[]
  plugin_type: "init" | "sidecar"
}

export interface UpdatePluginRequest {
  name?: string
  description?: string
  image?: string
  registry_username?: string
  command?: string
  env_vars?: { key: string, value: string }[]
  plugin_type?: "init" | "sidecar"
}

export const pluginsApi = {
  // Global Plugin Management (Admin)
  listPlugins: async (params?: PaginationParams) => {
    return client.get('/v1/plugins', {
      params
    }) as Promise<{ items: Plugin[], pagination: PaginationResponse }>
  },

  listPluginsSimple: async (projectId?: string) => {
    const url = projectId ? `/v1/projects/${projectId}/plugins/simple` : '/v1/plugins/simple'
    return client.get(url) as Promise<SimpleResponse[]>
  },

  getPlugin: async (pluginID: string) => {
    return client.get(`/v1/plugins/${pluginID}`) as Promise<Plugin>
  },

  createPlugin: async (data: CreatePluginRequest) => {
    return client.post('/v1/plugins', data) as Promise<Plugin>
  },

  updatePlugin: async (pluginID: string, data: UpdatePluginRequest) => {
    return client.put(`/v1/plugins/${pluginID}`, data) as Promise<Plugin>
  },

  deletePlugin: async (pluginID: string) => {
    return client.delete(`/v1/plugins/${pluginID}`)
  },

  // App Plugin Installation (Developer+)
  listAppPlugins: async (appID: string) => {
    return client.get(`/v1/apps/${appID}/plugins`) as Promise<AppPlugin[]>
  },

  installPlugin: async (appID: string, pluginID: string, envVars?: { key: string, value: string }[]) => {
    return client.post(`/v1/apps/${appID}/plugins`, { plugin_id: pluginID, env_vars: envVars }) as Promise<AppPlugin>
  },

  uninstallPlugin: async (appID: string, pluginID: string) => {
    return client.delete(`/v1/apps/${appID}/plugins/${pluginID}`)
  },

  togglePlugin: async (appID: string, pluginID: string, enabled: boolean) => {
    return client.patch(`/v1/apps/${appID}/plugins/${pluginID}/toggle`, { enabled })
  },

  updateAppPluginEnv: async (appID: string, pluginID: string, env_vars: { key: string, value: string }[]) => {
    return client.patch(`/v1/apps/${appID}/plugins/${pluginID}/env`, { env_vars })
  },

  updateAppPluginResources: async (appID: string, pluginID: string, resources: {
    request_cpu?: number
    limit_cpu?: number
    request_memory?: number
    limit_memory?: number
  }) => {
    return client.patch(`/v1/apps/${appID}/plugins/${pluginID}/resources`, resources)
  },

  getPluginInstalledApps: async (pluginID: string) => {
    return client.get(`/v1/plugins/${pluginID}/installed-apps`) as Promise<App[]>
  }
}
