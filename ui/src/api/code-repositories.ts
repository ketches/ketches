import type { App } from './apps'
import type { Build } from './builds'
import client from './client'
import type { ContainerRegistry } from './container-registries'
import { type PaginationParams, type PaginationResponse, type SimpleResponse } from './pagination'

export interface CodeRepository {
  id: string
  project_id: string
  name: string
  slug: string
  description?: string
  git_repo_url: string
  git_username: string
  git_password: string
  has_git_password: boolean
  webhook_secret: string
  webhook_enabled: boolean
  webhook_url: string
  created_at: string
  updated_at: string
}

export interface GitRef {
  name: string
  type: 'branch' | 'tag'
}

export interface CreateCodeRepositoryRequest {
  name?: string
  slug?: string
  git_repo_url: string
  git_username?: string
  git_password?: string
}

export interface UpdateCodeRepositoryRequest {
  name?: string
  git_repo_url?: string
  git_username?: string
  git_password?: string
  webhook_enabled?: boolean
}

export interface CodeRepositoryBuildConfig {
  id: string
  code_repository_id: string
  name: string
  git_ref: string
  dockerfile_path: string
  build_context: string
  image_name: string
  registry_id: string
  registry?: ContainerRegistry
  build_args: string
  auto_build: boolean
  auto_deploy: boolean
  webhook_enabled: boolean
  created_at: string
  updated_at: string
}

export interface CreateCodeRepositoryBuildConfigRequest {
  name: string
  git_ref?: string
  dockerfile_path?: string
  build_context?: string
  image_name: string
  registry_id: string
  build_args?: string
  auto_build?: boolean
  auto_deploy?: boolean
  webhook_enabled?: boolean
}

export interface UpdateCodeRepositoryBuildConfigRequest {
  name?: string
  git_ref?: string
  dockerfile_path?: string
  build_context?: string
  image_name?: string
  registry_id?: string
  build_args?: string
  auto_build?: boolean
  auto_deploy?: boolean
  webhook_enabled?: boolean
}

export interface TriggerCodeRepositoryBuildRequest {
  build_config_id: string
  build_env_id: string
  git_ref?: string
  auto_deploy?: boolean
  deploy_env_id?: string
  deploy_app_id?: string
  deploy_app_name?: string
  deploy_app_slug?: string
}

export interface DeployCodeRepositoryBuildRequest {
  target_env_id: string
  app_id?: string
  name?: string
  slug?: string
}

export const codeRepositoriesApi = {
  list: async (projectId: string, params?: PaginationParams) => {
    return client.get(`/v1/projects/${projectId}/code-repositories`, { params }) as Promise<{ items: CodeRepository[], pagination: PaginationResponse }>
  },
  listSimple: async (projectId: string) => {
    return client.get(`/v1/projects/${projectId}/code-repositories/simple`) as Promise<SimpleResponse[]>
  },
  create: async (projectId: string, data: CreateCodeRepositoryRequest) => {
    return client.post(`/v1/projects/${projectId}/code-repositories`, data) as Promise<CodeRepository>
  },
  get: async (repoId: string) => {
    return client.get(`/v1/code-repositories/${repoId}`) as Promise<CodeRepository>
  },
  update: async (repoId: string, data: UpdateCodeRepositoryRequest) => {
    return client.put(`/v1/code-repositories/${repoId}`, data) as Promise<CodeRepository>
  },
  delete: async (repoId: string) => {
    return client.delete(`/v1/code-repositories/${repoId}`)
  },
  listContainerRegistries: async (repoId: string) => {
    return client.get(`/v1/code-repositories/${repoId}/container-registries`) as Promise<ContainerRegistry[]>
  },
  listRefs: async (repoId: string) => {
    return client.get(`/v1/code-repositories/${repoId}/refs`) as Promise<{ refs: GitRef[] }>
  },
  listBuildConfigs: async (repoId: string) => {
    return client.get(`/v1/code-repositories/${repoId}/build-configs`) as Promise<CodeRepositoryBuildConfig[]>
  },
  createBuildConfig: async (repoId: string, data: CreateCodeRepositoryBuildConfigRequest) => {
    return client.post(`/v1/code-repositories/${repoId}/build-configs`, data) as Promise<CodeRepositoryBuildConfig>
  },
  getBuildConfig: async (repoId: string, configId: string) => {
    return client.get(`/v1/code-repositories/${repoId}/build-configs/${configId}`) as Promise<CodeRepositoryBuildConfig>
  },
  updateBuildConfig: async (repoId: string, configId: string, data: UpdateCodeRepositoryBuildConfigRequest) => {
    return client.put(`/v1/code-repositories/${repoId}/build-configs/${configId}`, data) as Promise<CodeRepositoryBuildConfig>
  },
  deleteBuildConfig: async (repoId: string, configId: string) => {
    return client.delete(`/v1/code-repositories/${repoId}/build-configs/${configId}`)
  },
  testGit: async (repoId: string, data: { git_repo_url: string; git_ref?: string; git_username?: string; git_password?: string }) => {
    return client.post(`/v1/code-repositories/${repoId}/test-git`, data)
  },
  listBuilds: async (repoId: string) => {
    return client.get(`/v1/code-repositories/${repoId}/builds`) as Promise<Build[]>
  },
  listDeployments: async (repoId: string) => {
    return client.get(`/v1/code-repositories/${repoId}/deployments`) as Promise<Build[]>
  },
  triggerBuild: async (repoId: string, data: TriggerCodeRepositoryBuildRequest) => {
    return client.post(`/v1/code-repositories/${repoId}/builds`, data) as Promise<Build>
  },
  getBuild: async (repoId: string, buildId: string) => {
    return client.get(`/v1/code-repositories/${repoId}/builds/${buildId}`) as Promise<Build>
  },
  cancelBuild: async (repoId: string, buildId: string) => {
    return client.post(`/v1/code-repositories/${repoId}/builds/${buildId}/cancel`) as Promise<Build>
  },
  deployBuild: async (repoId: string, buildId: string, data: DeployCodeRepositoryBuildRequest) => {
    return client.post(`/v1/code-repositories/${repoId}/builds/${buildId}/deploy`, data) as Promise<{ build: Build; app: App }>
  },
  getTopology: async (id: string) => {
    return client.get(`/v1/code-repositories/${id}/topology`) as Promise<{
      nodes: {
        id: string
        type: string
        name: string
        status?: string
        metadata?: Record<string, string>
      }[]
      edges: {
        source: string
        target: string
        type?: string
      }[]
    }>
  },
}
