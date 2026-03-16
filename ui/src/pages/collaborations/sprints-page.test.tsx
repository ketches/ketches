import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, describe, expect, it, vi } from "vitest"

const { refetchMock } = vi.hoisted(() => ({
  refetchMock: vi.fn(),
}))

const SPRINTS_RESPONSE = {
  items: [
    {
      id: "sprint-1",
      project_id: "project-1",
      name: "Sprint Alpha",
      goal: "Ship refresh support",
      status: "active",
      start_date: "2026-03-01",
      end_date: "2026-03-14",
      created_by: "user-1",
      updated_by: "user-1",
      created_at: "2026-03-01T00:00:00Z",
      updated_at: "2026-03-01T00:00:00Z",
    },
  ],
  pagination: {
    total: 1,
    page: 1,
    page_size: 20,
    total_pages: 1,
  },
} as const

vi.mock("@tanstack/react-query", () => ({
  useQuery: ({ queryKey }: { queryKey: unknown[] }) => {
    if (queryKey[0] === "sprints") {
      return {
        data: SPRINTS_RESPONSE,
        isLoading: false,
        refetch: refetchMock,
      }
    }

    return {
      data: undefined,
      isLoading: false,
      refetch: vi.fn(),
    }
  },
  useMutation: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
  useQueryClient: () => ({
    invalidateQueries: vi.fn(),
  }),
}))

vi.mock("react-router-dom", () => ({
  useParams: () => ({ projectId: "project-1" }),
}))

vi.mock("@/hooks/use-debounce", () => ({
  useDebounce: <T,>(value: T) => value,
}))

vi.mock("@/components/data-table/data-table", () => ({
  DataTable: ({ onRefresh }: { onRefresh?: () => void }) => (
    <div>
      {onRefresh ? (
        <button type="button" data-testid="refresh-sprints" onClick={onRefresh}>
          Refresh
        </button>
      ) : (
        <div data-testid="missing-refresh" />
      )}
    </div>
  ),
}))

vi.mock("@/components/collaborations/inline-editors", () => ({
  InlineStatusEditor: () => null,
}))

vi.mock("@/components/collaborations/sprint-dialogs", () => ({
  CreateSprintDialog: () => null,
  EditSprintDialog: () => null,
}))

vi.mock("@/components/layout/page-header", () => ({
  PageHeader: () => null,
}))

vi.mock("@/components/shared/empty-state", () => ({
  EmptyState: () => null,
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

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, ...props }: React.ComponentProps<"button">) => <button type="button" {...props}>{children}</button>,
}))

vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenu: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DropdownMenuContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DropdownMenuItem: ({ children, ...props }: React.ComponentProps<"button">) => <button type="button" {...props}>{children}</button>,
  DropdownMenuTrigger: ({ render }: { render?: React.ReactNode }) => <>{render ?? null}</>,
}))

vi.mock("@/components/ui/input", () => ({
  Input: (props: React.ComponentProps<"input">) => <input {...props} />,
}))

import SprintsPage from "./sprints-page"

describe("SprintsPage", () => {
  afterEach(() => {
    document.body.innerHTML = ""
    refetchMock.mockReset()
  })

  it("wires the refresh action to the sprint list refetch", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)

    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<SprintsPage projectId="project-1" />)
    })

    const refreshButton = container.querySelector('[data-testid="refresh-sprints"]') as HTMLButtonElement | null

    expect(refreshButton).not.toBeNull()

    await act(async () => {
      refreshButton?.click()
    })

    expect(refetchMock).toHaveBeenCalledTimes(1)

    await act(async () => {
      root.unmount()
    })
  })
})
