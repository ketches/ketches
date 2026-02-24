import client from './client'

export interface SignInResponse {
  user: {
    id: string
    username: string
    email: string
    fullname: string
    role: string
  }
  access_token: string
  refresh_token: string
}

export const authApi = {
  signIn: async (data: any) => {
    return client.post('/v1/users/sign-in', data) as Promise<SignInResponse>
  },
  signUp: async (data: any) => {
    return client.post('/v1/users/sign-up', data)
  },
}
