import client from './client'

export interface Template {
  id: string
  project_id: string
  name: string
  slug: string
  description: string
  type: string
  content: string
  status: string
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface CreateTemplateRequest {
  name: string
  slug: string
  description?: string
  type?: string
  content?: string
  status?: string
  enabled?: boolean
}

export interface UpdateTemplateRequest {
  name?: string
  slug?: string
  description?: string
  type?: string
  content?: string
  status?: string
  enabled?: boolean
}

export const templatesApi = {
  list: async (projectId: string) => {
    return client.get(`/v1/projects/${projectId}/templates`) as Promise<Template[]>
  },
  create: async (projectId: string, data: CreateTemplateRequest) => {
    return client.post(`/v1/projects/${projectId}/templates`, data) as Promise<Template>
  },
  get: async (templateId: string) => {
    return client.get(`/v1/templates/${templateId}`) as Promise<Template>
  },
  update: async (templateId: string, data: UpdateTemplateRequest) => {
    return client.put(`/v1/templates/${templateId}`, data) as Promise<Template>
  },
  delete: async (templateId: string) => {
    return client.delete(`/v1/templates/${templateId}`)
  },
}
