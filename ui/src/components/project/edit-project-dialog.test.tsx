import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

vi.mock("@/api/projects", () => ({
  projectsApi: {
    update: vi.fn(),
    listAiProviders: vi.fn(),
  },
}))

vi.mock("@tanstack/react-query", () => ({
  useQuery: ({ queryFn }: { queryFn?: () => Promise<unknown> }) => {
    if (queryFn) {
      void queryFn()
    }
    return { data: [] }
  },
  useMutation: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
  useQueryClient: () => ({
    invalidateQueries: vi.fn(),
  }),
}))

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

vi.mock("@/components/ui/dialog", () => ({
  Dialog: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogTitle: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogDescription: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogFooter: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
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

import { EditProjectDialog } from "./edit-project-dialog"

describe("EditProjectDialog", () => {
  beforeEach(() => {
    ;(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true
  })

  afterEach(() => {
    document.body.innerHTML = ""
    vi.clearAllMocks()
  })

  it("renders sidebar-style general and AI provider sections", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(
        <EditProjectDialog
          open
          onOpenChange={vi.fn()}
          project={{
            id: "project-1",
            name: "Demo Project",
            slug: "demo-project",
            description: "Demo description",
            collaboration_enabled: true,
            created_at: new Date().toISOString(),
          }}
        />
      )
    })

    expect(container.textContent).toContain("Edit Project")
    expect(container.textContent).toContain("General")
    expect(container.textContent).toContain("AI Providers")
    expect(container.textContent).toContain("Name")
    expect(container.textContent).toContain("Description")

    const providersButton = Array.from(container.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("AI Providers")
    ) as HTMLButtonElement | undefined

    await act(async () => {
      providersButton?.click()
    })

    expect(container.textContent).toContain("Configure project-level AI providers")

    await act(async () => {
      root.unmount()
    })
  })
})
