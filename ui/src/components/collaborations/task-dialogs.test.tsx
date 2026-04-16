import { act, createContext, useContext, useEffect, useMemo, useState } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, describe, expect, it, vi } from "vitest"

const listSprintsMock = vi.fn()
const listMembersMock = vi.fn()

vi.mock("@/api/collaboration", () => ({
  collaborationApi: {
    listSprints: (...args: unknown[]) => listSprintsMock(...args),
    createTask: vi.fn(),
    createTaskChild: vi.fn(),
    updateTask: vi.fn(),
  },
  TaskStatus: {
    TODO: "todo",
    IN_PROGRESS: "in_progress",
    REVIEW: "review",
    DONE: "done",
    CANCELLED: "cancelled",
  },
  TaskStatusOptions: [
    { label: "Todo", value: "todo", color: "gray", icon: () => null },
    { label: "In Progress", value: "in_progress", color: "blue", icon: () => null },
    { label: "Review", value: "review", color: "orange", icon: () => null },
    { label: "Done", value: "done", color: "green", icon: () => null },
    { label: "Cancelled", value: "cancelled", color: "gray", icon: () => null },
  ],
  CollabPriority: {
    P0: "p0",
    P1: "p1",
    P2: "p2",
    P3: "p3",
  },
  CollabPriorityOptions: [
    { label: "P0", value: "p0", color: "red", icon: () => null },
    { label: "P1", value: "p1", color: "orange", icon: () => null },
    { label: "P2", value: "p2", color: "blue", icon: () => null },
    { label: "P3", value: "p3", color: "green", icon: () => null },
  ],
}))

vi.mock("@/api/projects", () => ({
  projectsApi: {
    listMembers: (...args: unknown[]) => listMembersMock(...args),
  },
}))

vi.mock("@/stores/auth", () => ({
  useAuthStore: (selector: (state: { user: { id: string } | null }) => unknown) => selector({
    user: { id: "user-1" },
  }),
}))

vi.mock("@tanstack/react-query", () => ({
  useMutation: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
  useQuery: ({ queryFn, enabled = true }: { queryFn: () => Promise<unknown> | unknown; enabled?: boolean }) => {
    const [data, setData] = useState<unknown>(undefined)

    useEffect(() => {
      if (!enabled) {
        setData(undefined)
        return
      }

      let active = true

      Promise.resolve(queryFn()).then((result) => {
        if (active) {
          setData(result)
        }
      })

      return () => {
        active = false
      }
    }, [enabled, queryFn])

    return { data }
  },
  useQueryClient: () => ({
    invalidateQueries: vi.fn(),
  }),
}))

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

vi.mock("@/components/ui/dialog", () => ({
  Dialog: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogContent: ({ children }: { children: React.ReactNode }) => <div data-testid="dialog-content">{children}</div>,
  DialogDescription: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogFooter: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogTitle: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, ...props }: React.ComponentProps<"button">) => <button type="button" {...props}>{children}</button>,
}))

vi.mock("@/components/ui/input", () => ({
  Input: (props: React.ComponentProps<"input">) => <input {...props} />,
}))

vi.mock("@/components/ui/textarea", () => ({
  Textarea: (props: React.ComponentProps<"textarea">) => <textarea {...props} />,
}))

vi.mock("@/components/ui/field", () => ({
  Field: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  FieldContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  FieldLabel: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/calendar", () => ({
  Calendar: () => <div>Calendar</div>,
}))

vi.mock("@/components/ui/popover", () => ({
  Popover: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  PopoverContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  PopoverTrigger: ({ render }: { render: React.ReactNode }) => <div>{render}</div>,
}))

vi.mock("./inline-editors", () => ({
  MemberAvatar: ({ name }: { name: string }) => <span>{name}</span>,
}))

type ComboboxContextValue = {
  display: string
}

const ComboboxContext = createContext<ComboboxContextValue>({ display: "" })

vi.mock("@/components/ui/combobox", () => ({
  Combobox: ({
    children,
    value,
    itemToStringLabel,
  }: {
    children: React.ReactNode
    value: string
    itemToStringLabel: (item: string) => string
  }) => {
    const display = useMemo(() => itemToStringLabel(value), [itemToStringLabel, value])
    return <ComboboxContext.Provider value={{ display }}>{children}</ComboboxContext.Provider>
  },
  ComboboxContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  ComboboxInput: (props: React.ComponentProps<"input">) => {
    const { display } = useContext(ComboboxContext)
    return <input {...props} readOnly value={display} />
  },
  ComboboxItem: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  ComboboxList: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

import { CreateTaskDialog } from "./task-dialogs"

describe("CreateTaskDialog", () => {
  afterEach(() => {
    document.body.innerHTML = ""
    listSprintsMock.mockReset()
    listMembersMock.mockReset()
  })

  it("shows the selected status as a plain text label", async () => {
    listSprintsMock.mockResolvedValue({
      items: [{ id: "sprint-1", name: "Sprint 1" }],
      pagination: { total: 1, page: 1, page_size: 100, total_pages: 1 },
    })
    listMembersMock.mockResolvedValue({
      items: [{ user_id: "user-1", fullname: "Alice", username: "alice" }],
      pagination: { total: 1, page: 1, page_size: 100, total_pages: 1 },
    })

    const container = document.createElement("div")
    document.body.appendChild(container)

    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(
        <CreateTaskDialog
          open
          onOpenChange={vi.fn()}
          projectId="project-1"
        />
      )
    })

    await act(async () => {
      await Promise.resolve()
    })

    const inputValues = Array.from(container.querySelectorAll("input"))
      .map((input) => (input as HTMLInputElement).value)

    expect(container.querySelector('[data-testid="dialog-content"]')).not.toBeNull()
    expect(inputValues).toContain("Todo")
    expect(inputValues).not.toContain("[object Object]")

    await act(async () => {
      root.unmount()
    })
  })
})
