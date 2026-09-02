import { beforeEach, describe, expect, it, vi } from "vitest"

const authSessionMocks = vi.hoisted(() => ({
  beginSessionLogout: vi.fn(),
  completeSessionLogout: vi.fn(),
  cancelSessionLogout: vi.fn(),
}))

vi.mock("./auth-session", () => authSessionMocks)

import { logoutSession } from "./logout-session"

function createActions(requestLogout: () => Promise<void>) {
  return {
    requestLogout,
    markManualLogout: vi.fn(),
    clearQueries: vi.fn(),
    clearAuth: vi.fn(),
    navigateToLogin: vi.fn(),
  }
}

describe("logoutSession", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("waits for the server before clearing local session state", async () => {
    let resolveRequest: (() => void) | undefined
    const actions = createActions(() => new Promise<void>((resolve) => {
      resolveRequest = resolve
    }))

    const pending = logoutSession(actions)
    expect(actions.clearAuth).not.toHaveBeenCalled()
    expect(actions.navigateToLogin).not.toHaveBeenCalled()

    resolveRequest?.()
    await pending

    expect(authSessionMocks.beginSessionLogout).toHaveBeenCalledOnce()
    expect(authSessionMocks.completeSessionLogout).toHaveBeenCalledOnce()
    expect(authSessionMocks.cancelSessionLogout).not.toHaveBeenCalled()
    expect(actions.markManualLogout).toHaveBeenCalledOnce()
    expect(actions.clearQueries).toHaveBeenCalledOnce()
    expect(actions.clearAuth).toHaveBeenCalledOnce()
    expect(actions.navigateToLogin).toHaveBeenCalledOnce()
  })

  it("keeps local session state when the server logout fails", async () => {
    const actions = createActions(() => Promise.reject(new Error("network unavailable")))

    await expect(logoutSession(actions)).rejects.toThrow("network unavailable")
    expect(authSessionMocks.beginSessionLogout).toHaveBeenCalledOnce()
    expect(authSessionMocks.completeSessionLogout).not.toHaveBeenCalled()
    expect(authSessionMocks.cancelSessionLogout).toHaveBeenCalledOnce()
    expect(actions.markManualLogout).not.toHaveBeenCalled()
    expect(actions.clearQueries).not.toHaveBeenCalled()
    expect(actions.clearAuth).not.toHaveBeenCalled()
    expect(actions.navigateToLogin).not.toHaveBeenCalled()
  })
})
