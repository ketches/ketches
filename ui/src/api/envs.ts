import client from './client'

export interface Env {
  id: string
  slug: string
  name: string
  description: string
  project_id: string
  cluster_id: string
  cluster_namespace: string
  is_build_env: boolean
  status: string
  created_at: string
}

export interface CreateEnvRequest {
  name: string
  slug: string
  description: string
  project_id: string
  cluster_id: string
  cluster_namespace: string
}

export const envsApi = {
  list: async (projectId: string, search?: string) => {
    return client.get(`/v1/projects/${projectId}/envs`, {
      params: { search }
    }) as Promise<Env[]>
  },
  create: async (projectId: string, data: CreateEnvRequest) => {
    return client.post(`/v1/projects/${projectId}/envs`, data) as Promise<Env>
  },
  get: async (id: string) => {
    return client.get(`/v1/envs/${id}`) as Promise<Env>
  },

  update: async (id: string, data: Partial<Env>) => {
    return client.patch(`/v1/envs/${id}/basic`, data) as Promise<Env>
  },
  delete: async (id: string) => {
    return client.delete(`/v1/envs/${id}`)
  },

  setBuildEnv: async (id: string) => {
    return client.patch(`/v1/envs/${id}/set-build-env`) as Promise<Env>
  },
  unsetBuildEnv: async (id: string) => {
    return client.patch(`/v1/envs/${id}/unset-build-env`) as Promise<Env>
  },
}
