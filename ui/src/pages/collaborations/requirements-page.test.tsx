import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, describe, expect, it, vi } from "vitest"

const { refetchMock, requirementsResponseRef } = vi.hoisted(() => ({
  refetchMock: vi.fn(),
  requirementsResponseRef: {
    current: undefined as typeof REQUIREMENTS_RESPONSE | typeof EMPTY_REQUIREMENTS_RESPONSE | undefined,
  },
}))

const REQUIREMENTS_RESPONSE = {
  items: [
    {
      id: "requirement-1",
      project_id: "project-1",
      sprint_id: "sprint-1",
      title: "Support empty-state actions",
      description: "Requirement description",
      status: "draft",
      priority: "p1",
      planning_status: "planned",
      assignee_id: "user-1",
      parent_requirement_id: "",
      depth: 0,
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

const EMPTY_REQUIREMENTS_RESPONSE = {
  items: [],
  pagination: {
    total: 0,
    page: 1,
    page_size: 20,
    total_pages: 0,
  },
} as const

requirementsResponseRef.current = REQUIREMENTS_RESPONSE

vi.mock("@tanstack/react-query", () => ({
  useQuery: ({ queryKey }: { queryKey: unknown[] }) => {
    if (queryKey[0] === "requirements") {
      return {
        data: requirementsResponseRef.current,
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
  DataTable: ({
    onRefresh,
    sourceEmptyContent,
  }: {
    onRefresh?: () => void
    sourceEmptyContent?: React.ReactNode
  }) => (
    <div>
      {onRefresh ? (
        <button type="button" data-testid="refresh-requirements" onClick={onRefresh}>
          Refresh
        </button>
      ) : (
        <div data-testid="missing-refresh" />
      )}
      <div data-testid="requirements-source-empty">{sourceEmptyContent}</div>
    </div>
  ),
}))

vi.mock("@/components/collaborations/collab-filters", () => ({
  AssigneeFilter: () => null,
  PriorityFilter: () => null,
  StatusFilter: () => null,
}))

vi.mock("@/components/collaborations/inline-editors", () => ({
  InlineAssigneeEditor: () => null,
  InlinePriorityEditor: () => null,
  InlineStatusEditor: () => null,
}))

vi.mock("@/components/collaborations/requirement-dialogs", () => ({
  CreateRequirementDialog: ({ open }: { open: boolean }) => <div data-testid="create-requirement-dialog">{String(open)}</div>,
  DeleteRequirementDialog: () => null,
  EditRequirementDialog: () => null,
}))

vi.mock("@/components/layout/page-header", () => ({
  PageHeader: () => null,
}))

vi.mock("@/components/shared/empty-state", () => ({
  EmptyState: ({
    title,
    description,
    actionText,
    onAction,
  }: {
    title: string
    description?: string
    actionText?: string
    onAction?: () => void
  }) => (
    <div>
      <div>{title}</div>
      {description ? <div>{description}</div> : null}
      {actionText && onAction ? (
        <button type="button" onClick={onAction}>
          {actionText}
        </button>
      ) : null}
    </div>
  ),
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

import RequirementsPage from "./requirements-page"

describe("RequirementsPage", () => {
  afterEach(() => {
    document.body.innerHTML = ""
    refetchMock.mockReset()
    requirementsResponseRef.current = REQUIREMENTS_RESPONSE
  })

  it("wires the refresh action to the requirements list refetch", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)

    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<RequirementsPage projectId="project-1" />)
    })

    const refreshButton = container.querySelector('[data-testid="refresh-requirements"]') as HTMLButtonElement | null

    expect(refreshButton).not.toBeNull()

    await act(async () => {
      refreshButton?.click()
    })

    expect(refetchMock).toHaveBeenCalledTimes(1)

    await act(async () => {
      root.unmount()
    })
  })

  it("shows a create action in the empty state when no requirements exist", async () => {
    requirementsResponseRef.current = EMPTY_REQUIREMENTS_RESPONSE

    const container = document.createElement("div")
    document.body.appendChild(container)

    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<RequirementsPage projectId="project-1" />)
    })

    const createButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent === "New Requirement"
    ) as HTMLButtonElement | undefined

    expect(createButton).toBeDefined()
    expect(container.querySelector('[data-testid="create-requirement-dialog"]')?.textContent).toBe("false")

    await act(async () => {
      createButton?.click()
    })

    expect(container.querySelector('[data-testid="create-requirement-dialog"]')?.textContent).toBe("true")

    await act(async () => {
      root.unmount()
    })
  })
})
