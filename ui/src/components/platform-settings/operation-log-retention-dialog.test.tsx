import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const {
  mockInvalidateQueries,
  mockMutate,
  mockUseMutation,
  mockUseQueryClient,
  mockToastError,
  mockToastSuccess,
} = vi.hoisted(() => ({
  mockInvalidateQueries: vi.fn(),
  mockMutate: vi.fn(),
  mockUseMutation: vi.fn(),
  mockUseQueryClient: vi.fn(),
  mockToastError: vi.fn(),
  mockToastSuccess: vi.fn(),
}))

vi.mock("@tanstack/react-query", async () => {
  const actual = await vi.importActual<typeof import("@tanstack/react-query")>("@tanstack/react-query")
  return {
    ...actual,
    useMutation: (...args: unknown[]) => mockUseMutation(...args),
    useQueryClient: (...args: unknown[]) => mockUseQueryClient(...args),
  }
})

vi.mock("@/components/ui/dialog", () => ({
  Dialog: ({ open, children }: { open: boolean; children: React.ReactNode }) => open ? <div>{children}</div> : null,
  DialogContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogTitle: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogDescription: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogFooter: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, ...props }: React.ComponentProps<"button">) => <button {...props}>{children}</button>,
}))

vi.mock("@/components/ui/input", () => ({
  Input: (props: React.ComponentProps<"input">) => <input {...props} />,
}))

vi.mock("@/components/ui/field", () => ({
  Field: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  FieldLabel: ({ children, htmlFor }: { children: React.ReactNode; htmlFor?: string }) => <label htmlFor={htmlFor}>{children}</label>,
  FieldContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  FieldDescription: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("sonner", () => ({
  toast: {
    error: (...args: unknown[]) => mockToastError(...args),
    success: (...args: unknown[]) => mockToastSuccess(...args),
  },
}))

import { OperationLogRetentionDialog } from "./operation-log-retention-dialog"

function setInputValue(input: HTMLInputElement, value: string) {
  const descriptor = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")
  descriptor?.set?.call(input, value)
  input.dispatchEvent(new Event("input", { bubbles: true }))
  input.dispatchEvent(new Event("change", { bubbles: true }))
}

describe("OperationLogRetentionDialog", () => {
  beforeEach(() => {
    mockUseQueryClient.mockReturnValue({
      invalidateQueries: mockInvalidateQueries,
    })
    mockUseMutation.mockReturnValue({
      isPending: false,
      mutate: mockMutate,
    })
  })

  afterEach(() => {
    document.body.innerHTML = ""
    vi.clearAllMocks()
  })

  it("prefills the current retention days", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(
        <OperationLogRetentionDialog
          open
          onOpenChange={() => undefined}
          retentionDays={90}
        />
      )
    })

    expect((container.querySelector('input[name="retention-days"]') as HTMLInputElement | null)?.value).toBe("90")

    await act(async () => {
      root.unmount()
    })
  })

  it("blocks invalid values before submit", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(
        <OperationLogRetentionDialog
          open
          onOpenChange={() => undefined}
          retentionDays={90}
        />
      )
    })

    const input = container.querySelector('input[name="retention-days"]') as HTMLInputElement | null
    const saveButton = Array.from(container.querySelectorAll("button")).find((button) => button.textContent?.includes("Save")) as HTMLButtonElement | undefined

    await act(async () => {
      if (input) {
        setInputValue(input, "0")
      }
      saveButton?.click()
    })

    expect(mockMutate).not.toHaveBeenCalled()
    expect(mockToastError).toHaveBeenCalled()

    await act(async () => {
      root.unmount()
    })
  })

  it("saves the updated value, invalidates settings, and closes the dialog", async () => {
    const onOpenChange = vi.fn()
    mockMutate.mockImplementation((_value: number, options?: { onSuccess?: () => void }) => {
      options?.onSuccess?.()
    })

    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(
        <OperationLogRetentionDialog
          open
          onOpenChange={onOpenChange}
          retentionDays={90}
        />
      )
    })

    const input = container.querySelector('input[name="retention-days"]') as HTMLInputElement | null
    const saveButton = Array.from(container.querySelectorAll("button")).find((button) => button.textContent?.includes("Save")) as HTMLButtonElement | undefined

    await act(async () => {
      if (input) {
        setInputValue(input, "120")
      }
      saveButton?.click()
    })

    expect(mockMutate).toHaveBeenCalledWith(120, expect.any(Object))
    expect(mockInvalidateQueries).toHaveBeenCalledWith({ queryKey: ["operation-log-settings"] })
    expect(onOpenChange).toHaveBeenCalledWith(false)
    expect(mockToastSuccess).toHaveBeenCalled()

    await act(async () => {
      root.unmount()
    })
  })
})
