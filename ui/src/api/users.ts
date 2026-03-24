import client from './client'

export interface User {
  id: string
  username: string
  email: string
  fullname?: string
  bio?: string
  role: string
  created_at: string
}

export interface CreateUserRequest {
  username: string
  email: string
  password: string
  fullname?: string
  phone?: string
  role: string
}

export interface BatchImportResponse {
  succeeded: number
  failed: number
  errors: { index: number; message: string }[]
  users: User[]
}

export interface ListUsersResponse {
  total: number
  page: number
  page_size: number
  users: User[]
}

export interface ListUsersParams {
  page?: number
  page_size?: number
  search?: string
}

export interface UpdateMyProfileRequest {
  fullname: string
  email: string
  bio: string
}

export interface UpdateMyPasswordRequest {
  current_password: string
  new_password: string
}

export interface UserAiProvider {
  id: string
  provider_key: string
  display_name: string
  base_url: string
  default_model_profile_key: string
  default_model?: string
  enabled: boolean
  created_at: string
}

export interface UpsertMyAiProviderRequest {
  provider_key: string
  display_name: string
  base_url: string
  api_key: string
  default_model_profile_key: string
  enabled: boolean
}

export const usersApi = {
  list: async (params: ListUsersParams = {}) => {
    const searchParams = new URLSearchParams()
    if (params.page) searchParams.set('page', params.page.toString())
    if (params.page_size) searchParams.set('page_size', params.page_size.toString())
    if (params.search) searchParams.set('search', params.search)

    const query = searchParams.toString()
    const url = query ? `/v1/users?${query}` : '/v1/users'
    return client.get(url) as Promise<ListUsersResponse>
  },
  create: async (data: CreateUserRequest) => {
    return client.post('/v1/users', data) as Promise<User>
  },
  getMe: async () => {
    return client.get('/v1/users/me') as Promise<User>
  },
  updateMyProfile: async (data: UpdateMyProfileRequest) => {
    return client.put('/v1/users/me/profile', data) as Promise<User>
  },
  updateMyPassword: async (data: UpdateMyPasswordRequest) => {
    return client.patch('/v1/users/me/password', data) as Promise<void>
  },
  listMyAiProviders: async () => {
    return client.get('/v1/users/me/ai-providers') as Promise<UserAiProvider[]>
  },
  createMyAiProvider: async (data: UpsertMyAiProviderRequest) => {
    return client.post('/v1/users/me/ai-providers', data) as Promise<UserAiProvider>
  },
  updateMyAiProvider: async (id: string, data: UpsertMyAiProviderRequest) => {
    return client.put(`/v1/users/me/ai-providers/${id}`, data) as Promise<UserAiProvider>
  },
  importUsers: async (file: File, type: 'json' | 'csv' | 'excel' = 'json') => {
    const formData = new FormData()
    formData.append('file', file)
    formData.append('type', type)
    return client.post('/v1/users/import', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    }) as Promise<BatchImportResponse>
  },
  delete: async (id: string) => {
    return client.delete(`/v1/users/${id}`)
  },
  updateRole: async (id: string, role: string) => {
    return client.patch(`/v1/users/${id}/role`, { role })
  },
  updatePassword: async (id: string, password: string) => {
    return client.patch(`/v1/users/${id}/password`, { password }) as Promise<void>
  },
}
