import client from './client'
import { type PaginationParams, type PaginationResponse } from './pagination'

export interface Env {
  id: string
  slug: string
  name: string
  description: string
  project_id: string
  project_name: string
  cluster_id: string
  cluster_name: string
  cluster_connection_status?: string
  cluster_connection_status_reason?: string
  has_prometheus_integration?: boolean
  cluster_namespace: string
  is_build_env: boolean
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

export interface EnvNamespaceAvailabilityResponse {
  available: boolean
  source: string
  message: string
}

export interface ResourceQuota {
  cpu_request: string
  cpu_limit: string
  memory_request: string
  memory_limit: string
  pods: string
}

export interface UpdateResourceQuotaRequest {
  cpu_request: string
  cpu_limit: string
  memory_request: string
  memory_limit: string
  pods: string
}

export const envsApi = {
  list: async (projectId: string, params?: PaginationParams) => {
    return client.get(`/v1/projects/${projectId}/envs`, {
      params
    }) as Promise<{ items: Env[], pagination: PaginationResponse }>
  },
  listSimpleByProject: async (projectId: string) => {
    return client.get(`/v1/projects/${projectId}/envs/simple`) as Promise<Env[]>
  },
  create: async (projectId: string, data: CreateEnvRequest) => {
    return client.post(`/v1/projects/${projectId}/envs`, data) as Promise<Env>
  },
  checkNamespaceAvailability: async (projectId: string, params: { cluster_id: string; cluster_namespace: string }) => {
    return client.get(`/v1/projects/${projectId}/envs/namespace-availability`, {
      params,
    }) as Promise<EnvNamespaceAvailabilityResponse>
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
  getResourceQuota: async (envId: string) => {
    return client.get(`/v1/envs/${envId}/resource-quota`) as Promise<ResourceQuota>
  },
  updateResourceQuota: async (envId: string, data: UpdateResourceQuotaRequest) => {
    return client.put(`/v1/envs/${envId}/resource-quota`, data) as Promise<ResourceQuota>
  },
}
