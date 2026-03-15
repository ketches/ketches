import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const {
  mockUseMutation,
  mockUseQuery,
  mockUseQueryClient,
} = vi.hoisted(() => ({
  mockUseMutation: vi.fn(),
  mockUseQuery: vi.fn(),
  mockUseQueryClient: vi.fn(),
}))

vi.mock("@tanstack/react-query", async () => {
  const actual = await vi.importActual<typeof import("@tanstack/react-query")>("@tanstack/react-query")
  return {
    ...actual,
    useMutation: (...args: unknown[]) => mockUseMutation(...args),
    useQuery: (...args: unknown[]) => mockUseQuery(...args),
    useQueryClient: (...args: unknown[]) => mockUseQueryClient(...args),
  }
})

vi.mock("@/stores/auth", () => ({
  useAuthStore: (selector: (state: { user: { role: string } | null }) => unknown) =>
    selector({ user: { role: "member" } }),
}))

vi.mock("@/hooks/use-debounce", () => ({
  useDebounce: (value: string) => value,
}))

vi.mock("@/components/layout/page-header", () => ({
  PageHeader: () => null,
}))

vi.mock("@/components/shared/color-badge", () => ({
  ColorBadge: ({ children }: { children: React.ReactNode }) => <span>{children}</span>,
}))

vi.mock("@/components/shared/empty-state", () => ({
  EmptyState: ({ title, description }: { title: string; description?: string }) => (
    <div>
      <div>{title}</div>
      {description ? <div>{description}</div> : null}
    </div>
  ),
}))

vi.mock("@/components/data-table/data-table", () => ({
  DataTable: ({
    data,
    emptyContent,
    leftToolbar,
    rightToolbar,
  }: {
    data?: unknown[]
    emptyContent?: React.ReactNode
    leftToolbar?: () => React.ReactNode
    rightToolbar?: () => React.ReactNode
  }) => (
    <div>
      <div>{leftToolbar?.()}</div>
      <div>{rightToolbar?.()}</div>
      <div>{Array.isArray(data) && data.length > 0 ? "has-data" : emptyContent ?? "No results."}</div>
    </div>
  ),
}))

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, ...props }: React.ComponentProps<"button">) => <button {...props}>{children}</button>,
}))

vi.mock("@/components/ui/input", () => ({
  Input: (props: React.ComponentProps<"input">) => <input {...props} />,
}))

vi.mock("@/components/ui/combobox", () => ({
  Combobox: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  ComboboxContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  ComboboxInput: (props: React.ComponentProps<"input">) => <input {...props} />,
  ComboboxItem: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  ComboboxList: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/popover", () => ({
  Popover: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  PopoverContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  PopoverTrigger: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/calendar", () => ({
  Calendar: () => <div>calendar</div>,
}))

vi.mock("@/api/operation-logs", () => ({
  activitiesApi: {
    list: vi.fn(),
  },
  operationLogsApi: {
    getOperationLogSettings: vi.fn(),
    listOperationLogs: vi.fn(),
    updateOperationLogSettings: vi.fn(),
    exportOperationLogsCSV: vi.fn(),
  },
}))

vi.mock("sonner", () => ({
  toast: {
    error: vi.fn(),
    success: vi.fn(),
  },
}))

import { ActivitiesPage } from "./activities-page"

describe("ActivitiesPage", () => {
  beforeEach(() => {
    mockUseQueryClient.mockReturnValue({
      invalidateQueries: vi.fn(),
    })
    mockUseMutation.mockReturnValue({
      isPending: false,
      mutate: vi.fn(),
    })
    mockUseQuery.mockImplementation(({ queryKey }: { queryKey: unknown[] }) => {
      if (queryKey[0] === "operation-log-settings") {
        return {
          data: undefined,
          isLoading: false,
          isFetching: false,
        }
      }

      return {
        data: {
          items: [],
          pagination: {
            total: 0,
            page: 1,
            page_size: 10,
            total_pages: 0,
          },
        },
        isLoading: false,
        isFetching: false,
      }
    })
  })

  afterEach(() => {
    document.body.innerHTML = ""
    vi.clearAllMocks()
  })

  it("keeps the filters visible when the filtered activity result is empty", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<ActivitiesPage />)
    })

    const searchInput = container.querySelector('input[placeholder="Filter by action, resource..."]')

    expect(searchInput).not.toBeNull()
    expect(container.textContent).toContain("No activities found")

    await act(async () => {
      root.unmount()
    })
  })
})
