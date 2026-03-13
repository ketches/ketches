import { describe, expect, it } from "vitest"

import {
  buildLoginHref,
  buildUnauthenticatedLoginHref,
  getSafeRedirectTarget,
  markManualLogout,
  MANUAL_LOGOUT_MARKER_TTL_MS,
} from "./auth-redirect"

function createMemoryStorage(): Storage {
  const store = new Map<string, string>()

  return {
    get length() {
      return store.size
    },
    clear() {
      store.clear()
    },
    getItem(key) {
      return store.get(key) ?? null
    },
    key(index) {
      return Array.from(store.keys())[index] ?? null
    },
    removeItem(key) {
      store.delete(key)
    },
    setItem(key, value) {
      store.set(key, value)
    },
  }
}

describe("auth redirect helpers", () => {
  it("builds a login URL that preserves the current in-app path", () => {
    expect(buildLoginHref("/applications/app-1?tab=logs#metrics")).toBe(
      "/login?redirect=%2Fapplications%2Fapp-1%3Ftab%3Dlogs%23metrics"
    )
  })

  it("does not attach login or signup paths as redirect targets", () => {
    expect(buildLoginHref("/login")).toBe("/login")
    expect(buildLoginHref("/signup?invite=1")).toBe("/login")
  })

  it("reads a safe redirect target from the login page search string", () => {
    expect(
      getSafeRedirectTarget("?redirect=%2Fapplications%2Fapp-1%3Ftab%3Dlogs")
    ).toBe("/applications/app-1?tab=logs")
  })

  it("rejects unsafe redirect targets such as external URLs", () => {
    expect(
      getSafeRedirectTarget("?redirect=https%3A%2F%2Fevil.example")
    ).toBeNull()
    expect(getSafeRedirectTarget("?redirect=%2F%2Fevil.example")).toBeNull()
    expect(getSafeRedirectTarget("?redirect=%2Flogin")).toBeNull()
  })

  it("does not preserve redirect targets during the manual logout window", () => {
    const storage = createMemoryStorage()

    markManualLogout(storage, 1_000)

    expect(
      buildUnauthenticatedLoginHref("/applications/app-1?tab=logs", storage, 1_001)
    ).toBe("/login")
  })

  it("preserves redirect targets again after the manual logout marker expires", () => {
    const storage = createMemoryStorage()

    markManualLogout(storage, 1_000)

    expect(
      buildUnauthenticatedLoginHref(
        "/applications/app-1?tab=logs",
        storage,
        1_000 + MANUAL_LOGOUT_MARKER_TTL_MS + 1
      )
    ).toBe("/login?redirect=%2Fapplications%2Fapp-1%3Ftab%3Dlogs")
  })

  it("keeps an existing safe login redirect during the manual logout window", () => {
    const storage = createMemoryStorage()

    markManualLogout(storage, 1_000)

    expect(
      buildUnauthenticatedLoginHref(
        "/login?redirect=%2Fapplications%2Fapp-1%3Ftab%3Dlogs",
        storage,
        1_001
      )
    ).toBe("/login?redirect=%2Fapplications%2Fapp-1%3Ftab%3Dlogs")
  })
})
