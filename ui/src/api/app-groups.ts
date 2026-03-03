import client from './client'

export interface AppGroup {
  id: string
  project_id: string
  name: string
  description: string
  created_at: string
}

export interface AppGroupWithApps extends AppGroup {
  apps: Array<{ id: string; slug: string; name: string; status: string }>
}

export interface AppFavorite {
  id: string
  app_id: string
}

export const appGroupsApi = {
  list: (projectId: string): Promise<AppGroup[]> =>
    client.get(`/v1/projects/${projectId}/app-groups`),
  listGrouped: (projectId: string, envId: string): Promise<AppGroupWithApps[]> =>
    client.get(`/v1/projects/${projectId}/envs/${envId}/grouped-apps`),
  create: (projectId: string, data: { name: string; description?: string }): Promise<AppGroup> =>
    client.post(`/v1/projects/${projectId}/app-groups`, data),
  update: (groupId: string, data: { name: string; description?: string }): Promise<AppGroup> =>
    client.put(`/v1/app-groups/${groupId}`, data),
  delete: (groupId: string): Promise<void> =>
    client.delete(`/v1/app-groups/${groupId}`),
  addApp: (groupId: string, appId: string): Promise<void> =>
    client.post(`/v1/app-groups/${groupId}/apps/${appId}`),
  removeApp: (groupId: string, appId: string): Promise<void> =>
    client.delete(`/v1/app-groups/${groupId}/apps/${appId}`),
  listFavorites: (): Promise<AppFavorite[]> =>
    client.get('/v1/favorites/apps'),
  getFavoriteStatus: (appId: string): Promise<{ is_favorite: boolean }> =>
    client.get(`/v1/apps/${appId}/favorite`),
  addFavorite: (appId: string): Promise<AppFavorite> =>
    client.post(`/v1/apps/${appId}/favorite`),
  removeFavorite: (appId: string): Promise<void> =>
    client.delete(`/v1/apps/${appId}/favorite`),
}
