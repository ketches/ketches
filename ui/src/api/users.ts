import client from './client'

export interface User {
  id: string
  username: string
  email: string
  fullname?: string
  role: string
}

export const usersApi = {
  list: async () => {
    return client.get('/v1/users') as Promise<User[]>
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
