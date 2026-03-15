import { act, createContext, useContext, useEffect, useMemo, useState } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, describe, expect, it, vi } from "vitest"

const listRequirementsMock = vi.fn()
const listTasksMock = vi.fn()
const listTestCasesMock = vi.fn()
const listSprintsMock = vi.fn()

vi.mock("@/api/collaboration", () => ({
  collaborationApi: {
    listRequirements: (...args: unknown[]) => listRequirementsMock(...args),
    listTasks: (...args: unknown[]) => listTasksMock(...args),
    listTestCases: (...args: unknown[]) => listTestCasesMock(...args),
    listSprints: (...args: unknown[]) => listSprintsMock(...args),
    updateDefect: vi.fn(),
    createDefect: vi.fn(),
  },
  DefectSeverity: {
    CRITICAL: "critical",
    HIGH: "high",
    MEDIUM: "medium",
    LOW: "low",
  },
  DefectSeverityOptions: [
    { label: "Critical", value: "critical" },
    { label: "High", value: "high" },
    { label: "Medium", value: "medium" },
    { label: "Low", value: "low" },
  ],
  DefectStatus: {
    NEW: "new",
    PROCESSING: "processing",
    PENDING_VERIFY: "pending_verify",
    CLOSED: "closed",
    REJECTED: "rejected",
  },
  DefectStatusOptions: [
    { label: "New", value: "new" },
    { label: "Processing", value: "processing" },
    { label: "Pending Verify", value: "pending_verify" },
    { label: "Closed", value: "closed" },
    { label: "Rejected", value: "rejected" },
  ],
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

vi.mock("@/components/ui/sheet", () => ({
  Sheet: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  SheetContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  SheetDescription: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  SheetFooter: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  SheetHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  SheetTitle: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/dialog", () => ({
  Dialog: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
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

vi.mock("@/components/editor/basic-editor", () => ({
  BasicEditor: ({ value }: { value: string }) => <div>{value}</div>,
}))

vi.mock("@/components/editor/basic-editor-value", () => ({
  isBasicEditorEmpty: () => false,
}))

vi.mock("@/components/ui/field", () => ({
  Field: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  FieldContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  FieldLabel: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
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
    const display = useMemo(() => itemToStringLabel(value), [])
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

import { EditDefectDialog } from "./defect-dialogs"

describe("EditDefectDialog", () => {
  afterEach(() => {
    document.body.innerHTML = ""
    listRequirementsMock.mockReset()
    listTasksMock.mockReset()
    listTestCasesMock.mockReset()
    listSprintsMock.mockReset()
  })

  it("backfills requirement, task, and test case comboboxes with titles instead of IDs", async () => {
    listRequirementsMock.mockResolvedValue({
      items: [{ id: "req-1", title: "Requirement Title" }],
      pagination: { total: 1, page: 1, page_size: 100, total_pages: 1 },
    })
    listTasksMock.mockResolvedValue({
      items: [{ id: "task-1", title: "Task Title" }],
      pagination: { total: 1, page: 1, page_size: 100, total_pages: 1 },
    })
    listTestCasesMock.mockResolvedValue({
      items: [{ id: "tc-1", title: "Test Case Title" }],
      pagination: { total: 1, page: 1, page_size: 100, total_pages: 1 },
    })
    listSprintsMock.mockResolvedValue({
      items: [{ id: "sprint-1", name: "Sprint 1" }],
      pagination: { total: 1, page: 1, page_size: 100, total_pages: 1 },
    })

    const container = document.createElement("div")
    document.body.appendChild(container)

    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(
        <EditDefectDialog
          open
          onOpenChange={vi.fn()}
          projectId="project-1"
          defect={{
            id: "defect-1",
            project_id: "project-1",
            sprint_id: "sprint-1",
            requirement_id: "req-1",
            task_id: "task-1",
            test_case_id: "tc-1",
            title: "Defect title",
            description: "Defect description",
            severity: "medium",
            status: "new",
            created_by: "user-1",
            updated_by: "user-1",
            created_at: "2026-03-01T00:00:00Z",
            updated_at: "2026-03-01T00:00:00Z",
          }}
        />
      )
    })

    await act(async () => {
      await Promise.resolve()
    })

    const inputValues = Array.from(container.querySelectorAll("input"))
      .map((input) => (input as HTMLInputElement).value)

    expect(inputValues).toContain("Requirement Title")
    expect(inputValues).toContain("Task Title")
    expect(inputValues).toContain("Test Case Title")
    expect(inputValues).not.toContain("req-1")
    expect(inputValues).not.toContain("task-1")
    expect(inputValues).not.toContain("tc-1")

    await act(async () => {
      root.unmount()
    })
  })
})
