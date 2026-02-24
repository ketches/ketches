import client from './client'
import type { ContainerRegistry } from './container-registries'

export interface BuildConfig {
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
  auto_build: boolean
  auto_deploy: boolean
  webhook_secret: string
  webhook_enabled: boolean
  webhook_url: string
  created_at: string
  updated_at: string
}

export interface UpsertBuildConfigRequest {
  git_repo_url: string
  git_ref?: string
  git_username?: string
  git_password?: string
  dockerfile_path?: string
  build_context?: string
  image_name: string
  registry_id: string
  build_args?: string
  auto_build?: boolean
  auto_deploy?: boolean
  webhook_enabled?: boolean
}

export interface TestGitResponse {
  success: boolean
  message: string
}

export const buildConfigsApi = {
  get: async (appId: string) => {
    return client.get(`/v1/apps/${appId}/build-config`) as Promise<BuildConfig>
  },
  upsert: async (appId: string, data: UpsertBuildConfigRequest) => {
    return client.post(`/v1/apps/${appId}/build-config`, data) as Promise<BuildConfig>
  },
  delete: async (appId: string) => {
    return client.delete(`/v1/apps/${appId}/build-config`)
  },
  testGit: async (appId: string, data: { git_repo_url: string; git_ref?: string; git_username?: string; git_password?: string }) => {
    return client.post(`/v1/apps/${appId}/build-config/test-git`, data) as Promise<TestGitResponse>
  },
  listAvailableRegistries: async (appId: string) => {
    return client.get(`/v1/apps/${appId}/build-config/container-registries`) as Promise<ContainerRegistry[]>
  },
}
