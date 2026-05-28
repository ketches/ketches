import { beforeEach, describe, expect, it, vi } from "vitest"

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
  const buildUnauthenticatedLoginHrefMock = vi.fn(() => "/login")
  const getCurrentRelativePathMock = vi.fn(() => "/projects")

  function reset() {
    requestAttempts.clear()
    refreshPostMock.mockClear()
    refreshPostMock.mockImplementation(async () => ({ data: { data: {} } }))
    clearPersistedAuthStateMock.mockClear()
    markSessionRefreshedMock.mockClear()
    buildUnauthenticatedLoginHrefMock.mockClear()
    getCurrentRelativePathMock.mockClear()
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
    clientInstance,
    getCurrentRelativePathMock,
    refreshPostMock,
    reset,
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
  clearPersistedAuthState: testState.clearPersistedAuthStateMock,
  markSessionRefreshed: testState.markSessionRefreshedMock,
  getCSRFToken: vi.fn(() => "csrf-token"),
  shouldAttachCSRF: vi.fn(() => true),
}))

vi.mock("@/lib/auth-redirect", () => ({
  buildUnauthenticatedLoginHref: testState.buildUnauthenticatedLoginHrefMock,
  getCurrentRelativePath: testState.getCurrentRelativePathMock,
}))

import client from "./client"

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
})
