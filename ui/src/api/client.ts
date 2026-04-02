import { buildUnauthenticatedLoginHref, getCurrentRelativePath } from '@/lib/auth-redirect'
import { clearAuthCookie, clearPersistedAuthState, getStoredAccessToken, syncAuthCookie } from '@/lib/auth-session'
import axios, { type AxiosInstance } from 'axios'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api'

const client: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

client.interceptors.request.use((config) => {
  const accessToken = getStoredAccessToken()
  if (accessToken) {
    syncAuthCookie()
    config.headers.Authorization = `Bearer ${accessToken}`
  }
  return config
})

client.interceptors.response.use(
  (response) => {
    if (response.data && Object.prototype.hasOwnProperty.call(response.data, 'data')) {
      return response.data.data
    }
    return response.data
  },
  (error) => {
    if (error.response?.status === 401) {
      const requestUrl = String(error.config?.url ?? '')
      const isAuthRequest = requestUrl.endsWith('/v1/users/sign-in') || requestUrl.endsWith('/v1/users/sign-up')

      if (!isAuthRequest) {
        clearAuthCookie()
        clearPersistedAuthState()
        window.location.href = buildUnauthenticatedLoginHref(getCurrentRelativePath(window.location))
      }
    }
    return Promise.reject(error)
  }
)

export default client
