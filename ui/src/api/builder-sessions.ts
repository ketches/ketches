import client from './client'
import { type PaginationParams, type PaginationResponse } from './pagination'

// --- Types ---

export type BuilderSessionStatus =
  | 'provisioning'
  | 'ready'
  | 'running'
  | 'succeeded'
  | 'failed'
  | 'expired'

export type BuilderRunStatus =
  | 'queued'
  | 'executing'
  | 'completed'
  | 'succeeded'
  | 'failed'
  | 'cancelled'

export interface BuilderWorkspaceFile {
  name: string
  type: 'file' | 'dir'
  size: number
  modTime: string
}

export interface ListBuilderWorkspaceFilesResponse {
  path: string
  files: BuilderWorkspaceFile[]
}

export interface ReadBuilderWorkspaceFileResponse {
  path: string
  content: string
  size: number
}

export interface BuilderSession {
  id: string
  project_id: string
  build_env_id: string
  title: string
  summary: string
  status: BuilderSessionStatus
  created_by: string
  created_at: string
  updated_at: string
  last_activity_at: string
  expires_at: string | null
  latest_run_id: string
  latest_run_status: BuilderRunStatus
  current_workspace_id: string
  current_workspace_status: string
  current_workspace_root: string
  artifact_count?: number
}

export interface BuilderMessage {
  id: string
  session_id: string
  run_id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  metadata_json: string
  created_by: string
  created_at: string
  updated_at: string
}

export interface BuilderRun {
  id: string
  session_id: string
  trigger_message_id: string
  workspace_id: string
  status: BuilderRunStatus
  requested_by: string
  instruction_summary: string
  execution_log: string
  started_at: string | null
  completed_at: string | null
  error_message: string
  created_at: string
  updated_at: string
}

export interface BuilderWorkspace {
  id: string
  session_id: string
  build_env_id: string
  cluster_id: string
  namespace: string
  pod_name: string
  container_name: string
  status: string
  workspace_root: string
  terminated_at: string | null
  created_at: string
  updated_at: string
}

export interface BuilderArtifact {
  id: string
  session_id: string
  workspace_id: string
  run_id: string
  kind: string
  path: string
  metadata_json: string
  created_at: string
  updated_at: string
}

export type BuilderPreviewStatus = 'unavailable' | 'delivery_only' | 'previewable'

export interface BuilderPreviewSummary {
  available: boolean
  status: BuilderPreviewStatus
  resolved_run_id: string
  published_at: string | null
  completed_at: string | null
  output_root: string
  default_entry_path: string
  download_available: boolean
  preview_available: boolean
  is_stale: boolean
  newer_run_id: string
  newer_run_status: BuilderRunStatus | ''
  download_url?: string
  preview_launch_url?: string
}

export interface BuilderPreviewLaunch {
  frame_url: string
}

export interface BuilderModelOption {
  key: string
  modelLabel: string
  providerLabel: string
  scope: 'project' | 'user'
  providerKey: string
  modelProfileKey: string
}

export interface BuilderModelSelection {
  options: BuilderModelOption[]
  effective_default_source: 'project' | 'user' | 'none'
  effective_default_option?: BuilderModelOption
}

export interface BuilderSessionDetail {
  session: BuilderSession
  messages: BuilderMessage[]
  runs: BuilderRun[]
  workspace?: BuilderWorkspace
  preview?: BuilderPreviewSummary
  artifacts: BuilderArtifact[]
}

// --- Request types ---

export interface CreateBuilderSessionRequest {
  build_env_id: string
  title?: string
  prompt: string
  selected_model_key?: string
  provider_key?: string
  model_profile_key?: string
}

export interface PostBuilderSessionMessageRequest {
  content: string
}

// --- Status helpers ---

export const builderSessionStatusLabels: Record<BuilderSessionStatus, string> = {
  provisioning: 'Provisioning',
  ready: 'Ready',
  running: 'Running',
  succeeded: 'Succeeded',
  failed: 'Failed',
  expired: 'Expired',
}

export const builderRunStatusLabels: Record<BuilderRunStatus, string> = {
  queued: 'Queued',
  executing: 'Executing',
  completed: 'Completed',
  succeeded: 'Succeeded',
  failed: 'Failed',
  cancelled: 'Cancelled',
}

export const builderRunStatusColors: Record<BuilderRunStatus, string> = {
  queued: 'bg-yellow-100 text-yellow-800',
  executing: 'bg-blue-100 text-blue-800',
  completed: 'bg-gray-100 text-gray-800',
  succeeded: 'bg-green-100 text-green-800',
  failed: 'bg-red-100 text-red-800',
  cancelled: 'bg-gray-100 text-gray-800',
}

// --- API ---

export const builderSessionsApi = {
  list: async (projectId: string, params?: PaginationParams) => {
    return client.get(`/v1/projects/${projectId}/builder-sessions`, {
      params,
    }) as Promise<{ items: BuilderSession[], pagination: PaginationResponse }>
  },

  get: async (projectId: string, sessionId: string) => {
    return client.get(`/v1/projects/${projectId}/builder-sessions/${sessionId}`) as Promise<BuilderSessionDetail>
  },

  getPreview: async (projectId: string, sessionId: string) => {
    return client.get(`/v1/projects/${projectId}/builder-sessions/${sessionId}/preview`) as Promise<BuilderPreviewSummary>
  },

  launchPreview: async (projectId: string, sessionId: string, runId: string) => {
    return client.get(`/v1/projects/${projectId}/builder-sessions/${sessionId}/runs/${runId}/preview/launch`) as Promise<BuilderPreviewLaunch>
  },

  getModelSelection: async (projectId: string) => {
    return client.get(`/v1/projects/${projectId}/builder-model-selection`) as Promise<BuilderModelSelection>
  },

  downloadPreviewSnapshotBlob: async (projectId: string, sessionId: string, runId: string): Promise<void> => {
    const authData = localStorage.getItem('auth-storage')
    let token = ''
    if (authData) {
      try {
        const { state } = JSON.parse(authData)
        token = state.accessToken || ''
      } catch { /* ignore */ }
    }
    const baseUrl = import.meta.env.VITE_API_BASE_URL || '/api'
    const response = await fetch(
      `${baseUrl}/v1/projects/${projectId}/builder-sessions/${sessionId}/runs/${runId}/delivery/download`,
      {
        headers: { Authorization: `Bearer ${token}` },
      }
    )
    if (!response.ok) {
      throw new Error(`Download failed: ${response.statusText}`)
    }
    const blob = await response.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `builder-output-${runId.slice(0, 8)}.tar.gz`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  },

  create: async (projectId: string, data: CreateBuilderSessionRequest) => {
    return client.post(`/v1/projects/${projectId}/builder-sessions`, data) as Promise<BuilderSessionDetail>
  },

  postMessage: async (projectId: string, sessionId: string, data: PostBuilderSessionMessageRequest) => {
    return client.post(`/v1/projects/${projectId}/builder-sessions/${sessionId}/messages`, data) as Promise<{
      session: BuilderSession
      message: BuilderMessage
      run: BuilderRun
    }>
  },

  listFiles: async (projectId: string, sessionId: string, path = '/') => {
    return client.get(`/v1/projects/${projectId}/builder-sessions/${sessionId}/files`, {
      params: { path },
    }) as Promise<ListBuilderWorkspaceFilesResponse>
  },

  readFile: async (projectId: string, sessionId: string, path: string) => {
    return client.get(`/v1/projects/${projectId}/builder-sessions/${sessionId}/files/read`, {
      params: { path },
    }) as Promise<ReadBuilderWorkspaceFileResponse>
  },

  downloadTar: (projectId: string, sessionId: string): string => {
    const baseUrl = import.meta.env.VITE_API_BASE_URL || '/api'
    const authData = localStorage.getItem('auth-storage')
    let token = ''
    if (authData) {
      try {
        const { state } = JSON.parse(authData)
        token = state.accessToken || ''
      } catch { /* ignore */ }
    }
    return `${baseUrl}/v1/projects/${projectId}/builder-sessions/${sessionId}/files/download?path=&_token=${encodeURIComponent(token)}`
  },

  downloadTarBlob: async (projectId: string, sessionId: string): Promise<void> => {
    const authData = localStorage.getItem('auth-storage')
    let token = ''
    if (authData) {
      try {
        const { state } = JSON.parse(authData)
        token = state.accessToken || ''
      } catch { /* ignore */ }
    }
    const baseUrl = import.meta.env.VITE_API_BASE_URL || '/api'
    const response = await fetch(
      `${baseUrl}/v1/projects/${projectId}/builder-sessions/${sessionId}/files/download?path=`,
      {
        headers: { Authorization: `Bearer ${token}` },
      }
    )
    if (!response.ok) {
      throw new Error(`Download failed: ${response.statusText}`)
    }
    const blob = await response.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `workspace-${sessionId.slice(0, 8)}.tar`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  },
}
