import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const {
  mockRole,
  mockUseMutation,
  mockUseQuery,
  mockUseQueryClient,
} = vi.hoisted(() => ({
  mockRole: { current: "member" as "member" | "admin" },
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
    selector({ user: { role: mockRole.current } }),
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
    <div data-testid="activities-table">
      <div data-testid="left-toolbar">{leftToolbar?.()}</div>
      <div data-testid="right-toolbar">{rightToolbar?.()}</div>
      <div data-testid="table-body">
        {Array.isArray(data) && data.length > 0 ? "has-data" : emptyContent ?? "No results."}
      </div>
    </div>
  ),
}))

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, className, ...props }: React.ComponentProps<"button">) => <button className={className} {...props}>{children}</button>,
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

vi.mock("sonner", () => ({
  toast: {
    error: vi.fn(),
    success: vi.fn(),
  },
}))

import { ActivitiesPage } from "./activities-page"

function mockActivitiesResponse(items: Array<Record<string, unknown>>) {
  mockUseQuery.mockImplementation(({ queryKey }: { queryKey: unknown[] }) => {
    if (queryKey[0] === "operation-log-settings") {
      return {
        data: {
          retention_days: 90,
        },
        isLoading: false,
        isFetching: false,
      }
    }

    return {
      data: {
        items,
        pagination: {
          total: items.length,
          page: 1,
          page_size: 10,
          total_pages: items.length > 0 ? 1 : 0,
        },
      },
      isLoading: false,
      isFetching: false,
    }
  })
}

describe("ActivitiesPage", () => {
  beforeEach(() => {
    mockUseQueryClient.mockReturnValue({
      invalidateQueries: vi.fn(),
    })
    mockUseMutation.mockReturnValue({
      isPending: false,
      mutate: vi.fn(),
    })
  })

  afterEach(() => {
    document.body.innerHTML = ""
    vi.clearAllMocks()
    mockRole.current = "member"
  })

  it("keeps filters visible when the result is empty", async () => {
    mockRole.current = "member"
    mockActivitiesResponse([])

    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<ActivitiesPage />)
    })

    expect(container.querySelector('input[placeholder="Filter by action, resource..."]')).not.toBeNull()
    expect(container.textContent).toContain("No activities found")

    await act(async () => {
      root.unmount()
    })
  })

  it("keeps export for admins without rendering inline retention controls", async () => {
    mockRole.current = "admin"
    mockActivitiesResponse([
      {
        id: "log-1",
        created_at: "2026-03-15T00:00:00Z",
        username: "admin",
        action: "update",
        resource_type: "platform",
        resource_id: "platform",
        status: "success",
        status_code: 200,
        sensitivity: "internal",
        request_summary: "updated branding",
        client_ip: "127.0.0.1",
      },
    ])

    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<ActivitiesPage />)
    })

    expect(container.textContent).toContain("Export CSV")
    expect(container.textContent).not.toContain("Save Retention Days")
    expect(container.textContent).not.toContain("manage retention policy")

    await act(async () => {
      root.unmount()
    })
  })

  it("renders a single-line adaptive toolbar wrapper for the activities filters", async () => {
    mockRole.current = "admin"
    mockActivitiesResponse([
      {
        id: "log-1",
        created_at: "2026-03-15T00:00:00Z",
        username: "admin",
        action: "update",
        resource_type: "platform",
        resource_id: "platform",
        status: "success",
        status_code: 200,
        sensitivity: "internal",
        request_summary: "updated branding",
        client_ip: "127.0.0.1",
      },
    ])

    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<ActivitiesPage />)
    })

    const toolbarWrapper = container.querySelector('[data-testid="left-toolbar"] > div')
    const searchInput = container.querySelector('input[placeholder="Filter by user, action, resource..."]')

    expect(toolbarWrapper).not.toBeNull()
    expect(toolbarWrapper?.className).toContain("overflow-x-auto")
    expect(toolbarWrapper?.className).toContain("min-w-0")
    expect(searchInput?.className).toContain("flex-1")

    await act(async () => {
      root.unmount()
    })
  })
})
