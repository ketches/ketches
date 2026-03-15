import client from "./client"

export interface PlatformUpdateTargetConfig {
  image_repository: string
  namespace: string
  deployment_name: string
  container_name: string
}

export interface PlatformUpdateConfig {
  api: PlatformUpdateTargetConfig
  ui: PlatformUpdateTargetConfig
}

export interface PlatformUpdateComponentStatus {
  current_version: string
  available_versions: string[]
  latest_version: string
  update_available: boolean
  version_source: string
  rollout_phase: string
  rollout_message: string
  error?: string
}

export interface PlatformUpdateStatus {
  local_platform_version: string
  running_in_kubernetes: boolean
  can_rollout: boolean
  rollout_blockers: string[]
  config: PlatformUpdateConfig
  api: PlatformUpdateComponentStatus
  ui: PlatformUpdateComponentStatus
  recommended_shared_version: string
}

export interface CheckPlatformUpdateRequest {
  mode: "auto" | "manual"
}

export interface TriggerPlatformRolloutRequest {
  shared_version?: string
  api_version?: string
  ui_version?: string
}

export interface PlatformUpdateRolloutTarget {
  namespace: string
  deployment_name: string
  container_name: string
  previous_image: string
  target_image: string
}

export interface PlatformUpdateRolloutResult {
  api: PlatformUpdateRolloutTarget
  ui: PlatformUpdateRolloutTarget
  rollback_attempted: boolean
  rollback_succeeded: boolean
}

export const platformUpdateApi = {
  getConfig: async () => {
    return client.get("/v1/platform-update/config") as Promise<PlatformUpdateConfig>
  },

  updateConfig: async (data: PlatformUpdateConfig) => {
    return client.put("/v1/platform-update/config", data) as Promise<PlatformUpdateConfig>
  },

  getStatus: async () => {
    return client.get("/v1/platform-update/status") as Promise<PlatformUpdateStatus>
  },

  check: async (data: CheckPlatformUpdateRequest) => {
    return client.post("/v1/platform-update/check", data) as Promise<PlatformUpdateStatus>
  },

  triggerRollout: async (data: TriggerPlatformRolloutRequest) => {
    return client.post("/v1/platform-update/rollout", data) as Promise<PlatformUpdateRolloutResult>
  },
}
