import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import { AuthSessionManager, SESSION_REFRESH_INTERVAL_MS } from "./auth-session-manager"

const { mockAuthState, mockGetLastSessionRefreshAt, mockRefreshSession } = vi.hoisted(() => ({
  mockAuthState: {
    isAuthenticated: true,
  },
  mockGetLastSessionRefreshAt: vi.fn(() => 0),
  mockRefreshSession: vi.fn(async () => undefined),
}))

vi.mock("@/stores/auth", () => ({
  useAuthStore: (selector: (state: typeof mockAuthState) => unknown) => selector(mockAuthState),
}))

vi.mock("@/lib/auth-session", () => ({
  getLastSessionRefreshAt: () => mockGetLastSessionRefreshAt(),
}))

vi.mock("@/api/client", () => ({
  refreshSession: () => mockRefreshSession(),
}))

describe("AuthSessionManager", () => {
  beforeEach(() => {
    vi.useFakeTimers()
    mockAuthState.isAuthenticated = true
    mockGetLastSessionRefreshAt.mockReset()
    mockGetLastSessionRefreshAt.mockReturnValue(0)
    mockRefreshSession.mockReset()
    mockRefreshSession.mockResolvedValue(undefined)
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
})
