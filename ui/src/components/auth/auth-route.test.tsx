import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom"

import { AuthRoute } from "./auth-route"

const { mockAuthState } = vi.hoisted(() => ({
  mockAuthState: {
    isAuthenticated: false,
    user: null as null | {
      id: string
      username: string
      email: string
      fullname?: string
      role: string
    },
  },
}))

vi.mock("@/stores/auth", () => ({
  useAuthStore: (selector: (state: typeof mockAuthState) => unknown) =>
    selector(mockAuthState),
}))

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

const AUTH_USER = {
  id: "user-1",
  username: "demo",
  email: "demo@example.com",
  fullname: "Demo User",
  role: "admin",
}

describe("AuthRoute", () => {
  beforeEach(() => {
    mockAuthState.isAuthenticated = false
    mockAuthState.user = null
  })

  afterEach(() => {
    document.body.innerHTML = ""
  })

  it("redirects authenticated login-page access back to the requested page", async () => {
    mockAuthState.isAuthenticated = true
    mockAuthState.user = AUTH_USER

    const container = document.createElement("div")
    document.body.appendChild(container)

    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(
        <MemoryRouter initialEntries={["/login?redirect=%2Fapplications%2Fapp-1%3Ftab%3Dlogs"]}>
          <Routes>
            <Route
              path="/login"
              element={
                <AuthRoute>
                  <LocationProbe />
                </AuthRoute>
              }
            />
            <Route path="/applications/:appId" element={<LocationProbe />} />
            <Route path="/" element={<LocationProbe />} />
          </Routes>
        </MemoryRouter>
      )
    })

    expect(container.querySelector('[data-testid="location"]')?.textContent).toBe(
      "/applications/app-1?tab=logs"
    )

    await act(async () => {
      root.unmount()
    })
  })

  it("falls back to the dashboard for unsafe redirect targets", async () => {
    mockAuthState.isAuthenticated = true
    mockAuthState.user = AUTH_USER

    const container = document.createElement("div")
    document.body.appendChild(container)

    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(
        <MemoryRouter initialEntries={["/login?redirect=https%3A%2F%2Fevil.example"]}>
          <Routes>
            <Route
              path="/login"
              element={
                <AuthRoute>
                  <LocationProbe />
                </AuthRoute>
              }
            />
            <Route path="/" element={<LocationProbe />} />
          </Routes>
        </MemoryRouter>
      )
    })

    expect(container.querySelector('[data-testid="location"]')?.textContent).toBe("/")

    await act(async () => {
      root.unmount()
    })
  })
})
