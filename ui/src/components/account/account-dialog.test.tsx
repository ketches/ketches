import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

vi.mock("@/api/users", () => ({
  usersApi: {
    getMe: vi.fn(),
    updateMyProfile: vi.fn(),
    updateMyPassword: vi.fn(),
  },
}))

vi.mock("@tanstack/react-query", () => ({
  useQuery: ({ queryFn }: { queryFn?: () => Promise<unknown> }) => {
    if (queryFn) {
      void queryFn()
    }
    return { data: undefined }
  },
  useMutation: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
  useQueryClient: () => ({
    setQueryData: vi.fn(),
    invalidateQueries: vi.fn(),
  }),
}))

vi.mock("@/stores/auth", () => ({
  useAuthStore: (selector: (state: { user: { id: string }, updateUser: (data: unknown) => void }) => unknown) =>
    selector({ user: { id: "user-1" }, updateUser: vi.fn() }),
}))

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

vi.mock("@/components/ui/dialog", () => ({
  Dialog: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogTitle: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/sidebar", () => ({
  SidebarProvider: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  Sidebar: ({ children }: { children: React.ReactNode }) => <nav>{children}</nav>,
  SidebarContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  SidebarGroup: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  SidebarGroupContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  SidebarMenu: ({ children }: { children: React.ReactNode }) => <ul>{children}</ul>,
  SidebarMenuItem: ({ children }: { children: React.ReactNode }) => <li>{children}</li>,
  SidebarMenuButton: ({ children, onClick, isActive }: { children: React.ReactNode; onClick?: () => void; isActive?: boolean }) => (
    <button type="button" onClick={onClick} aria-current={isActive ? "page" : undefined}>{children}</button>
  ),
}))

import { AccountDialog } from "./account-dialog"
import { usersApi } from "@/api/users"

describe("AccountDialog", () => {
  beforeEach(() => {
    ;(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true
    vi.mocked(usersApi.getMe).mockResolvedValue({
      id: "user-1",
      username: "demo",
      fullname: "Demo User",
      email: "demo@example.com",
      bio: "Bio",
      role: "user",
      created_at: new Date().toISOString(),
    })
  })

  afterEach(() => {
    document.body.innerHTML = ""
    vi.clearAllMocks()
  })

  it("renders sidebar-style sections for profile, security, and AI providers", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(
        <AccountDialog
          open
          onOpenChange={vi.fn()}
          user={{
            fullname: "Demo User",
            email: "demo@example.com",
            bio: "Bio",
            avatar: "",
          }}
        />
      )
    })

    expect(container.textContent).toContain("Account Settings")
    expect(container.textContent).toContain("Profile")
    expect(container.textContent).toContain("Security")
    expect(container.textContent).toContain("AI Providers")
    expect(container.textContent).toContain("Full Name")

    const securityButton = Array.from(container.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("Security")
    ) as HTMLButtonElement | undefined

    await act(async () => {
      securityButton?.click()
    })

    expect(container.textContent).toContain("Current Password")

    const aiProvidersButton = Array.from(container.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("AI Providers")
    ) as HTMLButtonElement | undefined

    await act(async () => {
      aiProvidersButton?.click()
    })

    expect(container.textContent).toContain("Configure your personal AI providers")

    await act(async () => {
      root.unmount()
    })
  })
})
