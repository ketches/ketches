import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom"

import { markManualLogout } from "@/lib/auth-redirect"

import { ProtectedRoute } from "./protected-route"

const { mockAuthState } = vi.hoisted(() => ({
  mockAuthState: {
    isAuthenticated: false,
    user: null,
  },
}))

vi.mock("@/stores/auth", () => ({
  useAuthStore: (selector: (state: typeof mockAuthState) => unknown) =>
    selector(mockAuthState),
}))

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

function LocationProbe() {
  const location = useLocation()

  return (
    <div data-testid="location">
      {location.pathname}
      {location.search}
      {location.hash}
    </div>
  )
}

describe("ProtectedRoute", () => {
  beforeEach(() => {
    mockAuthState.isAuthenticated = false
    mockAuthState.user = null
    Object.defineProperty(globalThis, "sessionStorage", {
      configurable: true,
      value: createMemoryStorage(),
    })
  })

  afterEach(() => {
    document.body.innerHTML = ""
  })

  it("redirects unauthenticated access to login with the current path", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)

    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(
        <MemoryRouter initialEntries={["/applications/app-1?tab=logs#metrics"]}>
          <Routes>
            <Route path="/login" element={<LocationProbe />} />
            <Route
              path="/applications/:appId"
              element={
                <ProtectedRoute>
                  <LocationProbe />
                </ProtectedRoute>
              }
            />
          </Routes>
        </MemoryRouter>
      )
    })

    expect(container.querySelector('[data-testid="location"]')?.textContent).toBe(
      "/login?redirect=%2Fapplications%2Fapp-1%3Ftab%3Dlogs%23metrics"
    )

    await act(async () => {
      root.unmount()
    })
  })

  it("still preserves redirect when the user explicitly visits a protected page after manual logout", async () => {
    markManualLogout(sessionStorage, Date.now())

    const container = document.createElement("div")
    document.body.appendChild(container)

    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(
        <MemoryRouter initialEntries={["/applications/app-1?tab=logs"]}>
          <Routes>
            <Route path="/login" element={<LocationProbe />} />
            <Route
              path="/applications/:appId"
              element={
                <ProtectedRoute>
                  <LocationProbe />
                </ProtectedRoute>
              }
            />
          </Routes>
        </MemoryRouter>
      )
    })

    expect(container.querySelector('[data-testid="location"]')?.textContent).toBe(
      "/login?redirect=%2Fapplications%2Fapp-1%3Ftab%3Dlogs"
    )

    await act(async () => {
      root.unmount()
    })
  })
})
