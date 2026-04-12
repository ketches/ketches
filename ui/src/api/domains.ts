import client from "./client"
import type { PaginationParams, PaginationResponse } from "./pagination"

export interface Domain {
  id: string
  name: string
  domain: string
  description: string
  scope: "cluster" | "env"
  cluster_id: string
  env_id?: string
  created_at: string
}

export interface CreateDomainRequest {
  name: string
  domain: string
  description?: string
}

export interface UpdateDomainRequest {
  name?: string
  domain?: string
  description?: string
}

export const domainsApi = {
  listByCluster: async (clusterId: string, params?: PaginationParams, projectId?: string) => {
    return client.get(`/v1/clusters/${clusterId}/domains`, {
      params: {
        ...params,
        ...(projectId ? { project_id: projectId } : {}),
      },
    }) as Promise<{ items: Domain[]; pagination: PaginationResponse }>
  },
  createForCluster: async (clusterId: string, data: CreateDomainRequest) => {
    return client.post(`/v1/clusters/${clusterId}/domains`, data) as Promise<Domain>
  },
  listByEnv: async (envId: string, params?: PaginationParams) => {
    return client.get(`/v1/envs/${envId}/domains`, { params }) as Promise<{ items: Domain[]; pagination: PaginationResponse }>
  },
  createForEnv: async (envId: string, data: CreateDomainRequest) => {
    return client.post(`/v1/envs/${envId}/domains`, data) as Promise<Domain>
  },
  update: async (scopeType: "cluster" | "env", scopeId: string, domainId: string, data: UpdateDomainRequest) => {
    const base = scopeType === "cluster" ? `/v1/clusters/${scopeId}` : `/v1/envs/${scopeId}`
    return client.put(`${base}/domains/${domainId}`, data) as Promise<Domain>
  },
  delete: async (scopeType: "cluster" | "env", scopeId: string, domainId: string) => {
    const base = scopeType === "cluster" ? `/v1/clusters/${scopeId}` : `/v1/envs/${scopeId}`
    return client.delete(`${base}/domains/${domainId}`)
  },
}
