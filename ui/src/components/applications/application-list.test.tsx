import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const {
  dataTableProps,
  mockUseQuery,
  queryConfigs,
} = vi.hoisted(() => ({
  dataTableProps: [] as Array<Record<string, unknown>>,
  mockUseQuery: vi.fn(),
  queryConfigs: [] as Array<{ queryKey: unknown[]; refetchInterval?: unknown }>,
}))

const APPS = [
  {
    id: "app-1",
    name: "Portal",
    slug: "portal",
    description: "",
    env_id: "env-1",
    app_type: "Deployment",
    container_image: "nginx:latest",
    replicas: 1,
    status: "running",
    available_actions: [
      {
        action: "restart",
        label: "Restart",
        icon: "rotate-cw",
        category: "secondary",
        variant: "default",
      },
    ],
    created_at: "2026-03-01T00:00:00Z",
  },
] as const

vi.mock("@tanstack/react-query", () => ({
  useQuery: (config: { queryKey: unknown[]; refetchInterval?: unknown }) => {
    queryConfigs.push(config)
    return mockUseQuery(config)
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
  useProjectRole: () => "admin",
}))

vi.mock("@/components/data-table/data-table", () => ({
  DataTable: (props: Record<string, unknown>) => {
    dataTableProps.push(props)
    const columns = props.columns as Array<{
      id?: string
      cell?: (context: { row: { original: (typeof APPS)[number] } }) => React.ReactNode
    }>
    const actionsColumn = columns.find((column) => column.id === "actions")

    return (
      <div data-testid="data-table">
        {actionsColumn?.cell?.({ row: { original: APPS[0] } })}
      </div>
    )
  },
}))

vi.mock("@/components/applications/app-action-icons-wrapper", () => ({
  AppActionIconsWrapper: ({
    appId,
    onActionInteractionChange,
  }: {
    appId: string
    onActionInteractionChange?: (appId: string, active: boolean) => void
  }) => (
    <button
      type="button"
      data-testid={`actions-${appId}`}
      onClick={() => onActionInteractionChange?.(appId, true)}
    >
      Actions
    </button>
  ),
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

vi.mock("@/components/ui/alert-dialog", () => ({
  AlertDialog: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogAction: ({ children }: { children: React.ReactNode }) => <button type="button">{children}</button>,
  AlertDialogCancel: ({ children }: { children: React.ReactNode }) => <button type="button">{children}</button>,
  AlertDialogContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogDescription: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogFooter: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogTitle: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, type, ...props }: React.ComponentProps<"button">) => <button type={type ?? "button"} {...props}>{children}</button>,
}))

vi.mock("@/components/ui/checkbox", () => ({
  Checkbox: (props: React.ComponentProps<"input">) => <input type="checkbox" {...props} />,
}))

vi.mock("@/components/ui/input", () => ({
  Input: (props: React.ComponentProps<"input">) => <input {...props} />,
}))

vi.mock("@/components/ui/tabs", () => ({
  Tabs: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  TabsList: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  TabsTrigger: ({ children }: { children: React.ReactNode }) => <button type="button">{children}</button>,
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

vi.mock("sonner", () => ({
  toast: {
    error: vi.fn(),
    info: vi.fn(),
    success: vi.fn(),
  },
}))

import { ApplicationList } from "./application-list"

function latestDataTableProps() {
  return dataTableProps[dataTableProps.length - 1]
}

function latestAppsQueryConfig() {
  return [...queryConfigs].reverse().find((config) => config.queryKey[0] === "apps")
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

describe("ApplicationList", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    dataTableProps.length = 0
    queryConfigs.length = 0
    mockUseQuery.mockImplementation(({ queryKey }: { queryKey: unknown[] }) => {
      if (queryKey[0] === "apps") {
        return {
          data: {
            items: APPS,
            pagination: {
              total: APPS.length,
              page: 1,
              page_size: 10,
              total_pages: 1,
            },
          },
          isLoading: false,
          isFetching: true,
          refetch: vi.fn(),
        }
      }

      if (queryKey[0] === "app-favorites" || queryKey[0] === "app-groups") {
        return {
          data: [],
          isLoading: false,
          isFetching: false,
          refetch: vi.fn(),
        }
      }

      return {
        data: undefined,
        isLoading: false,
        isFetching: false,
        refetch: vi.fn(),
      }
    })
    Object.defineProperty(globalThis, "localStorage", {
      configurable: true,
      value: createMemoryStorage(),
    })
  })

  afterEach(() => {
    document.body.innerHTML = ""
  })

  it("pauses automatic refresh and hides fetching overlay while an action menu is active", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<ApplicationList envId="env-1" />)
    })

    expect(latestAppsQueryConfig()?.refetchInterval).toBe(5000)
    expect(latestDataTableProps()?.isLoading).toBe(true)

    const actionButton = container.querySelector('[data-testid="actions-app-1"]') as HTMLButtonElement
    await act(async () => {
      actionButton.dispatchEvent(new MouseEvent("click", { bubbles: true }))
    })

    expect(latestAppsQueryConfig()?.refetchInterval).toBe(false)
    expect(latestDataTableProps()?.isLoading).toBe(false)

    await act(async () => {
      root.unmount()
    })
  })

  it("uses application ids as stable table row ids", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<ApplicationList envId="env-1" />)
    })

    const getRowId = latestDataTableProps()?.getRowId as ((app: (typeof APPS)[number]) => string) | undefined
    expect(getRowId?.(APPS[0])).toBe("app-1")

    await act(async () => {
      root.unmount()
    })
  })
})
