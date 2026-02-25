import client from './client'
import { type PaginationResponse } from './pagination'

export type BuildStatus = 'pending' | 'cloning' | 'building' | 'succeeded' | 'failed' | 'cancelled'
export type BuildTriggerType = 'manual' | 'webhook' | 'auto'

export interface Build {
  id: string
  code_repository_id?: string
  code_repository_build_config_id?: string
  app_id: string
  app?: {
    id: string
    name: string
    env?: {
      id: string
      name: string
    }
  }
  build_config_id: string
  build_number: number
  status: BuildStatus
  build_env_id: string
  git_repo_url: string
  git_ref: string
  git_commit_sha: string
  git_commit_msg: string
  image_full_name: string
  trigger_type: BuildTriggerType
  triggered_by: string
  job_name: string
  job_namespace: string
  started_at: string | null
  completed_at: string | null
  duration: number
  error_message: string
  created_at: string
}

export interface TriggerBuildRequest {
  git_ref?: string
  image_tag?: string
  auto_deploy?: boolean
}

export const buildsApi = {
  list: async (appId: string, page = 1, pageSize = 10) => {
    return client.get(`/v1/apps/${appId}/builds`, {
      params: { page, page_size: pageSize }
    }) as Promise<{ items: Build[], pagination: PaginationResponse }>
  },
  trigger: async (appId: string, data?: TriggerBuildRequest) => {
    return client.post(`/v1/apps/${appId}/builds`, data || {}) as Promise<Build>
  },
  get: async (appId: string, buildId: string) => {
    return client.get(`/v1/apps/${appId}/builds/${buildId}`) as Promise<Build>
  },
  cancel: async (appId: string, buildId: string) => {
    return client.post(`/v1/apps/${appId}/builds/${buildId}/cancel`) as Promise<Build>
  },
  deploy: async (appId: string, buildId: string) => {
    return client.post(`/v1/apps/${appId}/builds/${buildId}/deploy`) as Promise<Build>
  },
  rebuild: async (appId: string, buildId: string, data?: { image_tag?: string }) => {
    return client.post(`/v1/apps/${appId}/builds/${buildId}/rebuild`, data || {}) as Promise<Build>
  },
}

export const buildStatusLabels: Record<BuildStatus, string> = {
  pending: 'Pending',
  cloning: 'Cloning',
  building: 'Building',
  succeeded: 'Succeeded',
  failed: 'Failed',
  cancelled: 'Cancelled',
}

export const buildStatusColors: Record<BuildStatus, string> = {
  pending: 'bg-yellow-100 text-yellow-800',
  cloning: 'bg-blue-100 text-blue-800',
  building: 'bg-blue-100 text-blue-800',
  succeeded: 'bg-green-100 text-green-800',
  failed: 'bg-red-100 text-red-800',
  cancelled: 'bg-gray-100 text-gray-800',
}
