import { act } from "react"
import type { ReactNode } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { readFileSync } from "node:fs"
import { join } from "node:path"

const { mockAuthState, layoutMountTracker } = vi.hoisted(() => ({
  mockAuthState: {
    isAuthenticated: true,
    user: { role: "admin" },
  },
  layoutMountTracker: {
    sidebarMountCount: 0,
  },
}))

vi.mock("@/stores/auth", () => ({
  useAuthStore: (selector: (state: typeof mockAuthState) => unknown) =>
    selector(mockAuthState),
}))

vi.mock("@/components/theme-provider/theme-provider", () => ({
  ThemeProvider: ({ children }: { children: ReactNode }) => <>{children}</>,
}))

vi.mock("@/components/ui/sonner", () => ({
  Toaster: () => null,
}))

vi.mock("@/contexts/breadcrumb-context", () => ({
  BreadcrumbProvider: ({ children }: { children: ReactNode }) => <>{children}</>,
}))

vi.mock("@/contexts/bottom-panel-context", () => ({
  BottomPanelProvider: ({ children }: { children: ReactNode }) => <>{children}</>,
}))

vi.mock("@/components/ui/sidebar", () => ({
  SidebarProvider: ({ children }: { children: ReactNode }) => <>{children}</>,
  SidebarInset: ({ children }: { children: ReactNode }) => <>{children}</>,
}))

vi.mock("@/components/layout/app-header", () => ({
  AppHeader: () => <div data-testid="app-header">header</div>,
}))

vi.mock("@/components/sidebar/app-sidebar", async () => {
  const React = await vi.importActual<typeof import("react")>("react")

  function MockSidebar() {
    React.useEffect(() => {
      layoutMountTracker.sidebarMountCount += 1
    }, [])

    return <div data-testid="app-sidebar">sidebar</div>
  }

  return {
    AppSidebar: MockSidebar,
  }
})

vi.mock("@/components/applications/bottom-panel", () => ({
  BottomPanel: () => <div data-testid="bottom-panel">bottom-panel</div>,
}))

vi.mock("@/pages/auth/login-page", () => ({
  LoginPage: () => <div data-testid="login-page">login-page</div>,
}))

vi.mock("@/pages/auth/signup-page", () => ({
  SignupPage: () => <div data-testid="signup-page">signup-page</div>,
}))

vi.mock("@/pages/dashboard/dashboard-page", () => ({
  DashboardPage: () => <div data-testid="dashboard-page">dashboard-page</div>,
}))

vi.mock("@/pages/projects/projects-page", () => ({
  ProjectsPage: () => <div data-testid="projects-page">projects-page</div>,
}))

vi.mock("@/pages/projects/project-detail-page", () => ({
  ProjectDetailPage: () => <div data-testid="project-detail-page">project-detail-page</div>,
}))

vi.mock("@/pages/builder-sessions/builder-sessions-page", () => ({
  BuilderSessionsPage: () => <div data-testid="builder-sessions-page">builder-sessions-page</div>,
}))

vi.mock("@/pages/builder-sessions/builder-workbench-page", () => ({
  BuilderWorkbenchPage: () => <div data-testid="builder-workbench-page">builder-workbench-page</div>,
}))

vi.mock("@/pages/collaborations/collaborations-page", () => ({
  CollaborationsPage: () => <div data-testid="collaborations-page">collaborations-page</div>,
}))

vi.mock("@/pages/activities/activities-page", () => ({
  ActivitiesPage: () => <div data-testid="activities-page">activities-page</div>,
}))

vi.mock("@/pages/environments/environments-page", () => ({
  EnvironmentsPage: () => <div data-testid="environments-page">environments-page</div>,
}))

vi.mock("@/pages/environments/environment-detail-page", () => ({
  EnvironmentDetailPage: () => <div data-testid="environment-detail-page">environment-detail-page</div>,
}))

vi.mock("@/pages/applications/applications-page", () => ({
  ApplicationsPage: () => <div data-testid="applications-page">applications-page</div>,
}))

vi.mock("@/pages/applications/application-detail-page", () => ({
  ApplicationDetailPage: () => <div data-testid="application-detail-page">application-detail-page</div>,
}))

vi.mock("@/pages/code-repositories/code-repositories-page", () => ({
  CodeRepositoriesPage: () => <div data-testid="code-repositories-page">code-repositories-page</div>,
}))

vi.mock("@/pages/code-repositories/code-repository-detail-page", () => ({
  CodeRepositoryDetailPage: () => <div data-testid="code-repository-detail-page">code-repository-detail-page</div>,
}))

vi.mock("@/pages/container-registries/container-registries-page", () => ({
  ContainerRegistriesPage: () => <div data-testid="container-registries-page">container-registries-page</div>,
}))

vi.mock("@/pages/clusters/clusters-page", () => ({
  ClustersPage: () => <div data-testid="clusters-page">clusters-page</div>,
}))

vi.mock("@/pages/clusters/cluster-detail-page", () => ({
  ClusterDetailPage: () => <div data-testid="cluster-detail-page">cluster-detail-page</div>,
}))

vi.mock("@/pages/clusters/cluster-node-detail-page", () => ({
  ClusterNodeDetailPage: () => <div data-testid="cluster-node-detail-page">cluster-node-detail-page</div>,
}))

vi.mock("@/pages/users/users-page", () => ({
  UsersPage: () => <div data-testid="users-page">users-page</div>,
}))

vi.mock("@/pages/extensions/extensions-page", () => ({
  ExtensionsPage: () => <div data-testid="extensions-page">extensions-page</div>,
}))

vi.mock("@/pages/platform-settings/platform-settings-page", () => ({
  PlatformSettingsPage: () => <div data-testid="platform-settings-page">platform-settings-page</div>,
}))

vi.mock("@/pages/plugins/plugins-page", () => ({
  PluginsPage: () => <div data-testid="plugins-page">plugins-page</div>,
}))

vi.mock("@/pages/members/members-page", () => ({
  MembersPage: () => <div data-testid="members-page">members-page</div>,
}))

vi.mock("@/pages/recycle-bin/recycle-bin-page", () => ({
  RecycleBinPage: () => <div data-testid="recycle-bin-page">recycle-bin-page</div>,
}))

describe("App routing", () => {
  beforeEach(() => {
    layoutMountTracker.sidebarMountCount = 0
    mockAuthState.isAuthenticated = true
    mockAuthState.user = { role: "admin" }
  })

  afterEach(() => {
    document.body.innerHTML = ""
    window.history.pushState({}, "", "/")
  })

  it("keeps the dashboard layout mounted across protected route navigation", async () => {
    const { App } = await import("./App")

    window.history.pushState({}, "", "/projects")
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<App />)
    })

    expect(container.querySelector('[data-testid="projects-page"]')?.textContent).toBe("projects-page")
    expect(layoutMountTracker.sidebarMountCount).toBe(1)

    await act(async () => {
      window.history.pushState({}, "", "/applications")
      window.dispatchEvent(new PopStateEvent("popstate"))
    })

    expect(container.querySelector('[data-testid="applications-page"]')?.textContent).toBe("applications-page")
    expect(layoutMountTracker.sidebarMountCount).toBe(1)

    await act(async () => {
      root.unmount()
    })
  })

  it("lazy-loads protected pages while keeping auth pages eagerly imported", () => {
    const source = readFileSync(join(process.cwd(), "src/App.tsx"), "utf8")

    expect(source).toContain('const DashboardPage = React.lazy(')
    expect(source).toContain('const ProjectsPage = React.lazy(')
    expect(source).toContain('const ApplicationDetailPage = React.lazy(')
    expect(source).toContain('const PlatformSettingsPage = React.lazy(')
    expect(source).toContain('import { LoginPage }')
    expect(source).toContain('import { SignupPage }')
    expect(source).not.toContain('import { DashboardPage }')
    expect(source).not.toContain('import { ProjectsPage }')
    expect(source).not.toContain('import { ApplicationsPage }')
  })
})
