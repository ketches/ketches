import client from './client'

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

export const appGroupsApi = {
  list: (envId: string): Promise<AppGroupWithApps[]> =>
    client.get(`/v1/envs/${envId}/app-groups`),
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
