import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import type { Sprint } from "@/api/collaboration"
import { collaborationApi } from "@/api/collaboration"
import CollaborationsPage from "./collaborations-page"

vi.mock("@/api/collaboration", () => ({
  collaborationApi: {
    listSprints: vi.fn(),
  },
}))

vi.mock("@/components/layout/page-header", () => ({
  PageHeader: () => null,
}))

vi.mock("./backlog-page", () => ({ default: () => null }))
vi.mock("./defects-page", () => ({ default: () => null }))
vi.mock("./requirements-page", () => ({ default: () => null }))
vi.mock("./sprints-page", () => ({ default: () => <div data-testid="sprints-page" /> }))
vi.mock("./tasks-page", () => ({ default: () => <div data-testid="tasks-page" /> }))
vi.mock("./test-cases-page", () => ({ default: () => null }))
vi.mock("@/components/collaborations/sprint-dialogs", () => ({
  CreateSprintDialog: ({ open }: { open: boolean }) => <div data-testid="create-sprint-dialog">{String(open)}</div>,
}))

const listSprintsMock = vi.mocked(collaborationApi.listSprints)

const ACTIVE_SPRINT: Sprint = {
  id: "sprint-123",
  project_id: "project-1",
  name: "Sprint Alpha",
  goal: "Ship the sprint filter fix",
  status: "active",
  start_date: "2026-03-01T00:00:00Z",
  end_date: "2026-03-14T00:00:00Z",
  created_by: "user-1",
  updated_by: "user-1",
  created_at: "2026-03-01T00:00:00Z",
  updated_at: "2026-03-01T00:00:00Z",
}

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

describe("CollaborationsPage", () => {
  beforeEach(() => {
    vi.stubGlobal("localStorage", createMemoryStorage())
    localStorage.clear()
    vi.clearAllMocks()

    if (!("ResizeObserver" in globalThis)) {
      vi.stubGlobal(
        "ResizeObserver",
        class ResizeObserver {
          observe() {}
          unobserve() {}
          disconnect() {}
        }
      )
    }

    const elementPrototype = Element.prototype as Element & {
      scrollIntoView?: () => void
    }

    if (!elementPrototype.scrollIntoView) {
      elementPrototype.scrollIntoView = () => {}
    }
  })

  afterEach(() => {
    document.body.innerHTML = ""
  })

  it("shows the sprint name for a persisted sprint selection after sprints load", async () => {
    localStorage.setItem("collab-sprint", ACTIVE_SPRINT.id)

    let resolveSprints: ((value: Awaited<ReturnType<typeof collaborationApi.listSprints>>) => void) | undefined
    listSprintsMock.mockReturnValue(
      new Promise((resolve) => {
        resolveSprints = resolve
      }) as ReturnType<typeof collaborationApi.listSprints>
    )

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
          <CollaborationsPage projectId="project-1" />
        </QueryClientProvider>
      )
    })

    expect(container.querySelector('input[placeholder="Filter by sprint..."]')).toBeNull()

    await act(async () => {
      resolveSprints?.({
        items: [ACTIVE_SPRINT],
        pagination: {
          page: 1,
          page_size: 100,
          total: 1,
          total_pages: 1,
        },
      })
      await flushPromises()
    })

    await act(async () => {
      await flushPromises()
    })

    const sprintInput = container.querySelector('input[placeholder="Filter by sprint..."]') as HTMLInputElement | null
    expect(sprintInput?.value).toBe(ACTIVE_SPRINT.name)

    await act(async () => {
      root.unmount()
    })
  })

  it("shows only scope switching and skeletons before sprints load", async () => {
    let resolveSprints: ((value: Awaited<ReturnType<typeof collaborationApi.listSprints>>) => void) | undefined
    listSprintsMock.mockReturnValue(
      new Promise((resolve) => {
        resolveSprints = resolve
      }) as ReturnType<typeof collaborationApi.listSprints>
    )

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
          <CollaborationsPage projectId="project-1" />
        </QueryClientProvider>
      )
    })

    expect(container.textContent?.includes("My Items")).toBe(true)
    expect(container.textContent?.includes("All Items")).toBe(true)
    expect(container.querySelector('input[placeholder="Filter by sprint..."]')).toBeNull()
    expect(container.textContent?.includes("Sprints")).toBe(false)
    expect(container.textContent?.includes("Tasks")).toBe(false)
    expect(container.querySelector('[data-slot="skeleton"]')).not.toBeNull()

    await act(async () => {
      resolveSprints?.({
        items: [ACTIVE_SPRINT],
        pagination: {
          page: 1,
          page_size: 100,
          total: 1,
          total_pages: 1,
        },
      })
      await flushPromises()
    })

    await act(async () => {
      root.unmount()
    })
  })

  it("shows the create sprint empty state for my items when the project has no sprints", async () => {
    localStorage.setItem("collab-scope", "my-items")
    listSprintsMock.mockResolvedValue({
      items: [],
      pagination: {
        page: 1,
        page_size: 100,
        total: 0,
        total_pages: 0,
      },
    })

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
          <CollaborationsPage projectId="project-1" />
        </QueryClientProvider>
      )
      await flushPromises()
    })

    await act(async () => {
      await flushPromises()
    })

    expect(container.querySelector('input[placeholder="Filter by sprint..."]')).toBeNull()
    expect(container.textContent?.includes("Sprints")).toBe(false)
    expect(container.textContent?.includes("Tasks")).toBe(false)
    expect(container.querySelector('[data-testid="tasks-page"]')).toBeNull()
    expect(container.textContent?.includes("No sprints yet")).toBe(true)
    expect(container.textContent?.includes("Create your first sprint to plan iteration work.")).toBe(true)
    expect(container.textContent?.includes("New Sprint")).toBe(true)

    await act(async () => {
      root.unmount()
    })
  })
})
