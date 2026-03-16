import { act, createContext, useContext, useMemo } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, describe, expect, it, vi } from "vitest"

vi.mock("@/api/collaboration", () => ({
  collaborationApi: {
    listSprints: vi.fn().mockResolvedValue({
      items: [{ id: "sprint-1", name: "Sprint 1" }],
      pagination: { total: 1, page: 1, page_size: 100, total_pages: 1 },
    }),
    createTestCase: vi.fn(),
    updateTestCase: vi.fn(),
    createTestRun: vi.fn(),
  },
  TestRunStatus: {
    PASSED: "passed",
    FAILED: "failed",
    BLOCKED: "blocked",
  },
  TestRunStatusOptions: [
    { label: "Passed", value: "passed" },
    { label: "Failed", value: "failed" },
    { label: "Blocked", value: "blocked" },
  ],
}))

vi.mock("@tanstack/react-query", () => ({
  useMutation: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
  useQuery: () => ({
    data: {
      items: [{ id: "sprint-1", name: "Sprint 1" }],
    },
  }),
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

const ComboboxContext = createContext<{ display: string }>({ display: "" })

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

import { CreateTestCaseDialog } from "./test-case-dialogs"

describe("CreateTestCaseDialog", () => {
  afterEach(() => {
    document.body.innerHTML = ""
  })

  it("renders the plain-text fields inside dialog content", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(
        <CreateTestCaseDialog
          open
          onOpenChange={() => undefined}
          projectId="project-1"
        />
      )
    })

    expect(container.querySelector('[data-testid="dialog-content"]')).not.toBeNull()
    expect(container.querySelectorAll("textarea")).toHaveLength(3)

    await act(async () => {
      root.unmount()
    })
  })
})
