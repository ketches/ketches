import client from './client'

export interface DashboardStats {
  cluster_count?: number
  project_count?: number
  environment_count?: number
  application_count?: number
  code_repository_count?: number
  plugin_count?: number
  user_count?: number
  project_member_count?: number
}

export interface EnvironmentResourceUsage {
  environment_id: string
  environment_name: string
  namespace: string
  cpu_usage: number
  cpu_limit: number
  memory_usage: number
  memory_limit: number
  pod_count: number
}

export const dashboardApi = {
  getStats: async (projectId?: string) => {
    const params = projectId ? `?project_id=${projectId}` : ''
    return client.get(`/v1/dashboard/stats${params}`) as Promise<DashboardStats>
  },

  getEnvironments: async (projectId: string) => {
    return client.get(`/v1/dashboard/environments?project_id=${projectId}`) as Promise<EnvironmentResourceUsage[]>
  },
}
