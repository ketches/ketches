import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const { mockUseQuery } = vi.hoisted(() => ({
  mockUseQuery: vi.fn(),
}))

vi.mock("@tanstack/react-query", async () => {
  const actual = await vi.importActual<typeof import("@tanstack/react-query")>("@tanstack/react-query")
  return {
    ...actual,
    useQuery: (...args: unknown[]) => mockUseQuery(...args),
  }
})

vi.mock("@/components/shared/empty-state", () => ({
  EmptyState: ({ title, description }: { title: string; description?: string }) => (
    <div>{title}{description}</div>
  ),
}))

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, ...props }: React.ComponentProps<"button">) => <button {...props}>{children}</button>,
}))

import { PlatformAuditLogTab } from "./platform-audit-log-tab"

describe("PlatformAuditLogTab", () => {
  beforeEach(() => {
    mockUseQuery.mockReturnValue({
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
    })
  })

  afterEach(() => {
    document.body.innerHTML = ""
    vi.clearAllMocks()
  })

  it("renders an empty state when no platform audit logs exist", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<PlatformAuditLogTab />)
    })

    expect(container.textContent).toContain("No platform audit logs")

    await act(async () => {
      root.unmount()
    })
  })

  it("renders platform audit log rows when data is returned", async () => {
    mockUseQuery.mockReturnValue({
      data: {
        items: [{
          id: "log-1",
          created_at: "2026-03-15T00:00:00Z",
          username: "admin",
          action: "update",
          resource_type: "platform_branding",
          resource_id: "platform",
          status: "success",
          status_code: 200,
          sensitivity: "sensitive",
          request_summary: "updated platform name",
          client_ip: "127.0.0.1",
        }],
        pagination: {
          total: 1,
          page: 1,
          page_size: 10,
          total_pages: 1,
        },
      },
      isLoading: false,
      isFetching: false,
    })

    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<PlatformAuditLogTab />)
    })

    expect(container.textContent).toContain("admin")
    expect(container.textContent).toContain("updated platform name")

    await act(async () => {
      root.unmount()
    })
  })
})
