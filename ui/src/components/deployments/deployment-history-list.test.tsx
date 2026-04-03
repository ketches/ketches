import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const {
  mockInvalidateQueries,
  mockMutate,
  mockUseQuery,
} = vi.hoisted(() => ({
  mockInvalidateQueries: vi.fn(),
  mockMutate: vi.fn(),
  mockUseQuery: vi.fn(),
}))

vi.mock("@tanstack/react-query", () => ({
  useMutation: () => ({
    mutate: mockMutate,
    isPending: false,
  }),
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
  useQueryClient: () => ({
    invalidateQueries: mockInvalidateQueries,
  }),
}))

vi.mock("@/components/data-table/data-table", () => ({
  DataTable: ({
    columns,
    data,
  }: {
    columns: Array<{
      id?: string
      accessorKey?: string
      cell?: (args: { row: { original: Record<string, unknown> } }) => React.ReactNode
    }>
    data: Array<Record<string, unknown>>
  }) => (
    <div>
      {data.map((row, rowIndex) => (
        <div key={String(row.id ?? rowIndex)}>
          {columns.map((column, columnIndex) => (
            <div key={`${column.id ?? column.accessorKey ?? columnIndex}-${rowIndex}`}>
              {column.cell ? column.cell({ row: { original: row } }) : null}
            </div>
          ))}
        </div>
      ))}
    </div>
  ),
}))

vi.mock("@/components/shared/empty-state", () => ({
  EmptyState: ({ title, description }: { title: string, description: string }) => (
    <div>
      <div>{title}</div>
      <div>{description}</div>
    </div>
  ),
}))

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, type, ...props }: React.ComponentProps<"button">) => <button type={type ?? "button"} {...props}>{children}</button>,
}))

vi.mock("@/components/ui/card", () => ({
  Card: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  CardContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  CardDescription: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  CardHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  CardTitle: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/dialog", () => ({
  Dialog: ({ open, children }: { open: boolean, children: React.ReactNode }) => open ? <div>{children}</div> : null,
  DialogContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogDescription: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogFooter: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogTitle: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

vi.mock("@/lib/utils", () => ({
  formatDate: (value: string) => value,
}))

import { DeploymentHistoryList } from "./deployment-history-list"

async function renderList() {
  const container = document.createElement("div")
  document.body.appendChild(container)
  const root = ReactDOMClient.createRoot(container)

  await act(async () => {
    root.render(<DeploymentHistoryList appId="app-1" />)
  })

  return { container, root }
}

describe("DeploymentHistoryList", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    ;(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true
    mockUseQuery.mockReturnValue({
      data: {
        items: [
          {
            id: "history-1",
            app_id: "app-1",
            created_at: "2026-04-03T00:00:00Z",
            image_before: "nginx:1.25",
            image_after: "nginx:1.26",
            replicas_before: 1,
            replicas_after: 2,
            request_cpu_before: 100,
            request_cpu_after: 250,
            request_memory_before: 128,
            request_memory_after: 256,
            limit_cpu_before: 200,
            limit_cpu_after: 500,
            limit_memory_before: 256,
            limit_memory_after: 512,
            deploy_type: "manual",
            deployed_by: "system",
            reason: "Container image updated",
            status: "success",
          },
        ],
        pagination: {
          total: 1,
        },
      },
      isLoading: false,
    })
  })

  afterEach(() => {
    document.body.innerHTML = ""
    vi.clearAllMocks()
  })

  it("describes rollback as image-only and omits configuration restore fields", async () => {
    const { container, root } = await renderList()

    const rollbackButton = container.querySelector('button[title="Rollback to this deployment"]')
    expect(rollbackButton).not.toBeNull()

    await act(async () => {
      rollbackButton?.dispatchEvent(new MouseEvent("click", { bubbles: true }))
    })

    expect(container.textContent).toContain(
      "This will restore only the previous image version. Other configuration changes are not rolled back.",
    )
    expect(container.textContent).toContain("Image:")
    expect(container.textContent).toContain("nginx:1.25")
    expect(container.textContent).not.toContain("Replicas:")
    expect(container.textContent).not.toContain("CPU Request:")
    expect(container.textContent).not.toContain("Memory Request:")

    await act(async () => {
      root.unmount()
    })
  })
})
