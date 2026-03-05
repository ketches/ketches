import client from './client'
import { type PaginationParams, type PaginationResponse, type SimpleResponse } from './pagination'

export type RegistryProvider = 'dockerhub' | 'harbor' | 'ghcr' | 'acr' | 'ecr' | 'aliyun' | 'custom'
export type RegistryScope = 'cluster' | 'project'

export interface ContainerRegistry {
  id: string
  name: string
  provider: RegistryProvider
  endpoint: string
  skip_tls_verify: boolean
  namespace: string
  username: string
  password?: string
  scope: RegistryScope
  cluster_id?: string
  project_id?: string
  is_default: boolean
  enabled: boolean
  description: string
  created_at: string
}

export interface CreateContainerRegistryRequest {
  name: string
  provider: RegistryProvider
  endpoint: string
  skip_tls_verify?: boolean
  namespace?: string
  username?: string
  password?: string
  is_default?: boolean
  enabled?: boolean
  description?: string
}

export interface UpdateContainerRegistryRequest {
  name?: string
  provider?: RegistryProvider
  endpoint?: string
  skip_tls_verify?: boolean
  namespace?: string
  username?: string
  password?: string
  is_default?: boolean
  enabled?: boolean
  description?: string
}

export interface TestRegistryResponse {
  success: boolean
  message: string
}

export const containerRegistriesApi = {
  // Cluster scope
  listByCluster: async (clusterId: string, params?: PaginationParams) => {
    return client.get(`/v1/clusters/${clusterId}/container-registries`, { params }) as Promise<{ items: ContainerRegistry[], pagination: PaginationResponse }>
  },
  createForCluster: async (clusterId: string, data: CreateContainerRegistryRequest) => {
    return client.post(`/v1/clusters/${clusterId}/container-registries`, data) as Promise<ContainerRegistry>
  },

  // Project scope
  listByProject: async (projectId: string, params?: PaginationParams) => {
    return client.get(`/v1/projects/${projectId}/container-registries`, { params }) as Promise<{ items: ContainerRegistry[], pagination: PaginationResponse }>
  },
  listSimpleByProject: async (projectId: string) => {
    return client.get(`/v1/projects/${projectId}/container-registries/simple`) as Promise<SimpleResponse[]>
  },
  createForProject: async (projectId: string, data: CreateContainerRegistryRequest) => {
    return client.post(`/v1/projects/${projectId}/container-registries`, data) as Promise<ContainerRegistry>
  },

  // Common
  get: async (id: string) => {
    return client.get(`/v1/container-registries/${id}`) as Promise<ContainerRegistry>
  },
  update: async (id: string, data: UpdateContainerRegistryRequest) => {
    return client.put(`/v1/container-registries/${id}`, data) as Promise<ContainerRegistry>
  },
  delete: async (id: string) => {
    return client.delete(`/v1/container-registries/${id}`)
  },
  test: async (id: string, data: { provider: string; endpoint: string; skip_tls_verify?: boolean; username?: string; password?: string }) => {
    return client.post(`/v1/container-registries/${id}/test`, data) as Promise<TestRegistryResponse>
  },
}

export const registryProviderLabels: Record<RegistryProvider, string> = {
  dockerhub: 'Docker Hub',
  harbor: 'Harbor',
  ghcr: 'GitHub Container Registry',
  acr: 'Azure Container Registry',
  ecr: 'AWS ECR',
  aliyun: 'Aliyun Container Registry',
  custom: 'Custom Registry',
}
