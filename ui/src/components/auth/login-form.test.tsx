import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const {
  mockSignIn,
  mockPlatformCheck,
  mockSetAuth,
  mockNavigate,
  mockFormValues,
  mockQueryClientClear,
} = vi.hoisted(() => ({
  mockSignIn: vi.fn(),
  mockPlatformCheck: vi.fn(),
  mockSetAuth: vi.fn(),
  mockNavigate: vi.fn(),
  mockFormValues: {} as Record<string, string>,
  mockQueryClientClear: vi.fn(),
}))

vi.mock("@/api/auth", () => ({
  authApi: {
    signIn: (...args: unknown[]) => mockSignIn(...args),
  },
}))

vi.mock("@/api/platform-update", () => ({
  platformUpdateApi: {
    check: (...args: unknown[]) => mockPlatformCheck(...args),
  },
}))

vi.mock("@/stores/auth", () => ({
  useAuthStore: (selector: (state: { setAuth: typeof mockSetAuth }) => unknown) =>
    selector({ setAuth: mockSetAuth }),
}))

vi.mock("@hookform/resolvers/zod", () => ({
  zodResolver: () => undefined,
}))

vi.mock("@tanstack/react-query", () => ({
  useQueryClient: () => ({
    clear: mockQueryClientClear,
  }),
}))

vi.mock("react-hook-form", () => ({
  useForm: () => ({
    register: (name: string) => ({
      name,
      onChange: (event: Event) => {
        const target = event.target as HTMLInputElement
        mockFormValues[name] = target.value
      },
      ref: () => undefined,
    }),
    handleSubmit: (submit: (values: Record<string, string>) => Promise<void>) => {
      return async (event?: Event) => {
        event?.preventDefault?.()
        await submit({
          username: mockFormValues.username ?? "",
          password: mockFormValues.password ?? "",
        })
      }
    },
    formState: {
      errors: {},
    },
  }),
}))

vi.mock("@/lib/auth-redirect", () => ({
  clearManualLogoutMarker: vi.fn(),
  getPostLoginTarget: () => "/",
}))

vi.mock("@/lib/auth-session", () => ({
  markSessionRefreshed: vi.fn(),
}))

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
    warning: vi.fn(),
  },
}))

vi.mock("react-router-dom", () => ({
  Link: ({ children, to }: { children: React.ReactNode; to: string }) => <a href={to}>{children}</a>,
  useNavigate: () => mockNavigate,
  useLocation: () => ({ search: "" }),
}))

import { LoginForm } from "./login-form"

describe("LoginForm", () => {
  beforeEach(() => {
    mockFormValues.username = ""
    mockFormValues.password = ""
    mockQueryClientClear.mockReset()
    mockSignIn.mockResolvedValue({
      user: {
        id: "admin-1",
        username: "admin",
        email: "admin@example.com",
        fullname: "Admin",
        role: "admin",
      },
      access_token: "access",
      refresh_token: "refresh",
      must_change_password: false,
      default_password_notice: "",
    })
    mockPlatformCheck.mockResolvedValue(undefined)
  })

  afterEach(() => {
    document.body.innerHTML = ""
    vi.clearAllMocks()
  })

  it("fires a silent admin-only auto-check after successful login", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<LoginForm />)
    })

    const usernameInput = container.querySelector("#username") as HTMLInputElement | null
    const passwordInput = container.querySelector("#password") as HTMLInputElement | null
    const submitButton = Array.from(container.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("Login")
    )

    await act(async () => {
      if (usernameInput) {
        usernameInput.value = "admin"
        usernameInput.dispatchEvent(new Event("change", { bubbles: true }))
      }
      if (passwordInput) {
        passwordInput.value = "secret"
        passwordInput.dispatchEvent(new Event("change", { bubbles: true }))
      }
    })

    await act(async () => {
      submitButton?.dispatchEvent(new MouseEvent("click", { bubbles: true }))
    })

    expect(mockSignIn).toHaveBeenCalled()
    expect(mockPlatformCheck).toHaveBeenCalledWith({ mode: "auto" })

    await act(async () => {
      root.unmount()
    })
  })
})
