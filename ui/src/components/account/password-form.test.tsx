import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, type, ...props }: React.ComponentProps<"button">) => <button type={type ?? "button"} {...props}>{children}</button>,
}))

vi.mock("@/components/ui/input", () => ({
  Input: (props: React.ComponentProps<"input">) => <input {...props} />,
}))

vi.mock("@/components/ui/field", () => ({
  Field: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  FieldContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  FieldGroup: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  FieldLabel: ({ children, htmlFor }: { children: React.ReactNode; htmlFor?: string }) => <label htmlFor={htmlFor}>{children}</label>,
}))

import { PasswordForm } from "./password-form"

function setInputValue(input: HTMLInputElement, value: string) {
  const descriptor = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")
  descriptor?.set?.call(input, value)
  input.dispatchEvent(new Event("input", { bubbles: true }))
  input.dispatchEvent(new Event("change", { bubbles: true }))
}

describe("PasswordForm", () => {
  beforeEach(() => {
    ;(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true
  })

  afterEach(() => {
    document.body.innerHTML = ""
    vi.clearAllMocks()
  })

  it("submits the real password payload after validation", async () => {
    const onSave = vi.fn()
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<PasswordForm onSave={onSave} />)
    })

    const currentPasswordInput = container.querySelector("#currentPassword") as HTMLInputElement | null
    const newPasswordInput = container.querySelector("#newPassword") as HTMLInputElement | null
    const confirmPasswordInput = container.querySelector("#confirmPassword") as HTMLInputElement | null
    const submitButton = Array.from(container.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("Update Password")
    )

    await act(async () => {
      if (currentPasswordInput) {
        setInputValue(currentPasswordInput, "secret123")
      }
      if (newPasswordInput) {
        setInputValue(newPasswordInput, "new-secret123")
      }
      if (confirmPasswordInput) {
        setInputValue(confirmPasswordInput, "new-secret123")
      }
    })

    await act(async () => {
      submitButton?.dispatchEvent(new MouseEvent("click", { bubbles: true }))
    })

    expect(onSave).toHaveBeenCalledWith({
      currentPassword: "secret123",
      newPassword: "new-secret123",
    })

    await act(async () => {
      root.unmount()
    })
  })
})
