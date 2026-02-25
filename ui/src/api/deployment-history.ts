import client, { type PaginationResponse } from "./client"

export interface DeploymentHistory {
  id: string
  app_id: string
  created_at: string
  
  image_before: string
  image_after: string
  replicas_before: number
  replicas_after: number
  
  request_cpu_before: number
  request_cpu_after: number
  request_memory_before: number
  request_memory_after: number
  limit_cpu_before: number
  limit_cpu_after: number
  limit_memory_before: number
  limit_memory_after: number
  
  deploy_type: string
  deployed_by: string
  reason: string
  status: string
  
  build_id?: string
}

export const deploymentHistoryApi = {
  list: async (appId: string, page = 1, pageSize = 10) => {
    return client.get(`/v1/apps/${appId}/deployment-history`, {
      params: { page, page_size: pageSize }
    }) as Promise<{ items: DeploymentHistory[], pagination: PaginationResponse }>
  },
  
  rollback: async (appId: string, historyId: string) => {
    return client.post(`/v1/apps/${appId}/deployment-history/rollback`, { history_id: historyId })
  }
}
