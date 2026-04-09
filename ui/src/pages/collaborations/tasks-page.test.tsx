import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, describe, expect, it, vi } from "vitest"

const { refetchMock, tasksResponseRef } = vi.hoisted(() => ({
  refetchMock: vi.fn(),
  tasksResponseRef: {
    current: undefined as typeof TASKS_RESPONSE | typeof EMPTY_TASKS_RESPONSE | undefined,
  },
}))

const TASKS_RESPONSE = {
  items: [
    {
      id: "task-1",
      project_id: "project-1",
      sprint_id: "sprint-1",
      requirement_id: "requirement-1",
      title: "Build refresh action",
      description: "Task description",
      status: "todo",
      priority: "p1",
      assignee_id: "user-1",
      due_date: "2026-03-20",
      estimate_hours: 4,
      parent_task_id: "",
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

const EMPTY_TASKS_RESPONSE = {
  items: [],
  pagination: {
    total: 0,
    page: 1,
    page_size: 20,
    total_pages: 0,
  },
} as const

tasksResponseRef.current = TASKS_RESPONSE

vi.mock("@tanstack/react-query", () => ({
  useQuery: ({ queryKey }: { queryKey: unknown[] }) => {
    if (queryKey[0] === "tasks") {
      return {
        data: tasksResponseRef.current,
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
        <button type="button" data-testid="refresh-tasks-list" onClick={onRefresh}>
          Refresh list
        </button>
      ) : (
        <div data-testid="missing-list-refresh" />
      )}
      <div data-testid="tasks-source-empty">{sourceEmptyContent}</div>
    </div>
  ),
}))

vi.mock("@/components/collaborations/kanban-board", () => ({
  KanbanBoard: () => <div>Kanban Board</div>,
}))

vi.mock("@/components/collaborations/collab-badges", () => ({
  DueDateBadge: () => null,
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

vi.mock("@/components/collaborations/task-dialogs", () => ({
  CreateTaskDialog: ({ open }: { open: boolean }) => <div data-testid="create-task-dialog">{String(open)}</div>,
  EditTaskDialog: () => null,
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

vi.mock("@/components/ui/tabs", () => ({
  Tabs: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  TabsList: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  TabsTrigger: ({ children }: { children: React.ReactNode }) => <button type="button">{children}</button>,
}))

import TasksPage from "./tasks-page"

describe("TasksPage", () => {
  afterEach(() => {
    document.body.innerHTML = ""
    refetchMock.mockReset()
    tasksResponseRef.current = TASKS_RESPONSE
  })

  it("wires the refresh action to the task list refetch in list view", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)

    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<TasksPage projectId="project-1" viewMode="list" />)
    })

    const refreshButton = container.querySelector('[data-testid="refresh-tasks-list"]') as HTMLButtonElement | null

    expect(refreshButton).not.toBeNull()

    await act(async () => {
      refreshButton?.click()
    })

    expect(refetchMock).toHaveBeenCalledTimes(1)

    await act(async () => {
      root.unmount()
    })
  })

  it("renders a refresh action in kanban view and triggers task refetch", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)

    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<TasksPage projectId="project-1" viewMode="kanban" />)
    })

    const refreshButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.getAttribute("aria-label") === "Refresh tasks"
    ) as HTMLButtonElement | undefined

    expect(refreshButton).toBeDefined()

    await act(async () => {
      refreshButton?.click()
    })

    expect(refetchMock).toHaveBeenCalledTimes(1)

    await act(async () => {
      root.unmount()
    })
  })

  it("shows a create action in the empty state when no tasks exist", async () => {
    tasksResponseRef.current = EMPTY_TASKS_RESPONSE

    const container = document.createElement("div")
    document.body.appendChild(container)

    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<TasksPage projectId="project-1" viewMode="list" />)
    })

    const createButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent === "New Task"
    ) as HTMLButtonElement | undefined

    expect(createButton).toBeDefined()
    expect(container.querySelector('[data-testid="create-task-dialog"]')?.textContent).toBe("false")

    await act(async () => {
      createButton?.click()
    })

    expect(container.querySelector('[data-testid="create-task-dialog"]')?.textContent).toBe("true")

    await act(async () => {
      root.unmount()
    })
  })
})
