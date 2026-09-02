import { beforeEach, describe, expect, it, vi } from "vitest"
import type { User } from "@/stores/auth"

type RequestConfig = {
  url?: string
  method?: string
  headers?: Record<string, string>
  _retry?: boolean
}

type Response = {
  data: unknown
  config: RequestConfig
}

type AxiosErrorLike = Error & {
  response?: {
    status?: number
    data?: unknown
  }
  config?: RequestConfig
}

const testState = vi.hoisted(() => {
  const requestInterceptors: Array<(config: RequestConfig) => RequestConfig> = []
  const responseSuccessInterceptors: Array<(response: Response) => unknown> = []
  const responseErrorInterceptors: Array<(error: AxiosErrorLike) => Promise<unknown>> = []
  const requestAttempts = new Map<string, number>()
  const refreshPostMock = vi.fn(async () => ({ data: { data: {} } }))
  const clearPersistedAuthStateMock = vi.fn()
  const markSessionRefreshedMock = vi.fn()
  const beginSessionLogoutMock = vi.fn()
  const completeSessionLogoutMock = vi.fn()
  let sessionGeneration = 0
  let sessionLogoutInProgress = false
  const cancelSessionLogoutMock = vi.fn(() => {
    sessionGeneration += 1
    sessionLogoutInProgress = false
  })
  const buildUnauthenticatedLoginHrefMock = vi.fn(() => "/login")
  const getCurrentRelativePathMock = vi.fn(() => "/projects")
  const setAuthMock = vi.fn()

  function reset() {
    requestAttempts.clear()
    refreshPostMock.mockClear()
    refreshPostMock.mockImplementation(async () => ({ data: { data: {} } }))
    clearPersistedAuthStateMock.mockClear()
    markSessionRefreshedMock.mockClear()
    beginSessionLogoutMock.mockClear()
    completeSessionLogoutMock.mockClear()
    cancelSessionLogoutMock.mockClear()
    sessionGeneration = 0
    sessionLogoutInProgress = false
    buildUnauthenticatedLoginHrefMock.mockClear()
    getCurrentRelativePathMock.mockClear()
    setAuthMock.mockClear()
  }

  async function dispatch(config: RequestConfig): Promise<unknown> {
    let nextConfig: RequestConfig = {
      ...config,
      headers: { ...(config.headers ?? {}) },
    }
    for (const interceptor of requestInterceptors) {
      nextConfig = interceptor(nextConfig)
    }

    try {
      const response = await transport(nextConfig)
      let nextResponse: unknown = response
      for (const interceptor of responseSuccessInterceptors) {
        nextResponse = interceptor(nextResponse as Response)
      }
      return nextResponse
    } catch (error) {
      const nextError = error as AxiosErrorLike
      for (const interceptor of responseErrorInterceptors) {
        return interceptor(nextError)
      }
      throw nextError
    }
  }

  async function transport(config: RequestConfig): Promise<Response> {
    const url = String(config.url ?? "")
    const attempt = (requestAttempts.get(url) ?? 0) + 1
    requestAttempts.set(url, attempt)

    if ((url === "/v1/projects" || url === "/v1/environments") && attempt === 1) {
      const error = new Error("Unauthorized") as AxiosErrorLike
      error.response = { status: 401 }
      error.config = config
      throw error
    }

    if (url === "/v1/lowercase-error") {
      const error = new Error("Bad request") as AxiosErrorLike
      error.response = {
        status: 400,
        data: {
          error: "project already exists",
        },
      }
      error.config = config
      throw error
    }

    if (url === "/v1/lowercase-message") {
      return {
        data: {
          data: {
            message: "namespace is available",
          },
        },
        config,
      }
    }

    return {
      data: {
        data: {
          ok: true,
          url,
          attempt,
        },
      },
      config,
    }
  }

  const clientInstance = Object.assign(
    vi.fn((config: RequestConfig) => dispatch(config)),
    {
      get: (url: string, config?: RequestConfig) =>
        dispatch({
          ...config,
          url,
          method: "get",
        }),
      interceptors: {
        request: {
          use: (interceptor: (config: RequestConfig) => RequestConfig) => {
            requestInterceptors.push(interceptor)
            return requestInterceptors.length - 1
          },
        },
        response: {
          use: (
            onFulfilled: (response: Response) => unknown,
            onRejected: (error: AxiosErrorLike) => Promise<unknown>
          ) => {
            responseSuccessInterceptors.push(onFulfilled)
            responseErrorInterceptors.push(onRejected)
            return responseSuccessInterceptors.length - 1
          },
        },
      },
    }
  )

  return {
    buildUnauthenticatedLoginHrefMock,
    clearPersistedAuthStateMock,
    markSessionRefreshedMock,
    beginSessionLogoutMock,
    completeSessionLogoutMock,
    cancelSessionLogoutMock,
    getSessionGeneration: () => sessionGeneration,
    isSessionLogoutInProgress: () => sessionLogoutInProgress,
    startLogout: () => {
      sessionGeneration += 1
      sessionLogoutInProgress = true
    },
    clientInstance,
    getCurrentRelativePathMock,
    refreshPostMock,
    reset,
    setAuthMock,
  }
})

vi.mock("axios", () => ({
  default: {
    create: vi.fn(() => testState.clientInstance),
    post: testState.refreshPostMock,
  },
}))

vi.mock("@/lib/auth-session", () => ({
  applyCSRFHeader: vi.fn((headers: Headers) => headers),
  beginSessionLogout: () => testState.startLogout(),
  cancelSessionLogout: testState.cancelSessionLogoutMock,
  clearPersistedAuthState: testState.clearPersistedAuthStateMock,
  completeSessionLogout: testState.completeSessionLogoutMock,
  getSessionGeneration: testState.getSessionGeneration,
  isSessionLogoutInProgress: testState.isSessionLogoutInProgress,
  markSessionRefreshed: testState.markSessionRefreshedMock,
  getCSRFToken: vi.fn(() => "csrf-token"),
  shouldAttachCSRF: vi.fn(() => true),
}))

vi.mock("@/lib/auth-redirect", () => ({
  buildUnauthenticatedLoginHref: testState.buildUnauthenticatedLoginHrefMock,
  getCurrentRelativePath: testState.getCurrentRelativePathMock,
}))

vi.mock("@/stores/auth", () => ({
  useAuthStore: {
    getState: () => ({ setAuth: testState.setAuthMock }),
  },
}))

import client, { refreshSession } from "./client"

describe("api client refresh handling", () => {
  beforeEach(() => {
    testState.reset()
  })

  it("reuses one refresh request for concurrent unauthorized responses", async () => {
    await expect(
      Promise.all([
        client.get("/v1/projects"),
        client.get("/v1/environments"),
      ])
    ).resolves.toEqual([
      { ok: true, url: "/v1/projects", attempt: 2 },
      { ok: true, url: "/v1/environments", attempt: 2 },
    ])

    expect(testState.refreshPostMock).toHaveBeenCalledTimes(1)
    expect(testState.markSessionRefreshedMock).toHaveBeenCalledTimes(1)
    expect(testState.clearPersistedAuthStateMock).not.toHaveBeenCalled()
  })

  it("capitalizes API error response messages before rejection", async () => {
    await expect(client.get("/v1/lowercase-error")).rejects.toMatchObject({
      response: {
        data: {
          error: "Project already exists",
        },
      },
    })
  })

  it("capitalizes API success response messages before unwrapping", async () => {
    await expect(client.get("/v1/lowercase-message")).resolves.toEqual({
      message: "Namespace is available",
    })
  })

  it("updates the auth store when a refresh returns changed user state", async () => {
    const refreshedUser = {
      id: "user-1",
      username: "alice",
      email: "alice@example.com",
      role: "admin",
    }
    testState.refreshPostMock.mockResolvedValueOnce({
      data: { data: { user: refreshedUser, must_change_password: true } },
    })

    await expect(refreshSession({ redirectOnFailure: false })).resolves.toMatchObject({ user: refreshedUser })
    expect(testState.setAuthMock).toHaveBeenCalledWith(refreshedUser, true)
  })

  it("ignores a refresh response that completes after logout starts", async () => {
    let resolveRefresh: ((response: { data: { data: { user: User } } }) => void) | undefined
    testState.refreshPostMock.mockImplementationOnce(() => new Promise((resolve) => {
      resolveRefresh = resolve
    }))

    const pendingRefresh = refreshSession()
    testState.startLogout()
    testState.cancelSessionLogoutMock()
    resolveRefresh?.({
      data: {
        data: {
          user: {
            id: "user-1",
            username: "alice",
            email: "alice@example.com",
            role: "admin",
          },
        },
      },
    })

    await expect(pendingRefresh).rejects.toThrow("Session refresh was invalidated")
    expect(testState.setAuthMock).not.toHaveBeenCalled()
    expect(testState.markSessionRefreshedMock).not.toHaveBeenCalled()
    expect(testState.clearPersistedAuthStateMock).not.toHaveBeenCalled()
    expect(testState.buildUnauthenticatedLoginHrefMock).not.toHaveBeenCalled()
  })
})
