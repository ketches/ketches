import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const { mockInvalidateQueries, mockOnOpenChange, mockToastSuccess, mockToastError } = vi.hoisted(() => ({
  mockInvalidateQueries: vi.fn(),
  mockOnOpenChange: vi.fn(),
  mockToastSuccess: vi.fn(),
  mockToastError: vi.fn(),
}))

vi.mock("@tanstack/react-query", () => ({
  useMutation: (options: {
    onSuccess?: (project: { id: string; name: string; slug: string }) => void
  }) => ({
    mutate: vi.fn(() => {
      options.onSuccess?.({
        id: "project-2",
        name: "New Project",
        slug: "new-project",
      })
    }),
    isPending: false,
  }),
  useQueryClient: () => ({
    invalidateQueries: mockInvalidateQueries,
  }),
}))

vi.mock("@/api/projects", () => ({
  projectsApi: {
    create: vi.fn(),
  },
}))

vi.mock("sonner", () => ({
  toast: {
    success: mockToastSuccess,
    error: mockToastError,
  },
}))

vi.mock("@/components/ui/dialog", () => ({
  Dialog: ({ open, children }: { open: boolean; children: React.ReactNode }) => open ? <div>{children}</div> : null,
  DialogContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogDescription: ({ children }: { children: React.ReactNode }) => <p>{children}</p>,
  DialogFooter: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogTitle: ({ children }: { children: React.ReactNode }) => <h1>{children}</h1>,
}))

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, type, ...props }: React.ComponentProps<"button">) => <button type={type ?? "button"} {...props}>{children}</button>,
}))

vi.mock("@/components/ui/checkbox", () => ({
  Checkbox: ({ checked, onCheckedChange, ...props }: { checked?: boolean; onCheckedChange?: (value: boolean) => void } & React.ComponentProps<"button">) => (
    <button
      type="button"
      aria-pressed={checked}
      onClick={() => onCheckedChange?.(!checked)}
      {...props}
    />
  ),
}))

vi.mock("@/components/ui/field", () => ({
  Field: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  FieldContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  FieldError: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  FieldLabel: ({ children, htmlFor }: { children: React.ReactNode; htmlFor?: string }) => <label htmlFor={htmlFor}>{children}</label>,
}))

vi.mock("@/components/ui/input", () => ({
  Input: (props: React.ComponentProps<"input">) => <input {...props} />,
}))

vi.mock("@/components/ui/textarea", () => ({
  Textarea: (props: React.ComponentProps<"textarea">) => <textarea {...props} />,
}))

import { CreateProjectDialog } from "./create-project-dialog"

const changeInputValue = async (input: HTMLInputElement, value: string) => {
  await act(async () => {
    const valueSetter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")?.set
    valueSetter?.call(input, value)
    input.dispatchEvent(new Event("input", { bubbles: true }))
    input.dispatchEvent(new Event("change", { bubbles: true }))
  })
}

describe("CreateProjectDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    ;(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true
  })

  afterEach(() => {
    document.body.innerHTML = ""
  })

  it("invalidates the switcher project list after a successful create", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(
        <CreateProjectDialog
          open
          onOpenChange={mockOnOpenChange}
        />
      )
    })

    const nameInput = container.querySelector('input[placeholder="My Project"]') as HTMLInputElement | null

    expect(nameInput).not.toBeNull()

    await changeInputValue(nameInput as HTMLInputElement, "New Project")

    const form = container.querySelector("form")
    expect(form).not.toBeNull()

    await act(async () => {
      form?.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
    })

    expect(mockInvalidateQueries).toHaveBeenCalledWith({ queryKey: ["projects"] })
    expect(mockInvalidateQueries).toHaveBeenCalledWith({ queryKey: ["projects-simple"] })

    await act(async () => {
      root.unmount()
    })
  })
})
