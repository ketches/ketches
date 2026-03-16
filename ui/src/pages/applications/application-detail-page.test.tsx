import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const { mockNavigate, mockSetSearchParams } = vi.hoisted(() => ({
  mockNavigate: vi.fn(),
  mockSetSearchParams: vi.fn(),
}))

const APP = {
  id: "app-1",
  name: "Demo App",
  slug: "demo-app",
  status: "running",
  description: "Demo application",
  created_at: "2026-03-01T00:00:00Z",
  container_image: "nginx:latest",
  request_cpu: 100,
  limit_cpu: 200,
  request_memory: 256,
  limit_memory: 512,
  app_type: "deployment",
  replicas: 1,
  auto_scaling: null,
  env_id: "env-1",
  available_actions: [],
} as const

const ENV = {
  id: "env-1",
  name: "Demo Env",
  cluster_id: "cluster-1",
  cluster_namespace: "demo",
  project_id: "project-1",
  project_name: "Demo Project",
} as const

const INSTANCE = {
  instance_name: "demo-app-abc123",
  status: "Running",
  ip: "10.0.0.1",
  node_name: "node-1",
  init_container_count: 0,
  container_count: 1,
  restart_count: 0,
  running_duration: "5m",
  containers: ["demo-app-app"],
  init_containers: [],
} as const

vi.mock("@tanstack/react-query", () => ({
  useQuery: ({ queryKey }: { queryKey: unknown[] }) => {
    const key = queryKey[0]

    if (key === "app") {
      return { data: APP, isLoading: false, error: null }
    }
    if (key === "env") {
      return { data: ENV, isLoading: false, error: null }
    }
    if (key === "envs-simple") {
      return { data: [], isLoading: false, error: null }
    }
    if (key === "apps-simple") {
      return { data: [], isLoading: false, error: null }
    }
    if (key === "app-favorite") {
      return { data: { is_favorite: false }, isLoading: false, error: null }
    }
    if (key === "app-instances") {
      return { data: [INSTANCE], isLoading: false, error: null }
    }
    if (key === "app-operation-logs") {
      return {
        data: {
          items: [],
          pagination: { total: 0, page: 1, page_size: 10, total_pages: 0 },
        },
        isLoading: false,
        isFetching: false,
        error: null,
      }
    }
    if (key === "app-metrics-v6") {
      return { data: null, isLoading: false, error: null }
    }

    return { data: undefined, isLoading: false, isFetching: false, error: null }
  },
  useMutation: () => ({
    mutate: vi.fn(),
    mutateAsync: vi.fn(),
    isPending: false,
  }),
  useQueryClient: () => ({
    invalidateQueries: vi.fn(),
  }),
}))

vi.mock("react-router-dom", () => ({
  Link: ({ children, ...props }: React.ComponentProps<"a">) => <a {...props}>{children}</a>,
  useNavigate: () => mockNavigate,
  useParams: () => ({ appId: APP.id }),
  useSearchParams: () => [new URLSearchParams("tab=overview"), mockSetSearchParams],
}))

vi.mock("@/contexts/bottom-panel-context", () => ({
  useBottomPanel: () => ({
    openPanel: vi.fn(),
  }),
}))

vi.mock("@/stores/project", () => ({
  useProjectStore: () => ({
    activeProjectId: ENV.project_id,
    activeEnvId: ENV.id,
    activeProjectName: ENV.project_name,
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

vi.mock("@/components/monitoring/use-prometheus-available", () => ({
  usePrometheusAvailable: () => ({
    available: false,
    isLoading: false,
  }),
}))

vi.mock("@/components/layout/page-header", () => ({
  PageHeader: () => <div>Page Header</div>,
}))

vi.mock("@/components/data-table/data-table", () => ({
  DataTable: () => <div>Data Table</div>,
}))

vi.mock("@/components/shared/stat-card", () => ({
  StatCard: ({ title }: { title: string }) => <div>{title}</div>,
}))

vi.mock("@/components/shared/color-badge", () => ({
  ColorBadge: ({ children }: { children: React.ReactNode }) => <span>{children}</span>,
}))

vi.mock("@/components/shared/empty-state", () => ({
  EmptyState: ({ title }: { title: string }) => <div>{title}</div>,
}))

vi.mock("@/components/applications/app-action-buttons", () => ({
  AppActionButtons: () => null,
}))

vi.mock("@/components/applications/app-topology-view", () => ({
  TopologyView: () => <div>Topology View</div>,
}))

vi.mock("@/components/applications/auto-scaling-config", () => ({
  AutoScalingConfig: () => <div>Auto Scaling</div>,
}))

vi.mock("@/components/applications/command-config", () => ({
  CommandConfig: () => <div>Command Config</div>,
}))

vi.mock("@/components/applications/config-files-table", () => ({
  ConfigFilesTable: () => <div>Config Files</div>,
}))

vi.mock("@/components/applications/edit-app-dialog", () => ({
  EditAppDialog: () => null,
}))

vi.mock("@/components/applications/env-var-table", () => ({
  EnvVarTable: () => <div>Env Vars</div>,
}))

vi.mock("@/components/applications/gateway-table", () => ({
  NetworkConfig: () => <div>Gateways</div>,
}))

vi.mock("@/components/applications/health-config", () => ({
  HealthConfig: () => <div>Health Config</div>,
}))

vi.mock("@/components/applications/image-editor", () => ({
  ImageEditor: () => null,
}))

vi.mock("@/components/applications/resource-config", () => ({
  ResourceConfig: () => <div>Resource Config</div>,
}))

vi.mock("@/components/applications/scheduling-config", () => ({
  SchedulingConfig: () => <div>Scheduling Config</div>,
}))

vi.mock("@/components/applications/volumes-table", () => ({
  VolumesTable: () => <div>Volumes</div>,
}))

vi.mock("@/components/builds/build-list", () => ({
  BuildList: () => <div>Build List</div>,
}))

vi.mock("@/components/code-repositories/unified-build-deploy-dialog", () => ({
  UnifiedBuildDeployDialog: () => null,
}))

vi.mock("@/components/deployments/deployment-history-list", () => ({
  DeploymentHistoryList: () => <div>Deployment History</div>,
}))

vi.mock("@/components/layout/not-found-page", () => ({
  NotFoundPage: () => <div>Not Found</div>,
}))

vi.mock("@/components/monitoring/instance-resource-metrics", () => ({
  InstanceResourceMetrics: () => null,
}))

vi.mock("@/components/monitoring/metrics-time-range-selector", () => ({
  MetricsTimeRangeSelector: () => <div>Time Range Selector</div>,
}))

vi.mock("@/components/plugins/app-plugins", () => ({
  AppPlugins: () => <div>Plugins</div>,
}))

vi.mock("@/components/ui/tabs", () => ({
  Tabs: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  TabsList: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  TabsTrigger: ({ children }: { children: React.ReactNode }) => <button type="button">{children}</button>,
  TabsContent: ({ children }: { children: React.ReactNode }) => <section>{children}</section>,
}))

import ApplicationDetailPage from "./application-detail-page"

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

describe("ApplicationDetailPage", () => {
  beforeEach(() => {
    Object.defineProperty(globalThis, "localStorage", {
      configurable: true,
      value: createMemoryStorage(),
    })
    mockNavigate.mockReset()
    mockSetSearchParams.mockReset()
  })

  afterEach(() => {
    document.body.innerHTML = ""
  })

  it("renders running instances in overview before metrics and removes the instances tab", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)

    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<ApplicationDetailPage />)
    })

    const textContent = container.textContent ?? ""
    const runningInstancesIndex = textContent.indexOf("Running Instances")
    const metricsIndex = textContent.indexOf("Metrics")
    const buttonLabels = Array.from(container.querySelectorAll("button")).map(
      (button) => button.textContent?.trim() ?? ""
    )
    const runningInstancesTitle = Array.from(
      container.querySelectorAll('[data-slot="card-title"]')
    ).find((element) => element.textContent?.includes("Running Instances"))
    const runningInstancesCard = runningInstancesTitle?.closest('[data-slot="card"]')

    expect(buttonLabels).not.toContain("Instances")
    expect(runningInstancesIndex).toBeGreaterThanOrEqual(0)
    expect(metricsIndex).toBeGreaterThanOrEqual(0)
    expect(runningInstancesIndex).toBeLessThan(metricsIndex)
    expect(runningInstancesCard?.querySelector('[data-slot="card-action"]')).not.toBeNull()
    expect(runningInstancesCard?.querySelector('input[placeholder="Filter pods..."]')).toBeNull()

    await act(async () => {
      root.unmount()
    })
  })
})
