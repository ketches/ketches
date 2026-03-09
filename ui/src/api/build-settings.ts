import client from './client'
import type { ContainerRegistry } from './container-registries'

export interface BuildSetting {
  id: string
  app_id: string
  git_repo_url: string
  git_ref: string
  git_username: string
  dockerfile_path: string
  build_context: string
  image_name: string
  registry_id: string
  registry?: ContainerRegistry
  build_args: string
  created_at: string
  updated_at: string
}

export interface UpsertBuildSettingRequest {
  git_repo_url: string
  git_ref?: string
  git_username?: string
  git_password?: string
  dockerfile_path?: string
  build_context?: string
  image_name: string
  registry_id: string
  build_args?: string
}

export interface TestGitResponse {
  success: boolean
  message: string
}

export const buildSettingsApi = {
  get: async (appId: string) => {
    return client.get(`/v1/apps/${appId}/build-setting`) as Promise<BuildSetting>
  },
  upsert: async (appId: string, data: UpsertBuildSettingRequest) => {
    return client.post(`/v1/apps/${appId}/build-setting`, data) as Promise<BuildSetting>
  },
  delete: async (appId: string) => {
    return client.delete(`/v1/apps/${appId}/build-setting`)
  },
  testGit: async (appId: string, data: { git_repo_url: string; git_ref?: string; git_username?: string; git_password?: string }) => {
    return client.post(`/v1/apps/${appId}/build-setting/test-git`, data) as Promise<TestGitResponse>
  },
  listAvailableRegistries: async (appId: string) => {
    return client.get(`/v1/apps/${appId}/build-setting/container-registries`) as Promise<ContainerRegistry[]>
  },
}
