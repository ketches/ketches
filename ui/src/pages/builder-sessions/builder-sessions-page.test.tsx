import { act } from "react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
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
    launchBuilderPreviewMock,
    listBuilderFilesMock,
    readBuilderFileMock,
    downloadTarBlobMock,
    envsListMock,
  mockAuthState,
} = vi.hoisted(() => ({
  listBuilderSessionsMock: vi.fn(),
  getBuilderSessionMock: vi.fn(),
  getBuilderModelSelectionMock: vi.fn(),
    createBuilderSessionMock: vi.fn(),
    postBuilderMessageMock: vi.fn(),
    launchBuilderPreviewMock: vi.fn(),
    listBuilderFilesMock: vi.fn(),
    readBuilderFileMock: vi.fn(),
    downloadTarBlobMock: vi.fn(),
  envsListMock: vi.fn(),
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
      create: createBuilderSessionMock,
      postMessage: postBuilderMessageMock,
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

vi.mock("@/components/layout/page-header", () => ({
  PageHeader: () => null,
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
  ComboboxInput: (props: React.ComponentProps<"input">) => <input {...props} />,
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
  onerror: ((this: EventSource, ev: Event) => unknown) | null = null
  onmessage: ((this: EventSource, ev: MessageEvent) => unknown) | null = null
  onopen: ((this: EventSource, ev: Event) => unknown) | null = null

  constructor(url: string) {
    this.url = url
  }

  addEventListener() {}

  removeEventListener() {}

  dispatchEvent() {
    return true
  }

  close() {}
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
    requested_by: "user-1",
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
    launchBuilderPreviewMock.mockReset()
    listBuilderFilesMock.mockReset()
    readBuilderFileMock.mockReset()
    downloadTarBlobMock.mockReset()
    envsListMock.mockReset()

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
      effective_default_source: "project",
      effective_default_option: {
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

    expect(historyRail).not.toBeNull()
    expect(historyRail?.textContent).toContain("Earlier builder session")

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

    expect(newConversationButton).toBeDefined()

    await act(async () => {
      newConversationButton?.click()
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
    })

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

  it("shows grouped model choices and includes the selected model when creating a Builder session", async () => {
    const { container, root } = await renderBuilderRoute("/projects/project-1/builder-sessions?draft=1")

    await settle()

    expect(container.textContent).toContain("Project models")
    expect(container.textContent).toContain("My models")
    expect(container.textContent).toContain("Anthropic · Project")
    expect(container.textContent).toContain("OpenAI · User")

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
      effective_default_source: "project",
      effective_default_option: {
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

    expect(container.textContent).toContain("Default from project settings")
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
      effective_default_source: "user",
      effective_default_option: {
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

    expect(container.textContent).toContain("Default from your account settings")
    expect(container.textContent).toContain("GPT-4.1")

    await act(async () => {
      root.unmount()
    })
  })

  it("leaves the selector unselected when no default exists", async () => {
    getBuilderModelSelectionMock.mockResolvedValue({
      options: [],
      effective_default_source: "none",
    } satisfies BuilderModelSelection)

    const { container, root } = await renderBuilderRoute("/projects/project-1/builder-sessions?draft=1")

    await settle()

    expect(container.textContent).toContain("Model")
    expect(container.textContent).not.toContain("Default from project settings")
    expect(container.textContent).not.toContain("Default from your account settings")

    await act(async () => {
      root.unmount()
    })
  })
})
