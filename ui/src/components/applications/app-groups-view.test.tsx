import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const GROUPS = [
  {
    id: "group-frontend",
    env_id: "env-1",
    name: "Frontend",
    description: "",
    apps: [],
  },
  {
    id: "group-backend",
    env_id: "env-1",
    name: "Backend",
    description: "",
    apps: [],
  },
] as const

const GROUP_APPS = {
  "group-frontend": [
    {
      id: "app-1",
      name: "Portal",
      slug: "portal",
      status: "running",
      app_type: "deployment",
      container_image: "nginx:latest",
      replicas: 1,
      created_at: "2026-03-01T00:00:00Z",
      available_actions: [],
    },
  ],
  "group-backend": [
    {
      id: "app-2",
      name: "API",
      slug: "api",
      status: "running",
      app_type: "deployment",
      container_image: "golang:latest",
      replicas: 1,
      created_at: "2026-03-01T00:00:00Z",
      available_actions: [],
    },
  ],
} as const

vi.mock("@tanstack/react-query", () => ({
  useQuery: ({ queryKey }: { queryKey: unknown[] }) => {
    const key = queryKey[0]

    if (key === "app-groups") {
      return {
        data: GROUPS,
        isLoading: false,
        isFetching: false,
        error: null,
      }
    }

    if (key === "group-apps") {
      const groupId = queryKey[1] as keyof typeof GROUP_APPS
      return {
        data: {
          items: GROUP_APPS[groupId] ?? [],
          pagination: {
            total: (GROUP_APPS[groupId] ?? []).length,
            page: 1,
            page_size: 10,
            total_pages: 1,
          },
        },
        isLoading: false,
        isFetching: false,
        error: null,
      }
    }

    if (key === "app-favorites") {
      return {
        data: [],
        isLoading: false,
        isFetching: false,
        error: null,
      }
    }

    return {
      data: undefined,
      isLoading: false,
      isFetching: false,
      error: null,
    }
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
  useNavigate: () => vi.fn(),
}))

vi.mock("@/hooks/use-debounce", () => ({
  useDebounce: <T,>(value: T) => value,
}))

vi.mock("@/hooks/useProjectRole", () => ({
  useProjectRole: () => "viewer",
}))

vi.mock("@/components/data-table/data-table", () => ({
  DataTable: ({
    leftToolbar,
    rightToolbar,
  }: {
    leftToolbar?: () => React.ReactNode
    rightToolbar?: () => React.ReactNode
  }) => (
    <div data-testid="data-table">
      <div data-testid="left-toolbar">{leftToolbar?.()}</div>
      <div data-testid="right-toolbar">{rightToolbar?.()}</div>
    </div>
  ),
}))

vi.mock("@/components/ui/tabs", () => ({
  Tabs: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  TabsList: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  TabsTrigger: ({ children }: { children: React.ReactNode }) => <button type="button">{children}</button>,
}))

vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenu: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DropdownMenuContent: () => null,
  DropdownMenuItem: () => null,
  DropdownMenuTrigger: ({ render }: { render?: React.ReactNode }) => <>{render ?? null}</>,
}))

vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  TooltipContent: () => null,
  TooltipTrigger: ({
    render,
    children,
  }: {
    render?: React.ReactNode
    children?: React.ReactNode
  }) => <>{render ?? children ?? null}</>,
}))

vi.mock("@/components/applications/app-action-icons-wrapper", () => ({
  AppActionIconsWrapper: () => null,
}))

vi.mock("@/components/applications/create-app-dialog", () => ({
  CreateAppDialog: () => null,
}))

vi.mock("@/components/applications/edit-app-dialog", () => ({
  EditAppDialog: () => null,
}))

vi.mock("@/components/applications/export-apps-dialog", () => ({
  ExportAppsDialog: () => null,
}))

vi.mock("@/components/applications/import-apps-dialog", () => ({
  ImportAppsDialog: () => null,
}))

vi.mock("./edit-app-group-dialog", () => ({
  EditAppGroupDialog: () => null,
}))

import { AppGroupsView } from "./app-groups-view"

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

describe("AppGroupsView", () => {
  beforeEach(() => {
    Object.defineProperty(globalThis, "localStorage", {
      configurable: true,
      value: createMemoryStorage(),
    })
  })

  afterEach(() => {
    document.body.innerHTML = ""
  })

  it("renders group name and actions to the left of the filter and separates groups with a dashed separator", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)

    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<AppGroupsView envId="env-1" />)
    })

    const leftToolbars = Array.from(
      container.querySelectorAll('[data-testid="left-toolbar"]')
    )
    const firstToolbar = leftToolbars[0] as HTMLElement | undefined
    const toolbarInput = firstToolbar?.querySelector(
      'input[placeholder="Search applications..."]'
    ) as HTMLInputElement | null
    const titleElement = Array.from(firstToolbar?.querySelectorAll("*") ?? []).find(
      (element) => element.textContent?.trim() === "Frontend"
    )
    const separators = Array.from(container.querySelectorAll('[data-slot="separator"]'))

    expect(firstToolbar).toBeDefined()
    expect(titleElement).toBeDefined()
    expect(firstToolbar?.querySelectorAll("button").length).toBeGreaterThan(0)
    expect(toolbarInput).not.toBeNull()
    if (!titleElement || !toolbarInput) {
      throw new Error("Expected group title and toolbar filter input to be rendered")
    }
    expect(
      titleElement.compareDocumentPosition(toolbarInput) &
        Node.DOCUMENT_POSITION_FOLLOWING
    ).toBeTruthy()
    expect(separators).toHaveLength(1)
    expect(separators[0]?.className).toContain("border-dashed")

    await act(async () => {
      root.unmount()
    })
  })
})
