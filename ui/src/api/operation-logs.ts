import client from './client'
import { type PaginationParams, type PaginationResponse } from './pagination'
import { authenticatedFetch } from '@/lib/auth-session'

export type OperationLogSensitivity = 'public' | 'internal' | 'sensitive'
export type OperationLogStatus = 'success' | 'failure'

export interface OperationLogItem {
  id: string
  created_at: string
  user_id?: string
  username: string
  action: string
  resource_type: string
  resource_id: string
  project_id?: string
  env_id?: string
  app_id?: string
  repo_id?: string
  status: OperationLogStatus
  status_code: number
  sensitivity: OperationLogSensitivity
  request_summary: string
  client_ip: string
}

export interface OperationLogListParams extends PaginationParams {
  user_id?: string
  action?: string
  resource_type?: string
  sensitivity?: OperationLogSensitivity
  status?: OperationLogStatus
  start?: string
  end?: string
  export?: boolean
}

export interface OperationLogListResponse {
  items: OperationLogItem[]
  pagination: PaginationResponse
}

export interface OperationLogSettings {
  retention_days: number
}

export const listActivities = async (params?: OperationLogListParams) => {
  return client.get('/v1/activities', { params }) as Promise<OperationLogListResponse>
}

export const listOperationLogs = async (params?: OperationLogListParams) => {
  return client.get('/v1/operation-logs', { params }) as Promise<OperationLogListResponse>
}

export const listPlatformAuditLogs = async (params?: OperationLogListParams) => {
  return client.get('/v1/platform-settings/audit-logs', { params }) as Promise<OperationLogListResponse>
}

export const listAppOperationLogs = async (appId: string, params?: OperationLogListParams) => {
  return client.get(`/v1/apps/${appId}/operation-logs`, { params }) as Promise<OperationLogListResponse>
}

export const listCodeRepositoryOperationLogs = async (repoId: string, params?: OperationLogListParams) => {
  return client.get(`/v1/code-repositories/${repoId}/operation-logs`, { params }) as Promise<OperationLogListResponse>
}

export const getOperationLogSettings = async () => {
  return client.get('/v1/operation-logs/settings') as Promise<OperationLogSettings>
}

export const updateOperationLogSettings = async (retentionDays: number) => {
  return client.put('/v1/operation-logs/settings', { retention_days: retentionDays })
}

export const exportOperationLogsCSV = async (params?: OperationLogListParams) => {
  const baseUrl = import.meta.env.VITE_API_BASE_URL || '/api'
  const query = new URLSearchParams()
  if (params?.page) query.set('page', String(params.page))
  if (params?.page_size) query.set('page_size', String(params.page_size))
  if (params?.search) query.set('search', params.search)
  if (params?.user_id) query.set('user_id', params.user_id)
  if (params?.action) query.set('action', params.action)
  if (params?.resource_type) query.set('resource_type', params.resource_type)
  if (params?.sensitivity) query.set('sensitivity', params.sensitivity)
  if (params?.status) query.set('status', params.status)
  if (params?.start) query.set('start', params.start)
  if (params?.end) query.set('end', params.end)
  query.set('export', 'true')

  const response = await authenticatedFetch(`${baseUrl}/v1/operation-logs?${query.toString()}`)

  if (!response.ok) {
    throw new Error(`Export failed: ${response.statusText}`)
  }

  const blob = await response.blob()
  const disposition = response.headers.get('Content-Disposition')
  const fileNameMatch = disposition?.match(/filename="?([^"]+)"?/)
  const filename = fileNameMatch?.[1] || 'operation-logs.csv'

  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

export const operationLogsApi = {
  listActivities,
  listOperationLogs,
  listPlatformAuditLogs,
  listAppOperationLogs,
  listCodeRepositoryOperationLogs,
  getOperationLogSettings,
  updateOperationLogSettings,
  exportOperationLogsCSV,
}

export const activitiesApi = {
  list: listActivities,
}
