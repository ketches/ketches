import client from './client'
import { type App } from './apps'
import { type PaginationParams, type PaginationResponse } from './pagination'

export interface AppGroup {
  id: string
  env_id: string
  name: string
  description: string
  created_at: string
}

export interface AppGroupWithApps extends AppGroup {
  apps: Array<{ id: string; slug: string; name: string; status: string }>
}
export interface GroupAppsResponse {
  items: App[]
  pagination: PaginationResponse
}

export const appGroupsApi = {
  list: (envId: string): Promise<AppGroupWithApps[]> =>
    client.get(`/v1/envs/${envId}/app-groups`),
  listGroupApps: (groupId: string, params?: PaginationParams): Promise<GroupAppsResponse> =>
    client.get(`/v1/app-groups/${groupId}/apps`, { params }),
  listSpecificApps: (groupId: string): Promise<AppGroupWithApps> =>
    client.get(`/v1/app-groups/${groupId}/apps`),
  create: (envId: string, data: { name: string; description?: string }): Promise<AppGroup> =>
    client.post(`/v1/envs/${envId}/app-groups`, data),
  update: (groupId: string, data: { name: string; description?: string }): Promise<AppGroup> =>
    client.put(`/v1/app-groups/${groupId}`, data),
  delete: (groupId: string): Promise<void> =>
    client.delete(`/v1/app-groups/${groupId}`),
  addApp: (groupId: string, appId: string): Promise<void> =>
    client.post(`/v1/app-groups/${groupId}/apps/${appId}`),
  removeApp: (groupId: string, appId: string): Promise<void> =>
    client.delete(`/v1/app-groups/${groupId}/apps/${appId}`),
}
