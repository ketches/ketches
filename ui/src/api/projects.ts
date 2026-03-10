import client from './client'
import { type PaginationParams, type PaginationResponse } from './pagination'

export interface Project {
  id: string
  slug: string
  name: string
  description?: string
  owner_name?: string
  created_at: string
}

export const ProjectRole = {
  OWNER: 'owner',
  DEVELOPER: 'developer',
  VIEWER: 'viewer',
} as const

export type ProjectRole = typeof ProjectRole[keyof typeof ProjectRole]

export const ProjectRoleLabels: Record<ProjectRole, string> = {
  [ProjectRole.OWNER]: 'Owner',
  [ProjectRole.DEVELOPER]: 'Developer',
  [ProjectRole.VIEWER]: 'Viewer',
}

export const ProjectRoleDescriptions: Record<ProjectRole, string> = {
  [ProjectRole.OWNER]: 'Full project access, can manage members and settings',
  [ProjectRole.DEVELOPER]: 'Can deploy and manage applications',
  [ProjectRole.VIEWER]: 'Read-only access to project resources',
}

export const PROJECT_ROLES = Object.values(ProjectRole) as ProjectRole[]

export interface ProjectMember {
  user_id: string
  username: string
  fullname?: string
  email: string
  project_role: ProjectRole
  joined_at: string
}

export const projectsApi = {
  list: async (params?: { page?: number; page_size?: number; search?: string }) => {
    return client.get('/v1/projects', { params }) as Promise<{ items: Project[], pagination: PaginationResponse }>
  },
  listSimple: async () => {
    return client.get('/v1/projects/simple') as Promise<Project[]>
  },
  create: async (data: any) => {
    return client.post('/v1/projects', data) as Promise<Project>
  },
  get: async (id: string) => {
    return client.get(`/v1/projects/${id}`) as Promise<Project>
  },

  listMembers: async (id: string, params?: PaginationParams) => {
    return client.get(`/v1/projects/${id}/members`, { params }) as Promise<{ items: ProjectMember[], pagination: PaginationResponse }>
  },

  inviteProjectMembers: async (projectId: string, data: { user_ids: string[]; role: string }) => {
    return client.post(`/v1/projects/${projectId}/members`, data)
  },

  removeProjectMember: async (projectId: string, userId: string) => {
    return client.delete(`/v1/projects/${projectId}/members`, {
      params: { user_id: userId }
    })
  },
  update: async (id: string, data: Partial<Project>) => {
    return client.put(`/v1/projects/${id}`, data) as Promise<Project>
  },
  delete: async (id: string) => {
    return client.delete(`/v1/projects/${id}`)
  },
}
