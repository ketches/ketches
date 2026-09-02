import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const {
  pageHeaderItemsMock,
  navigateMock,
  setActiveContextWithNamesMock,
} = vi.hoisted(() => ({
  pageHeaderItemsMock: vi.fn(),
  navigateMock: vi.fn(),
  setActiveContextWithNamesMock: vi.fn(),
}))

const PROJECT = {
  id: "project-1",
  slug: "project-1",
  name: "Project One",
  description: "Primary project",
  collaboration_enabled: true,
  owner_name: "Owner One",
  created_at: "2026-03-01T00:00:00Z",
} as const

const OTHER_PROJECT = {
  id: "project-2",
  slug: "project-2",
  name: "Project Two",
  description: "Secondary project",
  collaboration_enabled: true,
  owner_name: "Owner Two",
  created_at: "2026-03-02T00:00:00Z",
} as const

vi.mock("@tanstack/react-query", async () => {
  const actual = await vi.importActual<typeof import("@tanstack/react-query")>("@tanstack/react-query")

  return {
    ...actual,
    useMutation: () => ({
      mutate: vi.fn(),
      isPending: false,
    }),
  }
})

vi.mock("react-router-dom", () => ({
  useNavigate: () => navigateMock,
  useParams: () => ({ projectId: PROJECT.id }),
}))

vi.mock("@/hooks/useProjectRole", () => ({
  useProjectRole: () => "owner",
}))

vi.mock("@/stores/auth", () => ({
  useAuthStore: (selector: (state: { user: { role: string } }) => unknown) =>
    selector({ user: { role: "admin" } }),
}))

vi.mock("@/stores/project", () => ({
  useProjectStore: (selector?: (state: {
    activeProjectId: string | null
    setActiveContextWithNames: (...args: [string | null, string | null, string | null, string | null]) => void
  }) => unknown) => {
    const state = {
      activeProjectId: null,
      setActiveContextWithNames: setActiveContextWithNamesMock,
    }

    return selector ? selector(state) : state
  },
}))

vi.mock("@/api/projects", async () => {
  const actual = await vi.importActual<typeof import("@/api/projects")>("@/api/projects")

  return {
    ...actual,
    projectsApi: {
      ...actual.projectsApi,
      get: vi.fn(async () => PROJECT),
      list: vi.fn(async () => ({
        items: [PROJECT, OTHER_PROJECT],
        pagination: {
          page: 1,
          page_size: 100,
          total: 2,
          total_pages: 1,
        },
      })),
      listMembers: vi.fn(async () => ({
        items: [],
        pagination: {
          page: 1,
          page_size: 200,
          total: 0,
          total_pages: 0,
        },
      })),
      update: vi.fn(),
      delete: vi.fn(),
    },
  }
})

vi.mock("@/components/layout/page-header", () => ({
  PageHeader: ({ items }: { items: Array<{ label: string; dropdown?: React.ReactNode }> }) => {
    pageHeaderItemsMock(items)
    return (
      <div data-testid="page-header">
        {items.map((item) => (
          <div key={item.label}>
            <span>{item.label}</span>
            {item.dropdown}
          </div>
        ))}
      </div>
    )
  },
}))

vi.mock("@/components/layout/not-found-page", () => ({
  NotFoundPage: () => null,
}))

vi.mock("@/components/project/edit-project-dialog", () => ({
  EditProjectDialog: () => null,
}))

vi.mock("@/components/shared/color-badge", () => ({
  ColorBadge: ({ children }: { children: React.ReactNode }) => <span>{children}</span>,
}))

vi.mock("@/components/shared/empty-state", () => ({
  EmptyState: () => null,
}))

vi.mock("@/components/ui/alert-dialog", () => ({
  AlertDialog: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  AlertDialogAction: ({ children }: { children: React.ReactNode }) => <button type="button">{children}</button>,
  AlertDialogCancel: ({ children }: { children: React.ReactNode }) => <button type="button">{children}</button>,
  AlertDialogContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogDescription: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogFooter: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogTitle: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, ...props }: React.ComponentProps<"button">) => <button type="button" {...props}>{children}</button>,
}))

vi.mock("@/components/ui/card", () => ({
  Card: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  CardAction: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  CardContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  CardHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  CardTitle: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenu: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DropdownMenuContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DropdownMenuGroup: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DropdownMenuItem: ({ children, ...props }: React.ComponentProps<"button">) => <button type="button" {...props}>{children}</button>,
  DropdownMenuTrigger: ({ render }: { render?: React.ReactNode }) => <>{render ?? null}</>,
}))

vi.mock("@/components/ui/switch", () => ({
  Switch: () => null,
}))

vi.mock("@/components/ui/tabs", () => ({
  Tabs: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  TabsContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  TabsList: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  TabsTrigger: ({ children }: { children: React.ReactNode }) => <button type="button">{children}</button>,
}))

vi.mock("@/pages/applications/applications-page", () => ({
  ApplicationsPage: () => null,
}))

vi.mock("@/pages/code-repositories/code-repositories-page", () => ({
  CodeRepositoriesPage: () => null,
}))

vi.mock("@/pages/collaborations/collaborations-page", () => ({
  CollaborationsPage: () => null,
}))

vi.mock("@/pages/container-registries/container-registries-page", () => ({
  ContainerRegistriesPage: () => null,
}))

vi.mock("@/pages/dashboard/dashboard-page", () => ({
  UserDashboard: () => null,
}))

vi.mock("@/pages/environments/environments-page", () => ({
  EnvironmentsPage: () => null,
}))

vi.mock("@/pages/members/members-page", () => ({
  MembersPage: () => null,
}))

vi.mock("@/pages/plugins/plugins-page", () => ({
  PluginsPage: () => null,
}))

import { ProjectDetailPage } from "./project-detail-page"

function flushPromises() {
  return new Promise((resolve) => setTimeout(resolve, 0))
}

describe("ProjectDetailPage", () => {
  beforeEach(() => {
    Object.defineProperty(globalThis, "IS_REACT_ACT_ENVIRONMENT", {
      configurable: true,
      value: true,
    })
    pageHeaderItemsMock.mockReset()
    navigateMock.mockReset()
    setActiveContextWithNamesMock.mockReset()
  })

  afterEach(() => {
    document.body.innerHTML = ""
  })

  it("adds a project switcher dropdown to the project breadcrumb", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)

    const root = ReactDOMClient.createRoot(container)
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: {
          retry: false,
        },
      },
    })

    await act(async () => {
      root.render(
        <QueryClientProvider client={queryClient}>
          <ProjectDetailPage />
        </QueryClientProvider>
      )
    })

    await act(async () => {
      await flushPromises()
      await flushPromises()
    })

    const items = pageHeaderItemsMock.mock.calls.at(-1)?.[0] as Array<{
      label: string
      dropdown?: React.ReactNode
    }> | undefined

    expect(items?.[1]?.label).toBe(PROJECT.name)
    expect(items?.[1]?.dropdown).toBeTruthy()

    const projectSwitchButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent?.includes(OTHER_PROJECT.name)
    ) as HTMLButtonElement | undefined

    expect(projectSwitchButton).toBeDefined()

    await act(async () => {
      projectSwitchButton?.click()
    })

    expect(setActiveContextWithNamesMock).toHaveBeenCalledWith(OTHER_PROJECT.id, OTHER_PROJECT.name, null, null)
    expect(navigateMock).toHaveBeenCalledWith(`/projects/${OTHER_PROJECT.id}`)

    await act(async () => {
      root.unmount()
    })
  })

  it("keeps overflowing detail content scrollable with the themed detail scroll area", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)

    const root = ReactDOMClient.createRoot(container)
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: {
          retry: false,
        },
      },
    })

    await act(async () => {
      root.render(
        <QueryClientProvider client={queryClient}>
          <ProjectDetailPage />
        </QueryClientProvider>
      )
    })

    await act(async () => {
      await flushPromises()
      await flushPromises()
    })

    const scrollArea = container.querySelector('[data-detail-page-scroll-area="true"]')
    const content = container.querySelector('[data-slot="detail-page-scroll-content"]')

    expect(scrollArea).not.toBeNull()
    expect(scrollArea?.className).toContain("min-h-0")
    expect(scrollArea?.className).toContain("flex-1")
    expect(content?.className).toContain("gap-6")

    await act(async () => {
      root.unmount()
    })
  })
})
