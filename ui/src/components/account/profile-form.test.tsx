import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, type, ...props }: React.ComponentProps<"button">) => <button type={type ?? "button"} {...props}>{children}</button>,
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
  FieldGroup: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  FieldLabel: ({ children, htmlFor }: { children: React.ReactNode; htmlFor?: string }) => <label htmlFor={htmlFor}>{children}</label>,
}))

import { ProfileForm } from "./profile-form"

function setInputValue(input: HTMLInputElement, value: string) {
  const descriptor = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")
  descriptor?.set?.call(input, value)
  input.dispatchEvent(new Event("input", { bubbles: true }))
  input.dispatchEvent(new Event("change", { bubbles: true }))
}

function setTextareaValue(textarea: HTMLTextAreaElement, value: string) {
  const descriptor = Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, "value")
  descriptor?.set?.call(textarea, value)
  textarea.dispatchEvent(new Event("input", { bubbles: true }))
  textarea.dispatchEvent(new Event("change", { bubbles: true }))
}

describe("ProfileForm", () => {
  beforeEach(() => {
    ;(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true
  })

  afterEach(() => {
    document.body.innerHTML = ""
    vi.clearAllMocks()
  })

  it("renders Full Name and submits fullname email and bio", async () => {
    const onSave = vi.fn()
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(
        <ProfileForm
          user={{
            fullname: "Alice",
            email: "alice@example.com",
            bio: "Initial bio",
            avatar: "",
          }}
          onSave={onSave}
        />
      )
    })

    expect(container.textContent).toContain("Full Name")

    const fullnameInput = container.querySelector("#fullname") as HTMLInputElement | null
    const emailInput = container.querySelector("#email") as HTMLInputElement | null
    const bioInput = container.querySelector("#bio") as HTMLTextAreaElement | null
    const submitButton = Array.from(container.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("Save Changes")
    )

    expect(fullnameInput).not.toBeNull()
    expect(emailInput).not.toBeNull()
    expect(bioInput).not.toBeNull()

    await act(async () => {
      if (fullnameInput) {
        setInputValue(fullnameInput, "Alice Example")
      }
      if (emailInput) {
        setInputValue(emailInput, "alice+new@example.com")
      }
      if (bioInput) {
        setTextareaValue(bioInput, "Updated bio")
      }
    })

    await act(async () => {
      submitButton?.dispatchEvent(new MouseEvent("click", { bubbles: true }))
    })

    expect(onSave).toHaveBeenCalledWith({
      fullname: "Alice Example",
      email: "alice+new@example.com",
      bio: "Updated bio",
    })

    await act(async () => {
      root.unmount()
    })
  })
})
