import * as React from "react"
import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const {
  mockUsePlatformBranding,
  mockUseUpdatePlatformBrandingMutation,
  mockMutate,
  mockToastError,
  mockToastSuccess,
} = vi.hoisted(() => ({
  mockUsePlatformBranding: vi.fn(),
  mockUseUpdatePlatformBrandingMutation: vi.fn(),
  mockMutate: vi.fn(),
  mockToastError: vi.fn(),
  mockToastSuccess: vi.fn(),
}))

vi.mock("@/hooks/use-platform-settings", () => ({
  usePlatformBranding: () => mockUsePlatformBranding(),
  useUpdatePlatformBrandingMutation: () => mockUseUpdatePlatformBrandingMutation(),
}))

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, ...props }: React.ComponentProps<"button">) => <button {...props}>{children}</button>,
}))

vi.mock("@/components/ui/card", () => ({
  Card: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  CardAction: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  CardContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  CardDescription: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  CardHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  CardTitle: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/dialog", () => ({
  Dialog: ({ open, children }: { open: boolean; children: React.ReactNode }) => open ? <div>{children}</div> : null,
  DialogContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogDescription: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogFooter: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogTitle: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/field", () => ({
  Field: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  FieldLabel: ({ children, htmlFor }: { children: React.ReactNode; htmlFor?: string }) => <label htmlFor={htmlFor}>{children}</label>,
  FieldContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  FieldDescription: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/input", () => ({
  Input: (props: React.ComponentProps<"input">) => <input {...props} />,
}))

vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  TooltipContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  TooltipTrigger: ({ children, render }: { children: React.ReactNode; render?: React.ReactElement }) =>
    render ? React.cloneElement(render, {}, children) : <button type="button">{children}</button>,
}))

vi.mock("sonner", () => ({
  toast: {
    error: (...args: unknown[]) => mockToastError(...args),
    success: (...args: unknown[]) => mockToastSuccess(...args),
  },
}))

import { PlatformBrandingTab } from "./platform-branding-tab"

function setInputValue(input: HTMLInputElement, value: string) {
  const descriptor = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")
  descriptor?.set?.call(input, value)
  input.dispatchEvent(new Event("input", { bubbles: true }))
  input.dispatchEvent(new Event("change", { bubbles: true }))
}

describe("PlatformBrandingTab", () => {
  beforeEach(() => {
    mockUsePlatformBranding.mockReturnValue({
      data: {
        name: "Acme Control Plane",
      },
    })
    mockUseUpdatePlatformBrandingMutation.mockReturnValue({
      isPending: false,
      mutate: mockMutate,
    })
  })

  afterEach(() => {
    document.body.innerHTML = ""
    vi.clearAllMocks()
  })

  it("opens the dialog with the current branding name and saves the updated value", async () => {
    mockMutate.mockImplementation(
      (_value: { name: string }, options?: { onSuccess?: () => void }) => {
        options?.onSuccess?.()
      },
    )

    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<PlatformBrandingTab />)
    })

    const editButton = container.querySelector('button[aria-label="Edit branding name"]')
    expect(editButton).not.toBeNull()

    await act(async () => {
      editButton?.dispatchEvent(new MouseEvent("click", { bubbles: true }))
    })

    const input = container.querySelector('input[name="platform-name"]') as HTMLInputElement | null
    expect(input?.value).toBe("Acme Control Plane")

    const saveButton = Array.from(container.querySelectorAll("button")).find((button) => button.textContent?.includes("Save"))

    await act(async () => {
      if (input) {
        setInputValue(input, "Acme Console")
      }
      saveButton?.dispatchEvent(new MouseEvent("click", { bubbles: true }))
    })

    expect(mockMutate).toHaveBeenCalledWith({ name: "Acme Console" }, expect.any(Object))
    expect(mockToastSuccess).toHaveBeenCalled()
    expect(container.querySelector('input[name="platform-name"]')).toBeNull()

    await act(async () => {
      root.unmount()
    })
  })
})
