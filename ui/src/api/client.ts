import { buildUnauthenticatedLoginHref, getCurrentRelativePath } from '@/lib/auth-redirect'
import {
  applyCSRFHeader,
  clearPersistedAuthState,
  getCSRFToken,
  getSessionGeneration,
  isSessionLogoutInProgress,
  markSessionRefreshed,
  shouldAttachCSRF,
} from '@/lib/auth-session'
import { capitalizeDisplayMessage } from '@/lib/utils'
import { useAuthStore, type User } from '@/stores/auth'
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

export interface SessionRefreshResponse {
  user?: User
  must_change_password?: boolean
  default_password_notice?: string
}

type RefreshRequest = {
  generation: number
  promise: Promise<SessionRefreshResponse>
}

let refreshRequest: RefreshRequest | null = null
let isRedirectingToLogin = false

class SessionRefreshInvalidatedError extends Error {
  constructor() {
    super('Session refresh was invalidated')
    this.name = 'SessionRefreshInvalidatedError'
  }
}

function unwrapSessionRefreshResponse(responseData: unknown): SessionRefreshResponse {
  if (responseData && typeof responseData === 'object' && 'data' in responseData) {
    return ((responseData as { data?: SessionRefreshResponse }).data ?? {})
  }

  return (responseData ?? {}) as SessionRefreshResponse
}

function normalizeBackendMessagePayload(payload: unknown): void {
  if (!payload || typeof payload !== 'object' || Array.isArray(payload)) {
    return
  }

  const data = payload as Record<string, unknown>
  if (typeof data.error === 'string') {
    data.error = capitalizeDisplayMessage(data.error)
  }
  if (typeof data.message === 'string') {
    data.message = capitalizeDisplayMessage(data.message)
  }
}

function normalizeErrorResponseMessages(error: unknown): void {
  normalizeBackendMessagePayload((error as { response?: { data?: unknown } }).response?.data)
}

export function redirectToUnauthenticatedLogin(): void {
  if (isRedirectingToLogin) {
    return
  }

  isRedirectingToLogin = true
  clearPersistedAuthState()
  window.location.href = buildUnauthenticatedLoginHref(getCurrentRelativePath(window.location))
}

export async function refreshSession(options: { redirectOnFailure?: boolean } = {}): Promise<SessionRefreshResponse> {
  const { redirectOnFailure = true } = options

  if (isSessionLogoutInProgress()) {
    throw new SessionRefreshInvalidatedError()
  }

  const generation = getSessionGeneration()
  if (!refreshRequest || refreshRequest.generation !== generation) {
    const headers = applyCSRFHeader(new Headers(), 'POST')
    const promise = axios.post<unknown>(
      `${API_BASE_URL}/v1/users/refresh-token`,
      {},
      {
        withCredentials: true,
        headers: Object.fromEntries(headers.entries()),
      }
    ).then((response) => {
      const session = unwrapSessionRefreshResponse(response.data)
      if (generation !== getSessionGeneration() || isSessionLogoutInProgress()) {
        throw new SessionRefreshInvalidatedError()
      }

      markSessionRefreshed()
      if (session.user) {
        useAuthStore.getState().setAuth(session.user, session.must_change_password ?? false)
      }
      return session
    })

    const request: RefreshRequest = { generation, promise }
    refreshRequest = request
    void promise.then(
      () => {
        if (refreshRequest?.promise === promise) {
          refreshRequest = null
        }
      },
      () => {
        if (refreshRequest?.promise === promise) {
          refreshRequest = null
        }
      }
    )
  }

  try {
    return await refreshRequest.promise
  } catch (error) {
    if (
      redirectOnFailure &&
      !(error instanceof SessionRefreshInvalidatedError) &&
      !isSessionLogoutInProgress()
    ) {
      redirectToUnauthenticatedLogin()
    }
    throw error
  }
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
    const responseData = response.data
    normalizeBackendMessagePayload(responseData)

    if (responseData && Object.prototype.hasOwnProperty.call(responseData, 'data')) {
      const payload = (responseData as { data?: unknown }).data
      normalizeBackendMessagePayload(payload)
      return payload
    }
    return responseData
  },
  async (error) => {
    normalizeErrorResponseMessages(error)

    if (error.response?.status === 401) {
      const requestUrl = String(error.config?.url ?? '')
      const isAuthRequest =
        requestUrl.endsWith('/v1/users/sign-in') ||
        requestUrl.endsWith('/v1/users/sign-up') ||
        requestUrl.endsWith('/v1/users/sign-up/verification-code') ||
        requestUrl.endsWith('/v1/users/refresh-token')
      const originalRequest = error.config as typeof error.config & { _retry?: boolean }

      if (!isAuthRequest && isSessionLogoutInProgress()) {
        return Promise.reject(error)
      }

      if (!isAuthRequest && originalRequest && !originalRequest._retry) {
        originalRequest._retry = true
        try {
          await refreshSession({ redirectOnFailure: false })
          return client(originalRequest)
        } catch {
          redirectToUnauthenticatedLogin()
        }
      } else if (!isAuthRequest) {
        redirectToUnauthenticatedLogin()
      }
    }
    return Promise.reject(error)
  }
)

export default client
