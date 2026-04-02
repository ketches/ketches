import client from './client'
import { type PaginationParams, type PaginationResponse } from './pagination'
import { getStoredAccessToken, syncAuthCookie } from '@/lib/auth-session'

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
  phase: string
  requested_by: string
  planned_project_kind: string
  planned_project_summary: string
  planned_executor_policy_key: string
  planned_image_profile_key: string
  executor_policy_key: string
  execution_image_profile_key: string
  execution_image_ref: string
  error_code: string
  error_class: string
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

export interface BuilderExport {
  id: string
  session_id: string
  run_id: string
  workspace_id: string
  snapshot_id: string
  kind: string
  status: string
  file_name: string
  storage_path: string
  source_root: string
  file_count: number
  size_bytes: number
  metadata_json: string
  error_message: string
  created_by: string
  created_at: string
  updated_at: string
}

export interface BuilderExportPromotionRequest {
  name?: string
  slug?: string
  git_repo_url: string
  git_username?: string
  git_password?: string
}

export interface BuilderExportPromotionResponse {
  export: BuilderExport
  repository: {
    id: string
    project_id: string
    name: string
    slug: string
    git_repo_url: string
    git_username: string
    git_password: string
    created_at: string
    updated_at: string
  }
}

export interface BuilderExportPromotionPlan {
  export: BuilderExport
  source_kind: string
  planned_project_kind: string
  suggested_repository_name: string
  suggested_repository_slug: string
  suggested_build_env_id: string
  suggested_build_setting_name: string
  suggested_image_name: string
  suggested_dockerfile_path: string
  suggested_build_context: string
  can_trigger_initial_build: boolean
  requires_registry_selection: boolean
  missing_requirements: string[]
}

export interface BuilderExportInitialBuildPromotionRequest {
  name?: string
  slug?: string
  git_repo_url: string
  git_username?: string
  git_password?: string
  build_env_id: string
  registry_id: string
  build_setting_name?: string
  image_name?: string
  dockerfile_path?: string
  build_context?: string
  git_ref?: string
}

export interface BuilderExportInitialBuildPromotionResponse {
  promotion: BuilderExportPromotionResponse
  build_setting: {
    id: string
    code_repository_id?: string
    name: string
    git_ref: string
    dockerfile_path: string
    build_context: string
    image_name: string
    registry_id: string
    build_args: string
    platforms: string
    registry_cache_enabled: boolean
    registry_cache_ref: string
    created_at: string
    updated_at: string
  }
  build: {
    id: string
    build_setting_id: string
    build_number: number
    status: string
    build_env_id: string
    git_repo_url: string
    git_ref: string
    git_commit_sha: string
    git_commit_msg: string
    image_full_name: string
    trigger_type: string
    triggered_by: string
    job_name: string
    job_namespace: string
    started_at: string | null
    completed_at: string | null
    duration: number
    error_message: string
    log_persist_status: string
    log_persist_error: string
    created_at: string
  }
}

export interface BuilderExportDeployBuildRequest {
  repository_id: string
  build_id: string
  target_env_id: string
  app_id?: string
  name?: string
  slug?: string
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
  effectiveDefaultSource: 'project' | 'user' | 'none'
  effectiveDefaultOption?: BuilderModelOption
}

interface BuilderModelOptionWire {
  key: string
  model_label: string
  provider_label: string
  scope: 'project' | 'user'
  provider_key: string
  model_profile_key: string
}

interface BuilderModelSelectionWire {
  options: BuilderModelOptionWire[]
  effective_default_source: 'project' | 'user' | 'none'
  effective_default_option?: BuilderModelOptionWire
}

function mapBuilderModelOption(option: BuilderModelOptionWire): BuilderModelOption {
  return {
    key: option.key,
    modelLabel: option.model_label,
    providerLabel: option.provider_label,
    scope: option.scope,
    providerKey: option.provider_key,
    modelProfileKey: option.model_profile_key,
  }
}

function mapBuilderModelSelection(selection: BuilderModelSelectionWire): BuilderModelSelection {
  return {
    options: selection.options.map(mapBuilderModelOption),
    effectiveDefaultSource: selection.effective_default_source,
    effectiveDefaultOption: selection.effective_default_option
      ? mapBuilderModelOption(selection.effective_default_option)
      : undefined,
  }
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
  executor_policy_key?: string
  execution_image_profile_key?: string
}

export interface PostBuilderSessionMessageRequest {
  content: string
  selected_model_key?: string
  provider_key?: string
  model_profile_key?: string
  executor_policy_key?: string
  execution_image_profile_key?: string
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

  listExports: async (projectId: string, sessionId: string) => {
    return client.get(`/v1/projects/${projectId}/builder-sessions/${sessionId}/exports`) as Promise<BuilderExport[]>
  },

  createExport: async (projectId: string, sessionId: string) => {
    return client.post(`/v1/projects/${projectId}/builder-sessions/${sessionId}/exports`, {}) as Promise<BuilderExport>
  },

  getExportPromotionPlan: async (projectId: string, sessionId: string, exportId: string) => {
    return client.get(`/v1/projects/${projectId}/builder-sessions/${sessionId}/exports/${exportId}/promotion-plan`) as Promise<BuilderExportPromotionPlan>
  },

  promoteExportToRepository: async (projectId: string, sessionId: string, exportId: string, data: BuilderExportPromotionRequest) => {
    return client.post(`/v1/projects/${projectId}/builder-sessions/${sessionId}/exports/${exportId}/promote-repository`, data) as Promise<BuilderExportPromotionResponse>
  },

  promoteExportToInitialBuild: async (projectId: string, sessionId: string, exportId: string, data: BuilderExportInitialBuildPromotionRequest) => {
    return client.post(`/v1/projects/${projectId}/builder-sessions/${sessionId}/exports/${exportId}/promote-build`, data) as Promise<BuilderExportInitialBuildPromotionResponse>
  },

  deployExportBuild: async (projectId: string, sessionId: string, exportId: string, data: BuilderExportDeployBuildRequest) => {
    return client.post(`/v1/projects/${projectId}/builder-sessions/${sessionId}/exports/${exportId}/deploy-build`, data)
  },

  downloadExportBlob: async (projectId: string, sessionId: string, exportId: string): Promise<void> => {
    const token = getStoredAccessToken()
    const baseUrl = import.meta.env.VITE_API_BASE_URL || '/api'
    const response = await fetch(
      `${baseUrl}/v1/projects/${projectId}/builder-sessions/${sessionId}/exports/${exportId}/download`,
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
    a.download = `builder-export-${exportId.slice(0, 8)}.tar.gz`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  },

  getModelSelection: async (projectId: string) => {
    const selection = await client.get(`/v1/projects/${projectId}/builder-model-selection`) as BuilderModelSelectionWire
    return mapBuilderModelSelection(selection)
  },

  runLogsStreamUrl: (projectId: string, sessionId: string, runId: string) => {
    const baseUrl = import.meta.env.VITE_API_BASE_URL || '/api'
    syncAuthCookie()
    return `${baseUrl}/v1/projects/${projectId}/builder-sessions/${sessionId}/runs/${runId}/logs`
  },

  downloadPreviewSnapshotBlob: async (projectId: string, sessionId: string, runId: string): Promise<void> => {
    const token = getStoredAccessToken()
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
    syncAuthCookie()
    return `${baseUrl}/v1/projects/${projectId}/builder-sessions/${sessionId}/files/download?path=`
  },

  downloadTarBlob: async (projectId: string, sessionId: string): Promise<void> => {
    const token = getStoredAccessToken()
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
