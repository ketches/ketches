import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import { AuthSessionManager, SESSION_REFRESH_INTERVAL_MS } from "./auth-session-manager"

const { mockAuthState, mockGetLastSessionRefreshAt, mockRefreshSession } = vi.hoisted(() => ({
  mockAuthState: {
    hasCheckedSession: true,
    isAuthenticated: true,
    isRestoringSession: false,
    markSessionRestoreFinished: vi.fn(),
    markSessionRestoreStarted: vi.fn(),
    setAuth: vi.fn(),
    user: null as null | {
      id: string
      username: string
      email: string
      fullname?: string
      role: string
    },
  },
  mockGetLastSessionRefreshAt: vi.fn(() => 0),
  mockRefreshSession: vi.fn(async (_options?: unknown) => undefined as unknown),
}))

vi.mock("@/stores/auth", () => ({
  useAuthStore: (selector: (state: typeof mockAuthState) => unknown) => selector(mockAuthState),
}))

vi.mock("@/lib/auth-session", () => ({
  getLastSessionRefreshAt: () => mockGetLastSessionRefreshAt(),
}))

vi.mock("@/api/client", () => ({
  refreshSession: (options?: unknown) => mockRefreshSession(options),
}))

const AUTH_USER = {
  id: "user-1",
  username: "demo",
  email: "demo@example.com",
  fullname: "Demo User",
  role: "admin",
}

describe("AuthSessionManager", () => {
  beforeEach(() => {
    vi.useFakeTimers()
    mockAuthState.hasCheckedSession = true
    mockAuthState.isAuthenticated = true
    mockAuthState.isRestoringSession = false
    mockAuthState.user = null
    mockGetLastSessionRefreshAt.mockReset()
    mockGetLastSessionRefreshAt.mockReturnValue(0)
    mockRefreshSession.mockReset()
    mockRefreshSession.mockResolvedValue(undefined)
    mockAuthState.markSessionRestoreFinished.mockReset()
    mockAuthState.markSessionRestoreFinished.mockImplementation(() => {
      mockAuthState.hasCheckedSession = true
      mockAuthState.isRestoringSession = false
    })
    mockAuthState.markSessionRestoreStarted.mockReset()
    mockAuthState.markSessionRestoreStarted.mockImplementation(() => {
      mockAuthState.hasCheckedSession = false
      mockAuthState.isRestoringSession = true
    })
    mockAuthState.setAuth.mockReset()
    mockAuthState.setAuth.mockImplementation((user) => {
      mockAuthState.user = user
      mockAuthState.isAuthenticated = true
      mockAuthState.hasCheckedSession = true
      mockAuthState.isRestoringSession = false
    })
  })

  afterEach(() => {
    document.body.innerHTML = ""
    vi.useRealTimers()
  })

  it("refreshes immediately when the session is stale", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<AuthSessionManager />)
    })

    expect(mockRefreshSession).toHaveBeenCalledTimes(1)

    await act(async () => {
      root.unmount()
    })
  })

  it("refreshes on the background interval while authenticated", async () => {
    mockGetLastSessionRefreshAt.mockReturnValue(Date.now())

    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<AuthSessionManager />)
    })

    expect(mockRefreshSession).not.toHaveBeenCalled()

    await act(async () => {
      vi.advanceTimersByTime(SESSION_REFRESH_INTERVAL_MS)
    })

    expect(mockRefreshSession).toHaveBeenCalledTimes(1)

    await act(async () => {
      root.unmount()
    })
  })

  it("restores an authenticated user from the refresh cookie session", async () => {
    mockAuthState.hasCheckedSession = false
    mockAuthState.isAuthenticated = false
    mockAuthState.isRestoringSession = false
    mockRefreshSession.mockResolvedValue({ user: AUTH_USER, must_change_password: true })

    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<AuthSessionManager />)
      await Promise.resolve()
      await Promise.resolve()
    })

    expect(mockAuthState.markSessionRestoreStarted).toHaveBeenCalledTimes(1)
    expect(mockRefreshSession).toHaveBeenCalledWith({ redirectOnFailure: false })
    expect(mockAuthState.setAuth).toHaveBeenCalledWith(AUTH_USER, true)
    expect(mockAuthState.markSessionRestoreFinished).toHaveBeenCalledTimes(1)

    await act(async () => {
      root.unmount()
    })
  })
})
