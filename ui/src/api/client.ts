import { buildUnauthenticatedLoginHref, getCurrentRelativePath } from '@/lib/auth-redirect'
import { applyCSRFHeader, clearPersistedAuthState, getCSRFToken, shouldAttachCSRF } from '@/lib/auth-session'
import axios, { type AxiosInstance } from 'axios'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api'

const client: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  withCredentials: true,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

let refreshRequest: Promise<void> | null = null

async function refreshSession(): Promise<void> {
  if (!refreshRequest) {
    const headers = applyCSRFHeader(new Headers(), 'POST')
    refreshRequest = axios.post(
      `${API_BASE_URL}/v1/users/refresh-token`,
      {},
      {
        withCredentials: true,
        headers: Object.fromEntries(headers.entries()),
      }
    ).then(() => undefined).finally(() => {
      refreshRequest = null
    })
  }

  return refreshRequest
}

client.interceptors.request.use((config) => {
  if (shouldAttachCSRF(config.method)) {
    const csrfToken = getCSRFToken()
    if (csrfToken) {
      config.headers['X-CSRF-Token'] = csrfToken
    }
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
  async (error) => {
    if (error.response?.status === 401) {
      const requestUrl = String(error.config?.url ?? '')
      const isAuthRequest =
        requestUrl.endsWith('/v1/users/sign-in') ||
        requestUrl.endsWith('/v1/users/sign-up') ||
        requestUrl.endsWith('/v1/users/sign-up/verification-code') ||
        requestUrl.endsWith('/v1/users/refresh-token')
      const originalRequest = error.config as typeof error.config & { _retry?: boolean }

      if (!isAuthRequest && originalRequest && !originalRequest._retry) {
        originalRequest._retry = true
        try {
          await refreshSession()
          return client(originalRequest)
        } catch {
          clearPersistedAuthState()
          window.location.href = buildUnauthenticatedLoginHref(getCurrentRelativePath(window.location))
        }
      } else if (!isAuthRequest) {
        clearPersistedAuthState()
        window.location.href = buildUnauthenticatedLoginHref(getCurrentRelativePath(window.location))
      }
    }
    return Promise.reject(error)
  }
)

export default client
