import client from './client'
import { type PaginationParams, type PaginationResponse } from './pagination'

export interface Certificate {
  id: string
  name: string
  description: string
  scope: 'cluster' | 'env'
  cluster_id: string
  env_id: string
  created_at: string
}

export interface CreateCertificateRequest {
  name: string
  description?: string
  cert: string   // PEM certificate text
  key: string    // PEM private key text
  scope: 'cluster' | 'env'
}

export interface UpdateCertificateRequest {
  name?: string
  description?: string
  cert?: string
  key?: string
}

export const certificatesApi = {
  listByCluster: async (clusterId: string, params?: PaginationParams, projectId?: string) => {
    return client.get(`/v1/clusters/${clusterId}/certificates`, {
      params: {
        ...params,
        ...(projectId ? { project_id: projectId } : {}),
      },
    }) as Promise<{ items: Certificate[], pagination: PaginationResponse }>
  },
  createForCluster: async (clusterId: string, data: CreateCertificateRequest) => {
    return client.post(`/v1/clusters/${clusterId}/certificates`, data) as Promise<Certificate>
  },
  listByEnv: async (envId: string, params?: PaginationParams) => {
    return client.get(`/v1/envs/${envId}/certificates`, { params }) as Promise<{ items: Certificate[], pagination: PaginationResponse }>
  },
  createForEnv: async (envId: string, data: CreateCertificateRequest) => {
    return client.post(`/v1/envs/${envId}/certificates`, data) as Promise<Certificate>
  },
  get: async (scopeType: 'cluster' | 'env', scopeId: string, certId: string) => {
    const base = scopeType === 'cluster' ? `/v1/clusters/${scopeId}` : `/v1/envs/${scopeId}`
    return client.get(`${base}/certificates/${certId}`) as Promise<Certificate>
  },
  update: async (scopeType: 'cluster' | 'env', scopeId: string, certId: string, data: UpdateCertificateRequest) => {
    const base = scopeType === 'cluster' ? `/v1/clusters/${scopeId}` : `/v1/envs/${scopeId}`
    return client.put(`${base}/certificates/${certId}`, data) as Promise<Certificate>
  },
  delete: async (scopeType: 'cluster' | 'env', scopeId: string, certId: string) => {
    const base = scopeType === 'cluster' ? `/v1/clusters/${scopeId}` : `/v1/envs/${scopeId}`
    return client.delete(`${base}/certificates/${certId}`)
  },
}
