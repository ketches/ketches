import client from './client'

export interface User {
  id: string
  username: string
  email: string
  fullname?: string
  role: string
  created_at?: string
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
  pageSize?: number
  search?: string
}

export const usersApi = {
  list: async (params: ListUsersParams = {}) => {
    const searchParams = new URLSearchParams()
    if (params.page) searchParams.set('page', params.page.toString())
    if (params.pageSize) searchParams.set('page_size', params.pageSize.toString())
    if (params.search) searchParams.set('search', params.search)
    
    const query = searchParams.toString()
    const url = query ? `/v1/users?${query}` : '/v1/users'
    return client.get(url) as Promise<ListUsersResponse>
  },
  create: async (data: CreateUserRequest) => {
    return client.post('/v1/users', data) as Promise<User>
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
    return client.patch(`/v1/users/${id}/password`, { password })
  },
}
