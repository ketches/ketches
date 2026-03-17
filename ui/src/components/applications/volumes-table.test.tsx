import { act } from "react"
import ReactDOMClient from "react-dom/client"
import * as React from "react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const { mockInvalidateQueries } = vi.hoisted(() => ({
  mockInvalidateQueries: vi.fn(),
}))

vi.mock("@tanstack/react-query", () => ({
  useMutation: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
  useQuery: ({ queryKey }: { queryKey: unknown[] }) => {
    if (queryKey[0] === "app-volumes") {
      return {
        data: [
          {
            id: "vol-1",
            slug: "data",
            volume_type: "pvc",
            mount_path: "/data",
            capacity: 10,
            status: "Bound",
          },
          {
            id: "vol-2",
            slug: "cache",
            volume_type: "emptyDir",
            mount_path: "/cache",
            capacity: 1,
            status: "",
          },
        ],
        isLoading: false,
      }
    }

    return {
      data: [],
      isLoading: false,
    }
  },
  useQueryClient: () => ({
    invalidateQueries: mockInvalidateQueries,
  }),
}))

vi.mock("@/hooks/useProjectRole", () => ({
  useProjectRole: () => "viewer",
}))

vi.mock("@/components/applications/volume-editor", () => ({
  VolumeEditor: () => null,
}))

vi.mock("@/components/shared/empty-state", () => ({
  EmptyState: ({ title }: { title: string }) => <div>{title}</div>,
}))

vi.mock("@/components/ui/alert-dialog", () => ({
  AlertDialog: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  AlertDialogAction: ({ children }: { children: React.ReactNode }) => <button type="button">{children}</button>,
  AlertDialogCancel: ({ children }: { children: React.ReactNode }) => <button type="button">{children}</button>,
  AlertDialogContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogDescription: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogFooter: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogTitle: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/badge", () => ({
  Badge: ({ children }: { children: React.ReactNode }) => <span>{children}</span>,
}))

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, ...props }: React.ComponentProps<"button">) => <button {...props}>{children}</button>,
}))

vi.mock("@/components/ui/card", () => ({
  Card: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  CardContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  CardDescription: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  CardHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  CardTitle: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/checkbox", () => ({
  Checkbox: () => <input type="checkbox" />,
}))

vi.mock("@/components/ui/input", () => ({
  Input: (props: React.ComponentProps<"input">) => <input {...props} />,
}))

vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  TooltipTrigger: ({
    render,
    children,
  }: {
    render?: React.ReactNode
    children?: React.ReactNode
  }) => <>{render ?? children ?? null}</>,
  TooltipContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/data-table/data-table", () => ({
  DataTable: ({
    columns,
    data,
  }: {
    columns: Array<Record<string, unknown>>
    data: Array<Record<string, unknown>>
  }) => {
    const fakeTable = {
      getIsAllPageRowsSelected: () => false,
      toggleAllPageRowsSelected: () => undefined,
    }

    return (
      <table>
        <thead>
          <tr>
            {columns.map((column, columnIndex) => {
              const header = typeof column.header === "function"
                ? column.header({ table: fakeTable })
                : column.header

              return <th key={columnIndex}>{header as React.ReactNode}</th>
            })}
          </tr>
        </thead>
        <tbody>
          {data.map((row, rowIndex) => (
            <tr key={rowIndex}>
              {columns.map((column, columnIndex) => {
                const value = column.accessorKey ? row[String(column.accessorKey)] : undefined
                const cell = typeof column.cell === "function"
                  ? column.cell({
                    row: {
                      original: row,
                      getIsSelected: () => false,
                      toggleSelected: () => undefined,
                    },
                    getValue: () => value,
                  })
                  : value

                return <td key={columnIndex}>{cell as React.ReactNode}</td>
              })}
            </tr>
          ))}
        </tbody>
      </table>
    )
  },
}))

import { VolumesTable } from "./volumes-table"

describe("VolumesTable", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    ;(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true
  })

  afterEach(() => {
    document.body.innerHTML = ""
  })

  it("renders the status column and shows PVC status values", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(
        <VolumesTable
          app={{
            id: "app-1",
            env_id: "env-1",
          } as never}
        />
      )
    })

    expect(container.textContent).toContain("Status")
    expect(container.textContent).toContain("Bound")
    expect(container.textContent).toContain("cache")

    await act(async () => {
      root.unmount()
    })
  })
})
