import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const { mockSearch, recordedQueries } = vi.hoisted(() => ({
  mockSearch: { value: "tab=overview" },
  recordedQueries: [] as Array<{ key: unknown; enabled: unknown }>,
}))

const ENV = {
  id: "env-1",
  name: "Demo Env",
  slug: "demo-env",
  description: "Demo environment",
  created_at: "2026-03-01T00:00:00Z",
  cluster_id: "cluster-1",
  cluster_name: "Demo Cluster",
  cluster_connection_status: "connected",
  cluster_namespace: "demo",
  project_id: "project-1",
  project_name: "Demo Project",
  is_build_env: false,
  has_prometheus_integration: false,
} as const

vi.mock("@tanstack/react-query", () => ({
  useQuery: (options: { queryKey: unknown[]; enabled?: boolean }) => {
    recordedQueries.push({ key: options.queryKey[0], enabled: options.enabled })
    const key = options.queryKey[0]

    if (key === "envs") {
      return { data: { items: [] }, isLoading: false, error: null }
    }
    if (key === "env") {
      return { data: ENV, isLoading: false, error: null }
    }
    if (key === "apps") {
      return { data: { items: [] }, isLoading: false, error: null }
    }

    return { data: undefined, isLoading: false, error: null }
  },
  useMutation: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
  useQueryClient: () => ({
    invalidateQueries: vi.fn(),
  }),
}))

vi.mock("react-router-dom", () => ({
  useNavigate: () => vi.fn(),
  useParams: () => ({ envId: ENV.id }),
  useSearchParams: () => [new URLSearchParams(mockSearch.value), vi.fn()],
}))

vi.mock("@/stores/project", () => ({
  useProjectStore: () => ({
    activeProjectId: ENV.project_id,
    activeProjectName: ENV.project_name,
    activeEnvId: ENV.id,
    setActiveContextWithNames: vi.fn(),
  }),
}))

vi.mock("@/stores/auth", () => ({
  useAuthStore: (selector: (state: { user: { role: string } }) => unknown) =>
    selector({ user: { role: "viewer" } }),
}))

vi.mock("@/hooks/useProjectRole", () => ({
  useProjectRole: () => "viewer",
}))

vi.mock("@/components/monitoring/use-time-range", () => ({
  useTimeRange: () => ({
    timeRange: "1h",
    setTimeRange: vi.fn(),
    rangeSeconds: 3600,
    step: "60",
  }),
}))

vi.mock("@/components/layout/page-header", () => ({
  PageHeader: () => <div>Page Header</div>,
}))

vi.mock("@/components/applications/application-list", () => ({
  ApplicationList: () => <div>Application List</div>,
}))

vi.mock("@/components/environment/edit-environment-dialog", () => ({
  EditEnvironmentDialog: () => null,
}))

vi.mock("@/components/environment/env-certificates", () => ({
  EnvCertificates: () => <div>Env Certificates</div>,
}))

vi.mock("@/components/environment/env-settings-tab", () => ({
  EnvSettingsTab: () => <div>Env Settings</div>,
}))

vi.mock("@/components/layout/not-found-page", () => ({
  NotFoundPage: () => <div>Not Found</div>,
}))

vi.mock("@/components/monitoring/environment-resource-metrics", () => ({
  EnvironmentResourceMetrics: () => <div>Environment Metrics</div>,
}))

vi.mock("@/components/monitoring/metrics-time-range-selector", () => ({
  MetricsTimeRangeSelector: () => <div>Time Range Selector</div>,
}))

vi.mock("@/components/shared/empty-state", () => ({
  EmptyState: ({ title }: { title: string }) => <div>{title}</div>,
}))

vi.mock("@/components/shared/stat-card", () => ({
  StatCard: ({ title }: { title: string }) => <div>{title}</div>,
}))

vi.mock("@/components/ui/tabs", () => ({
  Tabs: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  TabsList: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  TabsTrigger: ({ children }: { children: React.ReactNode }) => <button type="button">{children}</button>,
  TabsContent: ({ children }: { children: React.ReactNode }) => <section>{children}</section>,
}))

import EnvironmentDetailPage from "./environment-detail-page"

describe("EnvironmentDetailPage", () => {
  beforeEach(() => {
    recordedQueries.length = 0
  })

  afterEach(() => {
    document.body.innerHTML = ""
  })

  it("loads apps for overview and applications tabs only", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    mockSearch.value = "tab=settings"
    await act(async () => {
      root.render(<EnvironmentDetailPage />)
    })

    const settingsAppsQuery = recordedQueries.find((entry) => entry.key === "apps")
    expect(settingsAppsQuery?.enabled).toBe(false)

    recordedQueries.length = 0
    mockSearch.value = "tab=overview"
    await act(async () => {
      root.render(<EnvironmentDetailPage />)
    })

    const overviewAppsQuery = recordedQueries.find((entry) => entry.key === "apps")
    expect(overviewAppsQuery?.enabled).toBe(true)

    await act(async () => {
      root.unmount()
    })
  })

  it("keeps overflowing detail content scrollable with the themed detail scroll area", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    mockSearch.value = "tab=overview"
    await act(async () => {
      root.render(<EnvironmentDetailPage />)
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
