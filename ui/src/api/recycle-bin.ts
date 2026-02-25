import client from './client'
import { type PaginationParams, type PaginationResponse } from './pagination'

export interface RecycleBinApp {
  id: string
  slug: string
  name: string
  description: string
  env_id: string
  env_name: string
  project_id: string
  project_name: string
  app_type: string
  container_image: string
  deleted_at: string
}

export interface RecycleBinEnv {
  id: string
  slug: string
  name: string
  description: string
  project_id: string
  project_name: string
  cluster_id: string
  cluster_name: string
  cluster_namespace: string
  deleted_at: string
}

export interface EnvDeletionConflict {
  apps: RecycleBinApp[]
}

export const recycleBinApi = {
  listApps: async (projectId?: string, params?: PaginationParams) => {
    return client.get('/v1/recycle-bin/apps', {
      params: { ...params, project_id: projectId }
    }) as Promise<{ items: RecycleBinApp[], pagination: PaginationResponse }>
  },

  listEnvs: async (projectId?: string, params?: PaginationParams) => {
    return client.get('/v1/recycle-bin/envs', {
      params: { ...params, project_id: projectId }
    }) as Promise<{ items: RecycleBinEnv[], pagination: PaginationResponse }>
  },

  restoreApps: async (ids: string[]) => {
    return client.post('/v1/recycle-bin/apps/restore', { ids })
  },

  permanentlyDeleteApps: async (ids: string[]) => {
    return client.post('/v1/recycle-bin/apps/permanently-delete', { ids })
  },

  restoreEnvs: async (ids: string[]) => {
    return client.post('/v1/recycle-bin/envs/restore', { ids })
  },

  permanentlyDeleteEnvs: async (ids: string[]) => {
    return client.post('/v1/recycle-bin/envs/permanently-delete', { ids })
  },

  checkEnvDeletionConflicts: async (envId: string) => {
    return client.get(`/v1/recycle-bin/envs/${envId}/deletion-conflicts`) as Promise<EnvDeletionConflict>
  }
}
