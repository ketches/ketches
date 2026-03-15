import { act, useEffect, useState } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, describe, expect, it, vi } from "vitest"

const listMembersMock = vi.fn()

vi.mock("@/api/projects", () => ({
  projectsApi: {
    listMembers: (...args: unknown[]) => listMembersMock(...args),
  },
}))

vi.mock("@tanstack/react-query", () => ({
  useMutation: () => ({
    mutate: vi.fn(),
  }),
  useQuery: ({ queryFn }: { queryFn: () => Promise<unknown> | unknown }) => {
    const [data, setData] = useState<unknown>(undefined)

    useEffect(() => {
      let active = true

      Promise.resolve(queryFn()).then((result) => {
        if (active) {
          setData(result)
        }
      })

      return () => {
        active = false
      }
    }, [queryFn])

    return { data }
  },
  useQueryClient: () => ({
    invalidateQueries: vi.fn(),
  }),
}))

vi.mock("@dnd-kit/core", () => ({
  DndContext: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DragOverlay: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  PointerSensor: function PointerSensor() { },
  useDraggable: () => ({
    attributes: {},
    listeners: {},
    setNodeRef: vi.fn(),
    transform: null,
    isDragging: false,
  }),
  useDroppable: () => ({
    setNodeRef: vi.fn(),
    isOver: false,
  }),
  useSensor: () => ({}),
  useSensors: () => ([]),
}))

vi.mock("@/components/collaboration/collab-badges", () => ({
  DueDateBadge: () => null,
  PriorityBadge: () => null,
}))

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, ...props }: React.ComponentProps<"button">) => <button type="button" {...props}>{children}</button>,
}))

vi.mock("@/components/ui/card", () => ({
  Card: ({ children, ...props }: React.ComponentProps<"div">) => <div {...props}>{children}</div>,
}))

vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenu: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DropdownMenuContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DropdownMenuItem: ({ children, ...props }: React.ComponentProps<"button">) => <button type="button" {...props}>{children}</button>,
  DropdownMenuSeparator: () => <div />,
  DropdownMenuTrigger: ({ render }: { render?: React.ReactNode }) => <>{render ?? null}</>,
}))

vi.mock("sonner", () => ({
  toast: {
    error: vi.fn(),
  },
}))

import { KanbanBoard } from "./kanban-board"

describe("KanbanBoard", () => {
  afterEach(() => {
    document.body.innerHTML = ""
    listMembersMock.mockReset()
  })

  it("renders the assignee username instead of the raw user ID", async () => {
    listMembersMock.mockResolvedValue({
      items: [
        {
          user_id: "user-123",
          username: "alice",
          fullname: "",
          email: "alice@example.com",
          project_role: "developer",
          joined_at: "2026-03-01T00:00:00Z",
        },
      ],
      pagination: {
        total: 1,
        page: 1,
        page_size: 100,
        total_pages: 1,
      },
    })

    const container = document.createElement("div")
    document.body.appendChild(container)

    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(
        <KanbanBoard
          projectId="project-1"
          tasks={[
            {
              id: "task-456",
              project_id: "project-1",
              sprint_id: "sprint-1",
              requirement_id: "requirement-1",
              title: "Fix assignee name",
              description: "Render username in kanban cards",
              status: "todo",
              priority: "p1",
              assignee_id: "user-123",
              due_date: "2026-03-20",
              estimate_hours: 2,
              parent_task_id: "",
              depth: 0,
              created_by: "user-123",
              updated_by: "user-123",
              created_at: "2026-03-01T00:00:00Z",
              updated_at: "2026-03-01T00:00:00Z",
            },
          ]}
        />
      )
    })

    await act(async () => {
      await Promise.resolve()
    })

    expect(container.textContent).toContain("alice")
    expect(container.textContent).not.toContain("user-123")

    await act(async () => {
      root.unmount()
    })
  })
})
