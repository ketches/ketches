import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import type {
  BuilderArtifact,
  BuilderMessage,
  BuilderModelSelection,
  BuilderPreviewSummary,
  BuilderRun,
  BuilderSession,
  BuilderSessionDetail,
} from "@/api/builder-sessions"

const {
  listBuilderSessionsMock,
  getBuilderSessionMock,
  getBuilderModelSelectionMock,
  createBuilderSessionMock,
  postBuilderMessageMock,
  listBuilderExportsMock,
  createBuilderExportMock,
  getBuilderExportPromotionPlanMock,
  promoteBuilderExportMock,
  promoteBuilderExportToBuildMock,
  deployBuilderExportBuildMock,
  downloadBuilderExportBlobMock,
  launchBuilderPreviewMock,
  listBuilderFilesMock,
  readBuilderFileMock,
  downloadTarBlobMock,
  envsListMock,
  pageHeaderItemsMock,
  mockAuthState,
  mockEventSources,
} = vi.hoisted(() => ({
  listBuilderSessionsMock: vi.fn(),
  getBuilderSessionMock: vi.fn(),
  getBuilderModelSelectionMock: vi.fn(),
  createBuilderSessionMock: vi.fn(),
  postBuilderMessageMock: vi.fn(),
  listBuilderExportsMock: vi.fn(),
  createBuilderExportMock: vi.fn(),
  getBuilderExportPromotionPlanMock: vi.fn(),
  promoteBuilderExportMock: vi.fn(),
  promoteBuilderExportToBuildMock: vi.fn(),
  deployBuilderExportBuildMock: vi.fn(),
  downloadBuilderExportBlobMock: vi.fn(),
  launchBuilderPreviewMock: vi.fn(),
  listBuilderFilesMock: vi.fn(),
  readBuilderFileMock: vi.fn(),
  downloadTarBlobMock: vi.fn(),
  envsListMock: vi.fn(),
  pageHeaderItemsMock: vi.fn(),
  mockEventSources: [] as MockEventSource[],
  mockAuthState: {
    user: {
      id: "user-1",
      username: "builder-user",
      email: "builder@example.com",
      role: "user",
      fullname: "Builder User",
    },
  },
}))

vi.mock("@/api/builder-sessions", async () => {
  const actual = await vi.importActual<typeof import("@/api/builder-sessions")>(
    "@/api/builder-sessions"
  )

  return {
    ...actual,
    builderSessionsApi: {
      list: listBuilderSessionsMock,
      get: getBuilderSessionMock,
      getModelSelection: getBuilderModelSelectionMock,
      runLogsStreamUrl: actual.builderSessionsApi.runLogsStreamUrl,
      create: createBuilderSessionMock,
      postMessage: postBuilderMessageMock,
      listExports: listBuilderExportsMock,
      createExport: createBuilderExportMock,
      getExportPromotionPlan: getBuilderExportPromotionPlanMock,
      promoteExportToRepository: promoteBuilderExportMock,
      promoteExportToInitialBuild: promoteBuilderExportToBuildMock,
      deployExportBuild: deployBuilderExportBuildMock,
      downloadExportBlob: downloadBuilderExportBlobMock,
      launchPreview: launchBuilderPreviewMock,
      listFiles: listBuilderFilesMock,
      readFile: readBuilderFileMock,
      downloadTar: vi.fn(),
      downloadTarBlob: downloadTarBlobMock,
    },
  }
})

vi.mock("@/api/envs", async () => {
  const actual = await vi.importActual<typeof import("@/api/envs")>("@/api/envs")

  return {
    ...actual,
    envsApi: {
      list: envsListMock,
    },
  }
})

vi.mock("@/stores/auth", () => ({
  useAuthStore: (selector: (state: typeof mockAuthState) => unknown) => selector(mockAuthState),
}))

vi.mock("@/stores/project", () => ({
  useProjectStore: (
    selector: (state: {
      activeProjectId: string | null
      activeProjectName: string | null
      activeEnvId: string | null
      activeEnvName: string | null
      setActiveContextWithNames: (projectId: string | null, projectName: string | null, envId: string | null, envName: string | null) => void
    }) => unknown
  ) =>
    selector({
      activeProjectId: "project-1",
      activeProjectName: "Demo Project",
      activeEnvId: "env-1",
      activeEnvName: "Builder Env",
      setActiveContextWithNames: vi.fn(),
    }),
}))

vi.mock("@/components/layout/page-header", () => ({
  PageHeader: ({ items }: { items: unknown }) => {
    pageHeaderItemsMock(items)
    return null
  },
}))

vi.mock("@/components/data-table/data-table", () => ({
  DataTable: () => <div data-testid="builder-sessions-table" />,
}))

vi.mock("@/components/shared/empty-state", () => ({
  EmptyState: ({
    title,
    actionText,
    onAction,
  }: {
    title: string
    actionText?: string
    onAction?: () => void
  }) => (
    <div>
      <div>{title}</div>
      {actionText ? (
        <button type="button" onClick={onAction}>
          {actionText}
        </button>
      ) : null}
    </div>
  ),
}))

vi.mock("@/components/ui/avatar", () => ({
  Avatar: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AvatarFallback: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/badge", () => ({
  Badge: ({ children }: { children: React.ReactNode }) => <span>{children}</span>,
}))

vi.mock("@/components/ui/button", () => ({
  Button: ({
    children,
    variant: _variant,
    size: _size,
    asChild: _asChild,
    ...props
  }: React.ComponentProps<"button"> & {
    variant?: string
    size?: string
    asChild?: boolean
  }) => (
    <button type="button" {...props}>
      {children}
    </button>
  ),
}))

vi.mock("@/components/ui/dialog", () => ({
  Dialog: ({ open, children }: { open?: boolean; children: React.ReactNode }) =>
    open ? <div>{children}</div> : null,
  DialogContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogDescription: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogFooter: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogTitle: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/combobox", () => ({
  Combobox: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  ComboboxContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  ComboboxEmpty: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  ComboboxInput: ({ children, ...props }: React.ComponentProps<"input"> & { children?: React.ReactNode }) => (
    <div>
      {children}
      <input {...props} />
    </div>
  ),
  ComboboxItem: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  ComboboxList: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/input", () => ({
  Input: (props: React.ComponentProps<"input">) => <input {...props} />,
}))

vi.mock("@/components/ui/label", () => ({
  Label: ({ children, ...props }: React.ComponentProps<"label">) => <label {...props}>{children}</label>,
}))

vi.mock("@/components/ui/textarea", () => ({
  Textarea: (props: React.ComponentProps<"textarea">) => <textarea {...props} />,
}))

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

import { BuilderSessionsPage } from "./builder-sessions-page"
import { BuilderWorkbenchPage } from "./builder-workbench-page"

const DEFAULT_PAGINATION = {
  page: 1,
  page_size: 20,
  total: 1,
  total_pages: 1,
} as const

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

function flushPromises() {
  return new Promise((resolve) => setTimeout(resolve, 0))
}

async function settle() {
  await act(async () => {
    await flushPromises()
    await flushPromises()
  })
}

class MockEventSource {
  readonly url: string
  readonly withCredentials = false
  private readonly listeners = new Map<string, Array<(event: { data: string }) => void>>()
  onerror: ((this: EventSource, ev: Event) => unknown) | null = null
  onmessage: ((this: EventSource, ev: MessageEvent) => unknown) | null = null
  onopen: ((this: EventSource, ev: Event) => unknown) | null = null

  constructor(url: string) {
    this.url = url
    mockEventSources.push(this)
  }

  addEventListener(type: string, listener: (event: { data: string }) => void) {
    const current = this.listeners.get(type) ?? []
    current.push(listener)
    this.listeners.set(type, current)
  }

  removeEventListener(type: string, listener: (event: { data: string }) => void) {
    const current = this.listeners.get(type) ?? []
    this.listeners.set(
      type,
      current.filter((item) => item !== listener)
    )
  }

  dispatchEvent() {
    return true
  }

  close() { }

  emit(type: string, data = "") {
    for (const listener of this.listeners.get(type) ?? []) {
      listener({ data })
    }
  }
}

function LocationProbe() {
  const location = useLocation()

  return <div data-testid="location">{location.pathname}{location.search}</div>
}

function buildSession(overrides: Partial<BuilderSession> = {}): BuilderSession {
  return {
    id: "session-1",
    project_id: "project-1",
    build_env_id: "env-1",
    title: "Landing page builder",
    summary: "Build a landing page",
    status: "ready",
    created_by: "user-1",
    created_at: "2026-03-20T00:00:00Z",
    updated_at: "2026-03-20T00:10:00Z",
    last_activity_at: new Date(Date.now() - 60 * 60 * 1000).toISOString(),
    expires_at: null,
    latest_run_id: "run-1",
    latest_run_status: "succeeded",
    current_workspace_id: "workspace-1",
    current_workspace_status: "ready",
    current_workspace_root: "/workspace",
    artifact_count: 0,
    ...overrides,
  }
}

function buildRun(overrides: Partial<BuilderRun> = {}): BuilderRun {
  return {
    id: "run-1",
    session_id: "session-1",
    trigger_message_id: "msg-user-1",
    workspace_id: "workspace-1",
    status: "succeeded",
    phase: "finalizing",
    requested_by: "user-1",
    planned_project_kind: "static_frontend_app",
    planned_project_summary: "Detected a static frontend application request.",
    planned_executor_policy_key: "workspace-node-static",
    planned_image_profile_key: "node-static",
    executor_policy_key: "workspace-only",
    execution_image_profile_key: "workspace-default-image",
    execution_image_ref: "node:22-bookworm",
    error_code: "",
    error_class: "",
    instruction_summary: "Create the first version",
    execution_log: "",
    started_at: "2026-03-20T00:02:00Z",
    completed_at: "2026-03-20T00:05:00Z",
    error_message: "",
    created_at: "2026-03-20T00:01:00Z",
    updated_at: "2026-03-20T00:05:00Z",
    ...overrides,
  }
}

function buildArtifact(overrides: Partial<BuilderArtifact> = {}): BuilderArtifact {
  return {
    id: "artifact-1",
    session_id: "session-1",
    workspace_id: "workspace-1",
    run_id: "run-1",
    kind: "file",
    path: "src/main.tsx",
    metadata_json: "{}",
    created_at: "2026-03-20T00:05:00Z",
    updated_at: "2026-03-20T00:05:00Z",
    ...overrides,
  }
}

function buildMessage(overrides: Partial<BuilderMessage> = {}): BuilderMessage {
  return {
    id: "msg-1",
    session_id: "session-1",
    run_id: "run-1",
    role: "assistant",
    content: "I can help build this app.",
    metadata_json: "{}",
    created_by: "user-1",
    created_at: "2026-03-20T00:01:00Z",
    updated_at: "2026-03-20T00:01:00Z",
    ...overrides,
  }
}

function buildDetail({
  session: sessionOverrides,
  runs,
  messages,
  artifacts,
  preview,
}: {
  session?: Partial<BuilderSession>
  runs?: BuilderRun[]
  messages?: BuilderMessage[]
  artifacts?: BuilderArtifact[]
  preview?: BuilderPreviewSummary
} = {}): BuilderSessionDetail {
  const session = buildSession(sessionOverrides)

  return {
    session,
    messages: messages ?? [buildMessage({ session_id: session.id, run_id: session.latest_run_id })],
    runs:
      runs ??
      [
        buildRun({
          id: session.latest_run_id,
          session_id: session.id,
          workspace_id: session.current_workspace_id,
          status: session.latest_run_status,
        }),
      ],
    preview,
    artifacts: artifacts ?? [],
  }
}

async function renderBuilderRoute(initialEntry: string) {
  const container = document.createElement("div")
  document.body.appendChild(container)

  const root = ReactDOMClient.createRoot(container)
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
      mutations: {
        retry: false,
      },
    },
  })

  await act(async () => {
    root.render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={[initialEntry]}>
          <LocationProbe />
          <Routes>
            <Route path="/projects/:projectId/builder-sessions" element={<BuilderSessionsPage />} />
            <Route
              path="/projects/:projectId/builder-sessions/:sessionId"
              element={<BuilderWorkbenchPage />}
            />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>
    )
  })

  return {
    container,
    root,
    queryClient,
  }
}

describe("Builder workspace routes", () => {
  beforeEach(() => {
    vi.useRealTimers()
    Object.defineProperty(globalThis, "IS_REACT_ACT_ENVIRONMENT", {
      configurable: true,
      value: true,
    })
    Object.defineProperty(globalThis, "localStorage", {
      configurable: true,
      value: createMemoryStorage(),
    })
    Object.defineProperty(globalThis, "EventSource", {
      configurable: true,
      value: MockEventSource,
    })

    mockAuthState.user = {
      id: "user-1",
      username: "builder-user",
      email: "builder@example.com",
      role: "user",
      fullname: "Builder User",
    }

    listBuilderSessionsMock.mockReset()
    getBuilderSessionMock.mockReset()
    getBuilderModelSelectionMock.mockReset()
    createBuilderSessionMock.mockReset()
    postBuilderMessageMock.mockReset()
    listBuilderExportsMock.mockReset()
    createBuilderExportMock.mockReset()
    getBuilderExportPromotionPlanMock.mockReset()
    promoteBuilderExportMock.mockReset()
    promoteBuilderExportToBuildMock.mockReset()
    deployBuilderExportBuildMock.mockReset()
    downloadBuilderExportBlobMock.mockReset()
    launchBuilderPreviewMock.mockReset()
    listBuilderFilesMock.mockReset()
    readBuilderFileMock.mockReset()
    downloadTarBlobMock.mockReset()
    envsListMock.mockReset()
    pageHeaderItemsMock.mockReset()
    mockEventSources.length = 0

    listBuilderSessionsMock.mockResolvedValue({
      items: [],
      pagination: DEFAULT_PAGINATION,
    })
    getBuilderSessionMock.mockResolvedValue(buildDetail())
    createBuilderSessionMock.mockResolvedValue(buildDetail())
    postBuilderMessageMock.mockResolvedValue({
      session: buildSession(),
      message: buildMessage({
        id: "msg-queued",
        role: "user",
        content: "Queue this change next",
      }),
      run: buildRun({ id: "run-queued", status: "queued" }),
    })
    listBuilderExportsMock.mockResolvedValue([])
    createBuilderExportMock.mockResolvedValue({
      id: "export-1",
      session_id: "session-1",
      run_id: "run-1",
      workspace_id: "workspace-1",
      snapshot_id: "",
      kind: "session_archive",
      status: "ready",
      file_name: "builder-output-run-1.tar.gz",
      storage_path: "builder-exports/session-1/builder-output-run-1.tar.gz",
      source_root: "dist",
      file_count: 0,
      size_bytes: 0,
      metadata_json: "",
      error_message: "",
      created_by: "user-1",
      created_at: "2026-03-20T00:05:00Z",
      updated_at: "2026-03-20T00:05:00Z",
    })
    getBuilderExportPromotionPlanMock.mockResolvedValue({
      export: {
        id: "export-1",
        session_id: "session-1",
        run_id: "run-1",
        workspace_id: "workspace-1",
        snapshot_id: "",
        kind: "session_archive",
        status: "ready",
        file_name: "builder-output-run-1.tar.gz",
        storage_path: "builder-exports/session-1/builder-output-run-1.tar.gz",
        source_root: "dist",
        file_count: 0,
        size_bytes: 0,
        metadata_json: "",
        error_message: "",
        created_by: "user-1",
        created_at: "2026-03-20T00:05:00Z",
        updated_at: "2026-03-20T00:05:00Z",
      },
      source_kind: "workspace_source",
      planned_project_kind: "go_api_service",
      suggested_repository_name: "Go API Service",
      suggested_repository_slug: "go-api-service",
      suggested_build_env_id: "env-1",
      suggested_build_setting_name: "builder-default",
      suggested_image_name: "go-api-service",
      suggested_dockerfile_path: "Dockerfile",
      suggested_build_context: ".",
      can_trigger_initial_build: true,
      requires_registry_selection: true,
      missing_requirements: [],
    })
    promoteBuilderExportMock.mockResolvedValue({
      export: {
        id: "export-1",
        session_id: "session-1",
        run_id: "run-1",
        workspace_id: "workspace-1",
        snapshot_id: "",
        kind: "session_archive",
        status: "ready",
        file_name: "builder-output-run-1.tar.gz",
        storage_path: "builder-exports/session-1/builder-output-run-1.tar.gz",
        source_root: "dist",
        file_count: 0,
        size_bytes: 0,
        metadata_json: "",
        error_message: "",
        created_by: "user-1",
        created_at: "2026-03-20T00:05:00Z",
        updated_at: "2026-03-20T00:05:00Z",
      },
      repository: {
        id: "repo-1",
        project_id: "project-1",
        name: "Builder Export Repo",
        slug: "builder-export-repo",
        git_repo_url: "https://example.com/demo.git",
        git_username: "",
        git_password: "",
        created_at: "2026-03-20T00:05:00Z",
        updated_at: "2026-03-20T00:05:00Z",
      },
    })
    promoteBuilderExportToBuildMock.mockResolvedValue({
      promotion: {
        export: {
          id: "export-1",
          session_id: "session-1",
          run_id: "run-1",
          workspace_id: "workspace-1",
          snapshot_id: "",
          kind: "session_archive",
          status: "ready",
          file_name: "builder-output-run-1.tar.gz",
          storage_path: "builder-exports/session-1/builder-output-run-1.tar.gz",
          source_root: "dist",
          file_count: 0,
          size_bytes: 0,
          metadata_json: "",
          error_message: "",
          created_by: "user-1",
          created_at: "2026-03-20T00:05:00Z",
          updated_at: "2026-03-20T00:05:00Z",
        },
        repository: {
          id: "repo-1",
          project_id: "project-1",
          name: "Builder Export Repo",
          slug: "builder-export-repo",
          git_repo_url: "https://example.com/demo.git",
          git_username: "",
          git_password: "",
          created_at: "2026-03-20T00:05:00Z",
          updated_at: "2026-03-20T00:05:00Z",
        },
      },
      build_setting: {
        id: "setting-1",
        code_repository_id: "repo-1",
        name: "builder-default",
        git_ref: "main",
        dockerfile_path: "Dockerfile",
        build_context: ".",
        image_name: "builder-export-repo",
        registry_id: "registry-1",
        build_args: "",
        platforms: "linux/amd64",
        registry_cache_enabled: true,
        registry_cache_ref: "",
        created_at: "2026-03-20T00:05:00Z",
        updated_at: "2026-03-20T00:05:00Z",
      },
      build: {
        id: "build-1",
        build_setting_id: "setting-1",
        build_number: 1,
        status: "pending",
        build_env_id: "env-1",
        git_repo_url: "https://example.com/demo.git",
        git_ref: "main",
        git_commit_sha: "",
        git_commit_msg: "",
        image_full_name: "",
        trigger_type: "",
        triggered_by: "user-1",
        job_name: "",
        job_namespace: "",
        started_at: null,
        completed_at: null,
        duration: 0,
        error_message: "",
        log_persist_status: "",
        log_persist_error: "",
        created_at: "2026-03-20T00:05:00Z",
      },
    })
    deployBuilderExportBuildMock.mockResolvedValue({
      App: {
        ID: "app-1",
        CreatedAt: "2026-03-20T00:05:00Z",
        UpdatedAt: "2026-03-20T00:05:00Z",
        DeletedAt: null,
        Slug: "builder-app",
        Name: "Builder App",
        Description: "",
        EnvID: "env-1",
        AppType: "",
        ContainerImage: "",
        ImagePullPolicy: "",
        ContainerCommand: "",
        RegistryUsername: "",
        RegistryPassword: "",
        Replicas: 1,
        RequestCPU: 0,
        RequestMemory: 0,
        LimitCPU: 0,
        LimitMemory: 0,
        DeployStatus: "deployed",
        CodeRepositoryID: "repo-1",
      },
      EnvContext: {
        Env: {
          ID: "env-1",
          CreatedAt: "2026-03-20T00:05:00Z",
          UpdatedAt: "2026-03-20T00:05:00Z",
          DeletedAt: null,
          Slug: "build-env",
          Name: "Build Env",
          Description: "",
          ProjectID: "project-1",
          ClusterID: "cluster-1",
          ClusterNamespace: "builder-ns",
          IsBuildEnv: true,
        },
        Project: {
          ID: "project-1",
          CreatedAt: "2026-03-20T00:05:00Z",
          UpdatedAt: "2026-03-20T00:05:00Z",
          DeletedAt: null,
          Slug: "demo-project",
          Name: "Demo Project",
          Description: "",
          CollaborationEnabled: false,
        },
        Cluster: {
          ID: "cluster-1",
          CreatedAt: "2026-03-20T00:05:00Z",
          UpdatedAt: "2026-03-20T00:05:00Z",
          DeletedAt: null,
          Slug: "cluster-1",
          Name: "Cluster 1",
          Description: "",
          KubeConfig: "",
          ApiServer: "",
          GatewayHost: "",
          Enabled: true,
          ConnectionStatus: "",
          ConnectionStatusReason: "",
          LastCheckedAt: null,
        },
      },
      EnvVars: null,
      Volumes: null,
      Gateways: null,
      Probes: null,
      ConfigFiles: null,
      SchedulingRule: null,
      AutoScaling: null,
      AppPlugins: null,
      Plugins: null,
    })
    downloadBuilderExportBlobMock.mockResolvedValue(undefined)
    listBuilderFilesMock.mockResolvedValue({
      path: "/",
      files: [],
    })
    readBuilderFileMock.mockResolvedValue({
      path: "/src/main.tsx",
      content: "export default function App() {}",
      size: 32,
    })
    downloadTarBlobMock.mockResolvedValue(undefined)
    launchBuilderPreviewMock.mockResolvedValue({ frame_url: "/builder-preview/default" })
    getBuilderModelSelectionMock.mockResolvedValue({
      options: [
        {
          key: "project-claude-sonnet",
          modelLabel: "Claude 4 Sonnet",
          providerLabel: "Anthropic",
          scope: "project",
          providerKey: "anthropic-project",
          modelProfileKey: "claude-sonnet-4",
        },
        {
          key: "user-gpt-4-1",
          modelLabel: "GPT-4.1",
          providerLabel: "OpenAI",
          scope: "user",
          providerKey: "openai-user",
          modelProfileKey: "gpt-4.1",
        },
      ],
      effectiveDefaultSource: "project",
      effectiveDefaultOption: {
        key: "project-claude-sonnet",
        modelLabel: "Claude 4 Sonnet",
        providerLabel: "Anthropic",
        scope: "project",
        providerKey: "anthropic-project",
        modelProfileKey: "claude-sonnet-4",
      },
    } satisfies BuilderModelSelection)
    envsListMock.mockResolvedValue({
      items: [
        {
          id: "env-1",
          name: "Builder Env",
          is_build_env: true,
        },
      ],
      pagination: DEFAULT_PAGINATION,
    })
  })

  afterEach(() => {
    vi.useRealTimers()
    document.body.innerHTML = ""
  })

  it("auto-opens the current user's most recent session when it is still fresh", async () => {
    const recentSession = buildSession({
      id: "session-recent",
      title: "Recent builder session",
      created_by: mockAuthState.user.id,
      last_activity_at: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
    })
    listBuilderSessionsMock.mockResolvedValue({
      items: [recentSession],
      pagination: { ...DEFAULT_PAGINATION, total: 1 },
    })
    getBuilderSessionMock.mockResolvedValue(buildDetail({ session: recentSession }))

    const { container, root } = await renderBuilderRoute("/projects/project-1/builder-sessions")

    await settle()

    expect(container.querySelector('[data-testid="location"]')?.textContent).toBe(
      "/projects/project-1/builder-sessions/session-recent"
    )

    await act(async () => {
      root.unmount()
    })
  })

  it("stays on the chat-first draft view when the current user has no fresh session to resume", async () => {
    const staleSession = buildSession({
      id: "session-stale",
      title: "Old builder session",
      created_by: mockAuthState.user.id,
      last_activity_at: new Date(Date.now() - 9 * 60 * 60 * 1000).toISOString(),
    })
    const otherUsersRecentSession = buildSession({
      id: "session-other-user",
      title: "Another user's recent session",
      created_by: "user-2",
      last_activity_at: new Date(Date.now() - 30 * 60 * 1000).toISOString(),
    })
    listBuilderSessionsMock.mockResolvedValue({
      items: [otherUsersRecentSession, staleSession],
      pagination: { ...DEFAULT_PAGINATION, total: 2 },
    })

    const { container, root } = await renderBuilderRoute("/projects/project-1/builder-sessions")

    await settle()

    expect(container.querySelector('[data-testid="location"]')?.textContent).toBe(
      "/projects/project-1/builder-sessions"
    )
    expect(container.querySelector('[data-testid="builder-session-history"]')).not.toBeNull()
    expect(container.querySelector('[data-testid="builder-composer"]')).not.toBeNull()
    expect(container.textContent).toContain("New conversation")

    await act(async () => {
      root.unmount()
    })
  })

  it("renders the session history rail on a direct session route", async () => {
    const activeSession = buildSession({
      id: "session-active",
      title: "Active builder session",
    })
    const olderSession = buildSession({
      id: "session-older",
      title: "Earlier builder session",
      last_activity_at: new Date(Date.now() - 6 * 60 * 60 * 1000).toISOString(),
    })
    listBuilderSessionsMock.mockResolvedValue({
      items: [activeSession, olderSession],
      pagination: { ...DEFAULT_PAGINATION, total: 2 },
    })
    getBuilderSessionMock.mockResolvedValue(buildDetail({ session: activeSession }))

    const { container, root } = await renderBuilderRoute(
      "/projects/project-1/builder-sessions/session-active"
    )

    await settle()

    const historyRail = container.querySelector('[data-testid="builder-session-history"]')
    const toggle = container.querySelector('[data-testid="builder-session-history-toggle"]') as HTMLButtonElement | null

    expect(historyRail).not.toBeNull()
    expect(historyRail?.textContent).toContain("Earlier builder session")

    await act(async () => {
      toggle?.click()
    })

    expect(container.querySelector('[data-testid="builder-session-history-list"]')).toBeNull()

    await act(async () => {
      root.unmount()
    })
  })

  it("keeps the Builder workspace locked to the available viewport height", async () => {
    const activeSession = buildSession({
      id: "session-active",
      title: "Active builder session",
    })
    getBuilderSessionMock.mockResolvedValue(buildDetail({ session: activeSession }))

    const { container, root } = await renderBuilderRoute(
      "/projects/project-1/builder-sessions/session-active"
    )

    await settle()

    const shell = container.querySelector('[data-testid="builder-workspace-shell"]')
    const body = container.querySelector('[data-testid="builder-workspace-body"]')
    const chatColumn = container.querySelector('[data-testid="builder-workspace-chat-column"]')
    const composerShell = container.querySelector('[data-testid="builder-composer-shell"]')

    expect(shell).not.toBeNull()
    expect(body).not.toBeNull()
    expect(chatColumn).not.toBeNull()
    expect(composerShell).not.toBeNull()
    expect(shell?.className).toContain("h-full")
    expect(body?.className).toContain("h-full")
    expect(body?.className).toContain("overflow-hidden")
    expect(chatColumn?.className).toContain("h-full")
    expect(chatColumn?.className).toContain("overflow-hidden")
    expect(composerShell?.className).toContain("shrink-0")

    await act(async () => {
      root.unmount()
    })
  })

  it("lets users force a fresh draft even when a resumable session exists", async () => {
    const activeSession = buildSession({
      id: "session-active",
      title: "Active builder session",
      created_by: mockAuthState.user.id,
      last_activity_at: new Date(Date.now() - 30 * 60 * 1000).toISOString(),
    })
    const olderSession = buildSession({
      id: "session-older",
      title: "Earlier builder session",
      created_by: mockAuthState.user.id,
      last_activity_at: new Date(Date.now() - 4 * 60 * 60 * 1000).toISOString(),
    })
    listBuilderSessionsMock.mockResolvedValue({
      items: [activeSession, olderSession],
      pagination: { ...DEFAULT_PAGINATION, total: 2 },
    })
    getBuilderSessionMock.mockResolvedValue(buildDetail({ session: activeSession }))

    const { container, root } = await renderBuilderRoute(
      "/projects/project-1/builder-sessions/session-active"
    )

    await settle()

    const newConversationButton = Array.from(container.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("New conversation")
    ) as HTMLButtonElement | undefined

    if (!newConversationButton) {
      const toggle = container.querySelector('[data-testid="builder-session-history-toggle"]') as HTMLButtonElement | null
      await act(async () => {
        toggle?.click()
      })
    }

    const expandedNewConversationButton = Array.from(container.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("New conversation")
    ) as HTMLButtonElement | undefined

    expect(expandedNewConversationButton).toBeDefined()

    await act(async () => {
      expandedNewConversationButton?.click()
      await flushPromises()
      await flushPromises()
    })

    expect(container.querySelector('[data-testid="location"]')?.textContent).toBe(
      "/projects/project-1/builder-sessions?draft=1"
    )
    expect(container.querySelector('[data-testid="builder-composer"]')).not.toBeNull()

    await act(async () => {
      root.unmount()
    })
  })

  it("keeps the composer enabled so users can queue another message while a run is active", async () => {
    const activeSession = buildSession({
      id: "session-active",
      latest_run_id: "run-active",
      latest_run_status: "executing",
    })
    getBuilderSessionMock.mockResolvedValue(
      buildDetail({
        session: activeSession,
        runs: [
          buildRun({
            id: "run-active",
            session_id: activeSession.id,
            workspace_id: activeSession.current_workspace_id,
            status: "executing",
            completed_at: null,
          }),
        ],
      })
    )

    const { container, root } = await renderBuilderRoute(
      "/projects/project-1/builder-sessions/session-active"
    )

    await settle()

    const composer = container.querySelector(
      '[data-testid="builder-composer"]'
    ) as HTMLTextAreaElement | null
    const sendButton = container.querySelector(
      '[data-testid="builder-send-message"]'
    ) as HTMLButtonElement | null

    expect(composer).not.toBeNull()
    expect(composer?.disabled).toBe(false)
    expect(sendButton).not.toBeNull()

    await act(async () => {
      if (composer) {
        composer.value = "Add a settings page next"
        composer.dispatchEvent(new Event("input", { bubbles: true }))
      }
      sendButton?.click()
      await flushPromises()
    })

    expect(postBuilderMessageMock).toHaveBeenCalledWith("project-1", "session-active", {
      content: "Add a settings page next",
      selected_model_key: "project-claude-sonnet",
      provider_key: "anthropic-project",
      model_profile_key: "claude-sonnet-4",
    })

    await act(async () => {
      root.unmount()
    })
  })

  it("shows queued follow-up messages above the composer while a run is still active", async () => {
    const activeSession = buildSession({
      id: "session-active",
      latest_run_id: "run-active",
      latest_run_status: "executing",
    })
    getBuilderSessionMock.mockResolvedValue(
      buildDetail({
        session: activeSession,
        runs: [
          buildRun({
            id: "run-active",
            session_id: activeSession.id,
            workspace_id: activeSession.current_workspace_id,
            status: "executing",
            completed_at: null,
          }),
        ],
      })
    )

    const { container, root } = await renderBuilderRoute(
      "/projects/project-1/builder-sessions/session-active"
    )

    await settle()

    const composer = container.querySelector(
      '[data-testid="builder-composer"]'
    ) as HTMLTextAreaElement | null
    const sendButton = container.querySelector(
      '[data-testid="builder-send-message"]'
    ) as HTMLButtonElement | null

    await act(async () => {
      if (composer) {
        composer.value = "Queue this change next"
        composer.dispatchEvent(new Event("input", { bubbles: true }))
      }
      sendButton?.click()
      await flushPromises()
      await flushPromises()
    })

    expect(container.querySelector('[data-testid="builder-queued-messages"]')).not.toBeNull()
    expect(container.textContent).toContain("Queued next")
    expect(container.textContent).toContain("Queue this change next")

    await act(async () => {
      root.unmount()
    })
  })

  it("streams run log events into a pending assistant bubble while a run is active", async () => {
    const activeSession = buildSession({
      id: "session-active",
      latest_run_id: "run-active",
      latest_run_status: "executing",
    })
    getBuilderSessionMock.mockResolvedValue(
      buildDetail({
        session: activeSession,
        runs: [
          buildRun({
            id: "run-active",
            session_id: activeSession.id,
            workspace_id: activeSession.current_workspace_id,
            status: "executing",
            completed_at: null,
          }),
        ],
      })
    )

    const { container, root } = await renderBuilderRoute(
      "/projects/project-1/builder-sessions/session-active"
    )

		await settle()

		expect(mockEventSources).toHaveLength(1)
		expect(mockEventSources[0]?.url).toContain("/api/v1/projects/project-1/builder-sessions/session-active/runs/run-active/logs")
		expect(mockEventSources[0]?.url).not.toContain("token=")
		expect(container.textContent).toContain("Builder is working")

    await act(async () => {
      mockEventSources[0]?.emit("log", "[agent] generating files...\n")
      await flushPromises()
    })

    expect(container.textContent).toContain("[agent] generating files...")

    await act(async () => {
      root.unmount()
    })
  })

  it("hides reply-only completion summaries from the conversation transcript", async () => {
    const activeSession = buildSession({
      id: "session-active",
      latest_run_id: "run-active",
      latest_run_status: "succeeded",
    })
    getBuilderSessionMock.mockResolvedValue(
      buildDetail({
        session: activeSession,
        runs: [
          buildRun({
            id: "run-active",
            session_id: activeSession.id,
            workspace_id: activeSession.current_workspace_id,
            status: "succeeded",
          }),
        ],
        messages: [
          buildMessage({
            id: "msg-user-1",
            role: "user",
            run_id: "run-active",
            session_id: activeSession.id,
            content: "hi",
          }),
          buildMessage({
            id: "msg-assistant-1",
            role: "assistant",
            run_id: "run-active",
            session_id: activeSession.id,
            content: "Hi! What would you like to build or change?",
          }),
          buildMessage({
            id: "msg-system-1",
            role: "system",
            run_id: "run-active",
            session_id: activeSession.id,
            content: "run completed: replied without workspace changes",
          }),
        ],
      })
    )

    const { container, root } = await renderBuilderRoute(
      "/projects/project-1/builder-sessions/session-active"
    )

    await settle()

    expect(container.textContent).toContain("Hi! What would you like to build or change?")
    expect(container.textContent).not.toContain("run completed: replied without workspace changes")

    await act(async () => {
      root.unmount()
    })
  })

  it("loads model selection for an existing session route", async () => {
    const activeSession = buildSession({
      id: "session-active",
      latest_run_id: "run-active",
      latest_run_status: "succeeded",
    })
    getBuilderSessionMock.mockResolvedValue(
      buildDetail({
        session: activeSession,
        runs: [
          buildRun({
            id: "run-active",
            session_id: activeSession.id,
            workspace_id: activeSession.current_workspace_id,
            status: "succeeded",
          }),
        ],
      })
    )

    const { root } = await renderBuilderRoute(
      "/projects/project-1/builder-sessions/session-active"
    )

    await settle()

    expect(getBuilderModelSelectionMock).toHaveBeenCalledWith("project-1")

    await act(async () => {
      root.unmount()
    })
  })

  it("includes the selected model when queuing a message in an existing session", async () => {
    const activeSession = buildSession({
      id: "session-active",
      latest_run_id: "run-active",
      latest_run_status: "succeeded",
    })
    getBuilderSessionMock.mockResolvedValue(
      buildDetail({
        session: activeSession,
        runs: [
          buildRun({
            id: "run-active",
            session_id: activeSession.id,
            workspace_id: activeSession.current_workspace_id,
            status: "succeeded",
          }),
        ],
      })
    )

    const { container, root } = await renderBuilderRoute(
      "/projects/project-1/builder-sessions/session-active"
    )

    await settle()

    const composer = container.querySelector(
      '[data-testid="builder-composer"]'
    ) as HTMLTextAreaElement | null
    const sendButton = container.querySelector(
      '[data-testid="builder-send-message"]'
    ) as HTMLButtonElement | null

    await act(async () => {
      if (composer) {
        composer.value = "Add a settings page next"
        composer.dispatchEvent(new Event("input", { bubbles: true }))
      }
      sendButton?.click()
      await flushPromises()
    })

    expect(postBuilderMessageMock).toHaveBeenCalledWith("project-1", "session-active", {
      content: "Add a settings page next",
      selected_model_key: "project-claude-sonnet",
      provider_key: "anthropic-project",
      model_profile_key: "claude-sonnet-4",
    })

    await act(async () => {
      root.unmount()
    })
  })

  it("shows resolved execution metadata for the latest run", async () => {
    const activeSession = buildSession({
      id: "session-active",
      latest_run_id: "run-active",
      latest_run_status: "succeeded",
    })
    getBuilderSessionMock.mockResolvedValue(
      buildDetail({
        session: activeSession,
        runs: [
          buildRun({
            id: "run-active",
            session_id: activeSession.id,
            workspace_id: activeSession.current_workspace_id,
            status: "succeeded",
            planned_project_kind: "static_frontend_app",
            planned_project_summary: "Detected a static frontend application request.",
            planned_executor_policy_key: "workspace-node-static",
            planned_image_profile_key: "node-static",
            executor_policy_key: "workspace-node-static",
            execution_image_profile_key: "node-static",
            execution_image_ref: "ghcr.io/ketches/builder-node-static:2026-03-29",
            phase: "testing",
            error_code: "",
            error_class: "",
          }),
        ],
      })
    )

    const { container, root } = await renderBuilderRoute(
      "/projects/project-1/builder-sessions/session-active"
    )

    await settle()

    expect(container.textContent).toContain("Executor: workspace-node-static")
    expect(container.textContent).toContain("Image: ghcr.io/ketches/builder-node-static:2026-03-29")
    expect(container.textContent).toContain("Project kind: static_frontend_app")
    expect(container.textContent).toContain("Phase: testing")

    await act(async () => {
      root.unmount()
    })
  })

  it("lists session exports and downloads a selected export", async () => {
    const activeSession = buildSession({
      id: "session-active",
      latest_run_id: "run-active",
      latest_run_status: "succeeded",
    })
    getBuilderSessionMock.mockResolvedValue(
      buildDetail({
        session: activeSession,
        runs: [
          buildRun({
            id: "run-active",
            session_id: activeSession.id,
            workspace_id: activeSession.current_workspace_id,
            status: "succeeded",
          }),
        ],
      })
    )
    listBuilderExportsMock.mockResolvedValue([
      {
        id: "export-1",
        session_id: activeSession.id,
        run_id: "run-active",
        workspace_id: activeSession.current_workspace_id,
        snapshot_id: "",
        kind: "session_archive",
        status: "ready",
        file_name: "builder-output-run-active.tar.gz",
        storage_path: "builder-exports/session-active/builder-output-run-active.tar.gz",
        source_root: "dist",
        file_count: 0,
        size_bytes: 0,
        metadata_json: "",
        error_message: "",
        created_by: "user-1",
        created_at: "2026-03-20T00:05:00Z",
        updated_at: "2026-03-20T00:05:00Z",
      },
    ])

    const { container, root } = await renderBuilderRoute(
      "/projects/project-1/builder-sessions/session-active"
    )

    await settle()

    expect(container.textContent).toContain("Exports")
    expect(container.textContent).toContain("builder-output-run-active.tar.gz")

    const downloadButton = Array.from(container.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("Download export")
    ) as HTMLButtonElement | undefined

    await act(async () => {
      downloadButton?.click()
      await flushPromises()
    })

    expect(downloadBuilderExportBlobMock).toHaveBeenCalledWith("project-1", "session-active", "export-1")

    await act(async () => {
      root.unmount()
    })
  })

  it("creates a session export from the Builder session page", async () => {
    const activeSession = buildSession({
      id: "session-active",
      latest_run_id: "run-active",
      latest_run_status: "succeeded",
    })
    getBuilderSessionMock.mockResolvedValue(
      buildDetail({
        session: activeSession,
        runs: [
          buildRun({
            id: "run-active",
            session_id: activeSession.id,
            workspace_id: activeSession.current_workspace_id,
            status: "succeeded",
          }),
        ],
      })
    )

    const { container, root } = await renderBuilderRoute(
      "/projects/project-1/builder-sessions/session-active"
    )

    await settle()

    const createExportButton = Array.from(container.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("Create export")
    ) as HTMLButtonElement | undefined

    await act(async () => {
      createExportButton?.click()
      await flushPromises()
    })

    expect(createBuilderExportMock).toHaveBeenCalledWith("project-1", "session-active")

    await act(async () => {
      root.unmount()
    })
  })

  it("promotes an export to a code repository from the Builder session page", async () => {
    const activeSession = buildSession({
      id: "session-active",
      latest_run_id: "run-active",
      latest_run_status: "succeeded",
    })
    getBuilderSessionMock.mockResolvedValue(
      buildDetail({
        session: activeSession,
        runs: [
          buildRun({
            id: "run-active",
            session_id: activeSession.id,
            workspace_id: activeSession.current_workspace_id,
            status: "succeeded",
          }),
        ],
      })
    )
    listBuilderExportsMock.mockResolvedValue([
      {
        id: "export-1",
        session_id: activeSession.id,
        run_id: "run-active",
        workspace_id: activeSession.current_workspace_id,
        snapshot_id: "",
        kind: "session_archive",
        status: "ready",
        file_name: "builder-output-run-active.tar.gz",
        storage_path: "builder-exports/session-active/builder-output-run-active.tar.gz",
        source_root: "dist",
        file_count: 0,
        size_bytes: 0,
        metadata_json: "",
        error_message: "",
        created_by: "user-1",
        created_at: "2026-03-20T00:05:00Z",
        updated_at: "2026-03-20T00:05:00Z",
      },
    ])

    const { container, root } = await renderBuilderRoute(
      "/projects/project-1/builder-sessions/session-active"
    )

    await settle()

    const promoteButton = Array.from(container.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("Promote to repository")
    ) as HTMLButtonElement | undefined

    await act(async () => {
      promoteButton?.click()
      await flushPromises()
    })

    const repoUrlInput = container.querySelector('input[name="builder_export_git_repo_url"]') as HTMLInputElement | null
    expect(repoUrlInput).not.toBeNull()

    await act(async () => {
      if (repoUrlInput) {
        repoUrlInput.value = "https://example.com/demo.git"
        repoUrlInput.dispatchEvent(new Event("input", { bubbles: true }))
      }
      await flushPromises()
    })

    const submitButton = Array.from(container.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("Promote export")
    ) as HTMLButtonElement | undefined
    expect(submitButton).toBeDefined()

    await act(async () => {
      submitButton?.click()
      await flushPromises()
    })

    expect(promoteBuilderExportMock).toHaveBeenCalledWith("project-1", "session-active", "export-1", {
      name: "",
      slug: "",
      git_repo_url: "https://example.com/demo.git",
      git_username: "",
      git_password: "",
    })

    await act(async () => {
      root.unmount()
    })
  })

  it("loads a promotion plan and promotes an export to an initial build", async () => {
    const activeSession = buildSession({
      id: "session-active",
      latest_run_id: "run-active",
      latest_run_status: "succeeded",
    })
    getBuilderSessionMock.mockResolvedValue(
      buildDetail({
        session: activeSession,
        runs: [
          buildRun({
            id: "run-active",
            session_id: activeSession.id,
            workspace_id: activeSession.current_workspace_id,
            status: "succeeded",
          }),
        ],
      })
    )
    listBuilderExportsMock.mockResolvedValue([
      {
        id: "export-1",
        session_id: activeSession.id,
        run_id: "run-active",
        workspace_id: activeSession.current_workspace_id,
        snapshot_id: "",
        kind: "session_archive",
        status: "ready",
        file_name: "builder-output-run-active.tar.gz",
        storage_path: "builder-exports/session-active/builder-output-run-active.tar.gz",
        source_root: "dist",
        file_count: 0,
        size_bytes: 0,
        metadata_json: "",
        error_message: "",
        created_by: "user-1",
        created_at: "2026-03-20T00:05:00Z",
        updated_at: "2026-03-20T00:05:00Z",
      },
    ])

    const { container, root } = await renderBuilderRoute(
      "/projects/project-1/builder-sessions/session-active"
    )

    await settle()

    const promoteButton = Array.from(container.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("Promote to repository")
    ) as HTMLButtonElement | undefined

    await act(async () => {
      promoteButton?.click()
      await flushPromises()
      await flushPromises()
    })

    expect(getBuilderExportPromotionPlanMock).toHaveBeenCalledWith("project-1", "session-active", "export-1")
    expect(container.textContent).toContain("Plan: go_api_service")
    expect(container.textContent).toContain("Suggested env: env-1")

    const repoUrlInput = container.querySelector('input[name="builder_export_git_repo_url"]') as HTMLInputElement | null
    const registryIdInput = container.querySelector('input[name="builder_export_registry_id"]') as HTMLInputElement | null
    expect(repoUrlInput).not.toBeNull()
    expect(registryIdInput).not.toBeNull()

    await act(async () => {
      if (repoUrlInput) {
        repoUrlInput.value = "https://example.com/demo.git"
        repoUrlInput.dispatchEvent(new Event("input", { bubbles: true }))
      }
      if (registryIdInput) {
        registryIdInput.value = "registry-1"
        registryIdInput.dispatchEvent(new Event("input", { bubbles: true }))
      }
      await flushPromises()
    })

    const promoteBuildButton = Array.from(container.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("Promote to initial build")
    ) as HTMLButtonElement | undefined
    expect(promoteBuildButton).toBeDefined()

    await act(async () => {
      promoteBuildButton?.click()
      await flushPromises()
    })

    expect(promoteBuilderExportToBuildMock).toHaveBeenCalledWith("project-1", "session-active", "export-1", {
      name: "",
      slug: "",
      git_repo_url: "https://example.com/demo.git",
      git_username: "",
      git_password: "",
      build_env_id: "env-1",
      registry_id: "registry-1",
      build_setting_name: "builder-default",
      image_name: "go-api-service",
      dockerfile_path: "Dockerfile",
      build_context: ".",
      git_ref: "main",
    })

    await act(async () => {
      root.unmount()
    })
  })

  it("deploys a promoted Builder build from the session page", async () => {
    const activeSession = buildSession({
      id: "session-active",
      latest_run_id: "run-active",
      latest_run_status: "succeeded",
    })
    getBuilderSessionMock.mockResolvedValue(
      buildDetail({
        session: activeSession,
        runs: [
          buildRun({
            id: "run-active",
            session_id: activeSession.id,
            workspace_id: activeSession.current_workspace_id,
            status: "succeeded",
          }),
        ],
      })
    )
    listBuilderExportsMock.mockResolvedValue([
      {
        id: "export-1",
        session_id: activeSession.id,
        run_id: "run-active",
        workspace_id: activeSession.current_workspace_id,
        snapshot_id: "",
        kind: "session_archive",
        status: "ready",
        file_name: "builder-output-run-active.tar.gz",
        storage_path: "builder-exports/session-active/builder-output-run-active.tar.gz",
        source_root: "dist",
        file_count: 0,
        size_bytes: 0,
        metadata_json: "",
        error_message: "",
        created_by: "user-1",
        created_at: "2026-03-20T00:05:00Z",
        updated_at: "2026-03-20T00:05:00Z",
      },
    ])

    const { container, root } = await renderBuilderRoute(
      "/projects/project-1/builder-sessions/session-active"
    )

    await settle()

    const promoteButton = Array.from(container.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("Promote to repository")
    ) as HTMLButtonElement | undefined

    await act(async () => {
      promoteButton?.click()
      await flushPromises()
      await flushPromises()
    })

    const repoUrlInput = container.querySelector('input[name="builder_export_git_repo_url"]') as HTMLInputElement | null
    const registryIdInput = container.querySelector('input[name="builder_export_registry_id"]') as HTMLInputElement | null

    await act(async () => {
      if (repoUrlInput) {
        repoUrlInput.value = "https://example.com/demo.git"
        repoUrlInput.dispatchEvent(new Event("input", { bubbles: true }))
      }
      if (registryIdInput) {
        registryIdInput.value = "registry-1"
        registryIdInput.dispatchEvent(new Event("input", { bubbles: true }))
      }
      await flushPromises()
    })

    const promoteBuildButton = Array.from(container.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("Promote to initial build")
    ) as HTMLButtonElement | undefined

    await act(async () => {
      promoteBuildButton?.click()
      await flushPromises()
    })

    const deployBuildButton = Array.from(container.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("Deploy build")
    ) as HTMLButtonElement | undefined
    expect(deployBuildButton).toBeDefined()

    await act(async () => {
      deployBuildButton?.click()
      await flushPromises()
    })

    expect(deployBuilderExportBuildMock).toHaveBeenCalledWith("project-1", "session-active", "export-1", {
      repository_id: "repo-1",
      build_id: "build-1",
      target_env_id: "env-1",
      app_id: "",
      name: "Builder App",
      slug: "builder-app",
    })

    await act(async () => {
      root.unmount()
    })
  })

  it("creates a new Builder session instead of posting to a failed session", async () => {
    const failedSession = buildSession({
      id: "session-failed",
      build_env_id: "env-2",
      status: "failed",
      latest_run_id: "run-failed",
      latest_run_status: "failed",
    })
    createBuilderSessionMock.mockResolvedValue(
      buildDetail({
        session: buildSession({
          id: "session-replacement",
          build_env_id: "env-2",
        }),
      })
    )
    getBuilderSessionMock.mockResolvedValue(
      buildDetail({
        session: failedSession,
        runs: [
          buildRun({
            id: "run-failed",
            session_id: failedSession.id,
            workspace_id: failedSession.current_workspace_id,
            status: "failed",
          }),
        ],
      })
    )

    const { container, root } = await renderBuilderRoute(
      "/projects/project-1/builder-sessions/session-failed"
    )

    await settle()

    const composer = container.querySelector(
      '[data-testid="builder-composer"]'
    ) as HTMLTextAreaElement | null
    const sendButton = container.querySelector(
      '[data-testid="builder-send-message"]'
    ) as HTMLButtonElement | null

    await act(async () => {
      if (composer) {
        composer.value = "Retry this app build in a fresh session"
        composer.dispatchEvent(new Event("input", { bubbles: true }))
      }
      sendButton?.click()
      await flushPromises()
      await flushPromises()
    })

    expect(postBuilderMessageMock).not.toHaveBeenCalled()
    expect(createBuilderSessionMock).toHaveBeenCalledWith("project-1", {
      build_env_id: "env-2",
      prompt: "Retry this app build in a fresh session",
      selected_model_key: "project-claude-sonnet",
      provider_key: "anthropic-project",
      model_profile_key: "claude-sonnet-4",
    })
    expect(container.querySelector('[data-testid="location"]')?.textContent).toBe(
      "/projects/project-1/builder-sessions/session-replacement"
    )

    await act(async () => {
      root.unmount()
    })
  })

  it("falls back to creating a new Builder session when postMessage returns not appendable", async () => {
    const closedSession = buildSession({
      id: "session-closed",
      build_env_id: "env-2",
      status: "running",
      latest_run_id: "run-late-close",
      latest_run_status: "executing",
    })
    postBuilderMessageMock.mockRejectedValue({
      isAxiosError: true,
      response: {
        status: 409,
        data: {
          error: "builder session session-closed is not appendable in status failed",
        },
      },
    })
    createBuilderSessionMock.mockResolvedValue(
      buildDetail({
        session: buildSession({
          id: "session-recreated",
          build_env_id: "env-2",
        }),
      })
    )
    getBuilderSessionMock.mockResolvedValue(
      buildDetail({
        session: closedSession,
        runs: [
          buildRun({
            id: "run-late-close",
            session_id: closedSession.id,
            workspace_id: closedSession.current_workspace_id,
            status: "executing",
            completed_at: null,
          }),
        ],
      })
    )

    const { container, root } = await renderBuilderRoute(
      "/projects/project-1/builder-sessions/session-closed"
    )

    await settle()

    const composer = container.querySelector(
      '[data-testid="builder-composer"]'
    ) as HTMLTextAreaElement | null
    const sendButton = container.querySelector(
      '[data-testid="builder-send-message"]'
    ) as HTMLButtonElement | null

    await act(async () => {
      if (composer) {
        composer.value = "Continue in a replacement session"
        composer.dispatchEvent(new Event("input", { bubbles: true }))
      }
      sendButton?.click()
      await flushPromises()
      await flushPromises()
    })

    expect(postBuilderMessageMock).toHaveBeenCalledWith("project-1", "session-closed", {
      content: "Continue in a replacement session",
      selected_model_key: "project-claude-sonnet",
      provider_key: "anthropic-project",
      model_profile_key: "claude-sonnet-4",
    })
    expect(createBuilderSessionMock).toHaveBeenCalledWith("project-1", {
      build_env_id: "env-2",
      prompt: "Continue in a replacement session",
      selected_model_key: "project-claude-sonnet",
      provider_key: "anthropic-project",
      model_profile_key: "claude-sonnet-4",
    })
    expect(container.querySelector('[data-testid="location"]')?.textContent).toBe(
      "/projects/project-1/builder-sessions/session-recreated"
    )

    await act(async () => {
      root.unmount()
    })
  })

  it("stops polling once an active run refresh reports completion", async () => {
    vi.useFakeTimers()

    const activeSession = buildSession({
      id: "session-polling",
      latest_run_id: "run-polling",
      latest_run_status: "executing",
    })
    const completedSession = buildSession({
      ...activeSession,
      latest_run_status: "succeeded",
    })

    getBuilderSessionMock
      .mockResolvedValueOnce(
        buildDetail({
          session: activeSession,
          runs: [
            buildRun({
              id: "run-polling",
              session_id: activeSession.id,
              workspace_id: activeSession.current_workspace_id,
              status: "executing",
              completed_at: null,
            }),
          ],
        })
      )
      .mockResolvedValueOnce(
        buildDetail({
          session: completedSession,
          runs: [
            buildRun({
              id: "run-polling",
              session_id: completedSession.id,
              workspace_id: completedSession.current_workspace_id,
              status: "succeeded",
            }),
          ],
        })
      )
      .mockResolvedValue(
        buildDetail({
          session: completedSession,
          runs: [
            buildRun({
              id: "run-polling",
              session_id: completedSession.id,
              workspace_id: completedSession.current_workspace_id,
              status: "succeeded",
            }),
          ],
        })
      )

    const { root } = await renderBuilderRoute(
      "/projects/project-1/builder-sessions/session-polling"
    )

    await act(async () => {
      await Promise.resolve()
      await Promise.resolve()
      await Promise.resolve()
    })

    const initialCallCount = getBuilderSessionMock.mock.calls.length

    expect(initialCallCount).toBe(1)

    await act(async () => {
      await vi.advanceTimersByTimeAsync(3000)
      await Promise.resolve()
      await Promise.resolve()
    })

    const completedRefreshCallCount = getBuilderSessionMock.mock.calls.length

    expect(completedRefreshCallCount).toBe(2)

    await act(async () => {
      await vi.advanceTimersByTimeAsync(3000)
      await Promise.resolve()
      await Promise.resolve()
    })

    expect(getBuilderSessionMock).toHaveBeenCalledTimes(completedRefreshCallCount)

    await act(async () => {
      root.unmount()
    })
  })

  it("does not reveal or query the files rail before any project files exist", async () => {
    const sessionWithoutFiles = buildSession({
      id: "session-without-files",
      artifact_count: 0,
    })
    getBuilderSessionMock.mockResolvedValue(
      buildDetail({
        session: sessionWithoutFiles,
        artifacts: [],
      })
    )

    const { container, root } = await renderBuilderRoute(
      "/projects/project-1/builder-sessions/session-without-files"
    )

    await settle()

    expect(container.querySelector('[data-testid="builder-files-rail"]')).toBeNull()
    expect(listBuilderFilesMock).not.toHaveBeenCalled()
    expect(container.textContent).not.toContain("No files yet")

    await act(async () => {
      root.unmount()
    })
  })

  it("shows only a collapsed files toggle by default after files exist", async () => {
    const sessionWithFiles = buildSession({
      id: "session-with-files",
      artifact_count: 1,
    })
    getBuilderSessionMock.mockResolvedValue(
      buildDetail({
        session: sessionWithFiles,
        artifacts: [
          buildArtifact({
            session_id: sessionWithFiles.id,
            workspace_id: sessionWithFiles.current_workspace_id,
            run_id: sessionWithFiles.latest_run_id,
            path: "src/main.tsx",
          }),
        ],
      })
    )
    listBuilderFilesMock.mockResolvedValue({
      path: "/",
      files: [
        {
          name: "src",
          type: "dir",
          size: 0,
          modTime: "2026-03-20T00:05:00Z",
        },
      ],
    })

    const { container, root } = await renderBuilderRoute(
      "/projects/project-1/builder-sessions/session-with-files"
    )

    await settle()

    expect(container.querySelector('[data-testid="builder-files-rail-toggle"]')).not.toBeNull()
    expect(container.querySelector('[data-testid="builder-files-list"]')).toBeNull()
    expect(container.textContent).not.toContain("src")

    await act(async () => {
      root.unmount()
    })
  })

  it("does not render a duplicate in-page header on the chat view", async () => {
    const activeSession = buildSession({
      id: "session-active",
      title: "Active builder session",
    })
    getBuilderSessionMock.mockResolvedValue(
      buildDetail({
        session: activeSession,
        messages: [
          buildMessage({
            session_id: activeSession.id,
            run_id: activeSession.latest_run_id,
            content: "Continue building the landing page.",
          }),
        ],
      })
    )

    const { container, root } = await renderBuilderRoute(
      "/projects/project-1/builder-sessions/session-active"
    )

    await settle()

    expect(container.textContent).toContain("Continue building the landing page.")
    expect(container.textContent).not.toContain("Active builder session")
    expect(container.textContent).not.toContain(
      "Continue working in this Builder session. New prompts will be queued even while a run is active."
    )

    await act(async () => {
      root.unmount()
    })
  })

  it("shows a default-expanded session history rail that can be collapsed", async () => {
    listBuilderSessionsMock.mockResolvedValue({
      items: [
        buildSession({ id: "session-1", title: "Landing page builder" }),
        buildSession({ id: "session-2", title: "API builder" }),
      ],
      pagination: DEFAULT_PAGINATION,
    })

    const { container, root } = await renderBuilderRoute(
      "/projects/project-1/builder-sessions/session-1"
    )

    await settle()

    expect(container.querySelector('[data-testid="builder-session-history-list"]')).not.toBeNull()

    const header = container.querySelector('[data-testid="builder-session-history-header"]')
    const footer = container.querySelector('[data-testid="builder-session-history-footer"]')
    const newConversationButton = container.querySelector('[data-testid="builder-session-history-new"]') as HTMLButtonElement | null
    const toggle = container.querySelector('[data-testid="builder-session-history-toggle"]') as HTMLButtonElement | null

    expect(toggle).not.toBeNull()
    expect(newConversationButton).not.toBeNull()
    expect(header?.contains(newConversationButton ?? null)).toBe(true)
    expect(footer?.contains(toggle ?? null)).toBe(true)

    await act(async () => {
      toggle?.click()
    })

    expect(container.querySelector('[data-testid="builder-session-history-list"]')).toBeNull()

    await act(async () => {
      root.unmount()
    })
  })

  it("publishes the Builder, environment, and current-session breadcrumb", async () => {
    getBuilderSessionMock.mockResolvedValue(
      buildDetail({
        session: buildSession({
          id: "session-1",
          title: "Landing page builder",
        }),
      })
    )

    const { root } = await renderBuilderRoute(
      "/projects/project-1/builder-sessions/session-1"
    )

    await settle()

    expect(pageHeaderItemsMock).toHaveBeenCalled()
    const latestItems = pageHeaderItemsMock.mock.calls.at(-1)?.[0] as Array<{
      label: string
      icon?: unknown
      dropdown?: unknown
    }> | undefined
    expect(latestItems?.map((item) => item.label)).toEqual([
      "Builder",
      "Builder Env",
      "Current Session",
    ])
    expect(latestItems?.[0]?.icon).toBeDefined()
    expect(latestItems?.[1]?.icon).toBeDefined()
    expect(latestItems?.[1]?.dropdown).toBeDefined()
    expect(latestItems?.[2]?.icon).toBeDefined()

    await act(async () => {
      root.unmount()
    })
  })

  it("keeps the draft workspace breadcrumb in Builder → environment → Current Session form", async () => {
    const { root } = await renderBuilderRoute(
      "/projects/project-1/builder-sessions?draft=1"
    )

    await settle()

    expect(pageHeaderItemsMock).toHaveBeenCalled()
    const latestItems = pageHeaderItemsMock.mock.calls.at(-1)?.[0] as Array<{
      label: string
      dropdown?: unknown
    }> | undefined
    expect(latestItems?.map((item) => item.label)).toEqual([
      "Builder",
      "Builder Env",
      "Current Session",
    ])
    expect(latestItems?.[1]?.dropdown).toBeDefined()

    await act(async () => {
      root.unmount()
    })
  })

  it("shows a delivery-only preview panel without opening the files rail", async () => {
    const sessionWithPreview = buildSession({
      id: "session-with-preview",
      artifact_count: 0,
    })
    getBuilderSessionMock.mockResolvedValue(
      buildDetail({
        session: sessionWithPreview,
        artifacts: [],
        preview: {
          available: true,
          status: "delivery_only",
          resolved_run_id: sessionWithPreview.latest_run_id,
          published_at: null,
          completed_at: null,
          output_root: "dist",
          default_entry_path: "",
          download_available: true,
          preview_available: false,
          is_stale: false,
          newer_run_id: "",
          newer_run_status: "",
          download_url: `/api/v1/projects/project-1/builder-sessions/${sessionWithPreview.id}/runs/${sessionWithPreview.latest_run_id}/delivery/download`,
          preview_launch_url: "",
        },
      })
    )

    const { container, root } = await renderBuilderRoute(
      "/projects/project-1/builder-sessions/session-with-preview"
    )

    await settle()

    expect(container.textContent).toContain("Preview output")
    expect(container.textContent).toContain("Download snapshot")
    expect(container.textContent).toContain("Preview is unavailable for this output")
    expect(container.querySelector('[data-testid="builder-files-rail"]')).toBeNull()

    await act(async () => {
      root.unmount()
    })
  })

  it("shows stale preview messaging when a newer run exists", async () => {
    const sessionWithStalePreview = buildSession({
      id: "session-with-stale-preview",
      latest_run_id: "run-3",
      latest_run_status: "failed",
    })
    getBuilderSessionMock.mockResolvedValue(
      buildDetail({
        session: sessionWithStalePreview,
        runs: [
          buildRun({ id: "run-3", session_id: sessionWithStalePreview.id, status: "failed" }),
          buildRun({ id: "run-2", session_id: sessionWithStalePreview.id, status: "succeeded" }),
        ],
        preview: {
          available: true,
          status: "previewable",
          resolved_run_id: "run-2",
          published_at: null,
          completed_at: null,
          output_root: "dist",
          default_entry_path: "dist/index.html",
          download_available: true,
          preview_available: true,
          is_stale: true,
          newer_run_id: "run-3",
          newer_run_status: "failed",
          download_url: `/api/v1/projects/project-1/builder-sessions/${sessionWithStalePreview.id}/runs/run-2/delivery/download`,
          preview_launch_url: `/api/v1/projects/project-1/builder-sessions/${sessionWithStalePreview.id}/runs/run-2/preview/launch`,
        },
      })
    )

    const { container, root } = await renderBuilderRoute(
      "/projects/project-1/builder-sessions/session-with-stale-preview"
    )

    await settle()

    expect(container.textContent).toContain("A newer run exists")
    expect(container.textContent).toContain("run-3")
    expect(container.textContent).toContain("failed")

    await act(async () => {
      root.unmount()
    })
  })

  it("launches a sandboxed preview iframe for previewable snapshots", async () => {
    const sessionWithPreview = buildSession({
      id: "session-previewable",
      artifact_count: 0,
    })
    getBuilderSessionMock.mockResolvedValue(
      buildDetail({
        session: sessionWithPreview,
        artifacts: [],
        preview: {
          available: true,
          status: "previewable",
          resolved_run_id: sessionWithPreview.latest_run_id,
          published_at: null,
          completed_at: null,
          output_root: "dist",
          default_entry_path: "dist/index.html",
          download_available: true,
          preview_available: true,
          is_stale: false,
          newer_run_id: "",
          newer_run_status: "",
          download_url: `/api/v1/projects/project-1/builder-sessions/${sessionWithPreview.id}/runs/${sessionWithPreview.latest_run_id}/delivery/download`,
          preview_launch_url: `/api/v1/projects/project-1/builder-sessions/${sessionWithPreview.id}/runs/${sessionWithPreview.latest_run_id}/preview/launch`,
        },
      })
    )
    launchBuilderPreviewMock.mockResolvedValue({
      frame_url: "/builder-preview/projects/project-1/sessions/session-previewable/runs/run-1/",
    })

    const { container, root } = await renderBuilderRoute(
      "/projects/project-1/builder-sessions/session-previewable"
    )

    await settle()

    const openPreviewButton = Array.from(container.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("Open preview")
    ) as HTMLButtonElement | undefined

    expect(openPreviewButton).toBeDefined()

    await act(async () => {
      openPreviewButton?.click()
      await flushPromises()
      await flushPromises()
    })

    expect(launchBuilderPreviewMock).toHaveBeenCalledWith("project-1", "session-previewable", "run-1")

    const iframe = container.querySelector('[data-testid="builder-preview-iframe"]') as HTMLIFrameElement | null
    expect(iframe).not.toBeNull()
    expect(iframe?.getAttribute("src")).toBe("/builder-preview/projects/project-1/sessions/session-previewable/runs/run-1/")
    expect(iframe?.getAttribute("sandbox")).toContain("allow-scripts")

    await act(async () => {
      root.unmount()
    })
  })

  it("does not render a preview iframe for delivery-only snapshots", async () => {
    const sessionWithDeliveryOnly = buildSession({
      id: "session-delivery-only",
      artifact_count: 0,
    })
    getBuilderSessionMock.mockResolvedValue(
      buildDetail({
        session: sessionWithDeliveryOnly,
        artifacts: [],
        preview: {
          available: true,
          status: "delivery_only",
          resolved_run_id: sessionWithDeliveryOnly.latest_run_id,
          published_at: null,
          completed_at: null,
          output_root: "dist",
          default_entry_path: "",
          download_available: true,
          preview_available: false,
          is_stale: false,
          newer_run_id: "",
          newer_run_status: "",
          download_url: `/api/v1/projects/project-1/builder-sessions/${sessionWithDeliveryOnly.id}/runs/${sessionWithDeliveryOnly.latest_run_id}/delivery/download`,
          preview_launch_url: "",
        },
      })
    )

    const { container, root } = await renderBuilderRoute(
      "/projects/project-1/builder-sessions/session-delivery-only"
    )

    await settle()

    expect(container.querySelector('[data-testid="builder-preview-iframe"]')).toBeNull()
    expect(launchBuilderPreviewMock).not.toHaveBeenCalled()

    await act(async () => {
      root.unmount()
    })
  })

  it("requires a build environment selection before creating the first Builder session", async () => {
    envsListMock.mockResolvedValue({
      items: [
        {
          id: "env-1",
          name: "Builder Env",
          is_build_env: true,
        },
        {
          id: "env-2",
          name: "Secondary Env",
          is_build_env: true,
        },
      ],
      pagination: DEFAULT_PAGINATION,
    })

    const { container, root } = await renderBuilderRoute("/projects/project-1/builder-sessions?draft=1")

    await settle()

    expect(container.querySelector('[data-testid="builder-composer"]')).not.toBeNull()
    expect(container.querySelector('[data-testid="builder-composer-footer"]')).not.toBeNull()
    expect(container.textContent).toContain("New conversation")
    expect(container.textContent).not.toContain("Build environment")
    expect(container.textContent).not.toContain("The first message creates the Builder session in the selected environment.")

    const composer = container.querySelector('[data-testid="builder-composer"]') as HTMLTextAreaElement | null
    const sendButton = container.querySelector('[data-testid="builder-send-message"]') as HTMLButtonElement | null

    await act(async () => {
      if (composer) {
        composer.value = "Build me a dashboard app"
        composer.dispatchEvent(new Event("input", { bubbles: true }))
      }
      sendButton?.click()
      await flushPromises()
    })

    expect(createBuilderSessionMock).not.toHaveBeenCalled()

    await act(async () => {
      root.unmount()
    })
  })

  it("shows grouped model choices and includes the selected model when creating a Builder session", async () => {
    const { container, root } = await renderBuilderRoute("/projects/project-1/builder-sessions?draft=1")

    await settle()

    expect(container.querySelector('[data-testid="builder-model-selector-compact"]')).not.toBeNull()
    expect(container.textContent).toContain("Claude 4 Sonnet")

    const composer = container.querySelector('[data-testid="builder-composer"]') as HTMLTextAreaElement | null
    const sendButton = container.querySelector('[data-testid="builder-send-message"]') as HTMLButtonElement | null

    await act(async () => {
      if (composer) {
        composer.value = "Build me a dashboard app"
        composer.dispatchEvent(new Event("input", { bubbles: true }))
      }
      sendButton?.click()
      await flushPromises()
    })

    expect(createBuilderSessionMock).toHaveBeenCalledWith("project-1", {
      build_env_id: "env-1",
      prompt: "Build me a dashboard app",
      selected_model_key: "project-claude-sonnet",
      provider_key: "anthropic-project",
      model_profile_key: "claude-sonnet-4",
    })

    await act(async () => {
      root.unmount()
    })
  })

  it("preselects the project default model and labels its source", async () => {
    getBuilderModelSelectionMock.mockResolvedValue({
      options: [
        {
          key: "project-claude-sonnet",
          modelLabel: "Claude 4 Sonnet",
          providerLabel: "Anthropic",
          scope: "project",
          providerKey: "anthropic-project",
          modelProfileKey: "claude-sonnet-4",
        },
      ],
      effectiveDefaultSource: "project",
      effectiveDefaultOption: {
        key: "project-claude-sonnet",
        modelLabel: "Claude 4 Sonnet",
        providerLabel: "Anthropic",
        scope: "project",
        providerKey: "anthropic-project",
        modelProfileKey: "claude-sonnet-4",
      },
    } satisfies BuilderModelSelection)

    const { container, root } = await renderBuilderRoute("/projects/project-1/builder-sessions?draft=1")

    await settle()

    expect(container.textContent).toContain("Claude 4 Sonnet")

    await act(async () => {
      root.unmount()
    })
  })

  it("falls back to the user default when no project default exists", async () => {
    getBuilderModelSelectionMock.mockResolvedValue({
      options: [
        {
          key: "user-gpt-4-1",
          modelLabel: "GPT-4.1",
          providerLabel: "OpenAI",
          scope: "user",
          providerKey: "openai-user",
          modelProfileKey: "gpt-4.1",
        },
      ],
      effectiveDefaultSource: "user",
      effectiveDefaultOption: {
        key: "user-gpt-4-1",
        modelLabel: "GPT-4.1",
        providerLabel: "OpenAI",
        scope: "user",
        providerKey: "openai-user",
        modelProfileKey: "gpt-4.1",
      },
    } satisfies BuilderModelSelection)

    const { container, root } = await renderBuilderRoute("/projects/project-1/builder-sessions?draft=1")

    await settle()

    expect(container.textContent).toContain("GPT-4.1")

    await act(async () => {
      root.unmount()
    })
  })

  it("leaves the selector unselected when no default exists", async () => {
    getBuilderModelSelectionMock.mockResolvedValue({
      options: [],
      effectiveDefaultSource: "none",
    } satisfies BuilderModelSelection)

    const { container, root } = await renderBuilderRoute("/projects/project-1/builder-sessions?draft=1")

    await settle()

    expect(container.textContent).not.toContain("Default from project settings")
    expect(container.textContent).not.toContain("Default from your account settings")

    await act(async () => {
      root.unmount()
    })
  })
})
