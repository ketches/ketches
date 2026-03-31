import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const { mockMutate, mockInvalidateQueries, mockOnOpenChange, mockOnClose, mockToastError } = vi.hoisted(() => ({
  mockMutate: vi.fn(),
  mockInvalidateQueries: vi.fn(),
  mockOnOpenChange: vi.fn(),
  mockOnClose: vi.fn(),
  mockToastError: vi.fn(),
}))

vi.mock("@tanstack/react-query", () => ({
  useMutation: () => ({
    mutate: mockMutate,
    isPending: false,
  }),
  useQueryClient: () => ({
    invalidateQueries: mockInvalidateQueries,
  }),
}))

vi.mock("@/stores/project", () => ({
  useProjectStore: () => ({
    activeEnvId: "env-1",
  }),
}))

vi.mock("@/api/apps", () => ({
  appsApi: {
    create: vi.fn(),
  },
}))

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: mockToastError,
  },
}))

vi.mock("@/components/ui/dialog", () => ({
  Dialog: ({
    open,
    children,
  }: {
    open: boolean
    children: React.ReactNode
  }) => open ? <div>{children}</div> : null,
  DialogContent: ({
    children,
    className,
  }: {
    children: React.ReactNode
    className?: string
  }) => <div data-testid="dialog-content" className={className}>{children}</div>,
  DialogDescription: ({ children }: { children: React.ReactNode }) => <p>{children}</p>,
  DialogFooter: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogTitle: ({ children }: { children: React.ReactNode }) => <h1>{children}</h1>,
}))

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, type, ...props }: React.ComponentProps<"button">) => <button type={type ?? "button"} {...props}>{children}</button>,
}))

vi.mock("@/components/ui/input", () => ({
  Input: (props: React.ComponentProps<"input">) => <input {...props} />,
}))

vi.mock("@/components/ui/textarea", () => ({
  Textarea: (props: React.ComponentProps<"textarea">) => <textarea {...props} />,
}))

vi.mock("@/components/ui/checkbox", () => ({
  Checkbox: ({
    checked,
    onCheckedChange,
    ...props
  }: {
    checked?: boolean
    onCheckedChange?: (checked: boolean) => void
  } & React.ComponentProps<"input">) => (
    <input
      {...props}
      checked={checked}
      type="checkbox"
      onChange={(event) => onCheckedChange?.(event.target.checked)}
    />
  ),
}))

vi.mock("@/components/ui/field", () => ({
  Field: ({
    children,
    className,
  }: {
    children: React.ReactNode
    className?: string
  }) => <div className={className}>{children}</div>,
  FieldContent: ({
    children,
    className,
  }: {
    children: React.ReactNode
    className?: string
  }) => <div className={className}>{children}</div>,
  FieldError: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  FieldLabel: ({
    children,
    htmlFor,
    className,
  }: {
    children: React.ReactNode
    htmlFor?: string
    className?: string
  }) => <label htmlFor={htmlFor} className={className}>{children}</label>,
}))

import { CreateAppDialog } from "./create-app-dialog"

async function renderDialog() {
  const container = document.createElement("div")
  document.body.appendChild(container)
  const root = ReactDOMClient.createRoot(container)

  await act(async () => {
    root.render(
      <CreateAppDialog
        open
        onOpenChange={mockOnOpenChange}
        onClose={mockOnClose}
      />
    )
  })

  return { container, root }
}

const changeInputValue = async (input: HTMLInputElement, value: string) => {
  await act(async () => {
    const valueSetter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")?.set
    valueSetter?.call(input, value)
    input.dispatchEvent(new Event("input", { bubbles: true }))
    input.dispatchEvent(new Event("change", { bubbles: true }))
  })
}

const clickElement = async (element: HTMLElement) => {
  await act(async () => {
    element.dispatchEvent(new MouseEvent("click", { bubbles: true }))
  })
}

describe("CreateAppDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // Silence React's act() environment warning in these DOM-only tests.
    ;(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true
  })

  afterEach(() => {
    document.body.innerHTML = ""
  })

  it("shows the image field first and keeps registry credentials collapsed by default", async () => {
    const { container, root } = await renderDialog()

    const textInputs = Array.from(
      container.querySelectorAll('input:not([type]), input[type="text"], input[type="password"]')
    ) as HTMLInputElement[]

    expect(textInputs[0]?.placeholder).toMatch(/enter or paste an image/i)
    expect(container.textContent).not.toContain("Registry Username")
    expect(container.textContent).not.toContain("Registry Password")

    await act(async () => {
      root.unmount()
    })
  })

  it("reveals registry credential fields when the key button is clicked", async () => {
    const { container, root } = await renderDialog()

    const toggle = container.querySelector('button[aria-label="Registry credentials"]') as HTMLButtonElement | null

    expect(toggle).not.toBeNull()

    await clickElement(toggle as HTMLButtonElement)

    expect(container.textContent).toContain("Registry Username")
    expect(container.textContent).toContain("Registry Password")

    await act(async () => {
      root.unmount()
    })
  })

  it("derives name, slug, and stateful app type from the image", async () => {
    const { container, root } = await renderDialog()

    const imageInput = container.querySelector('input[placeholder*="image URL"]') as HTMLInputElement | null

    expect(imageInput).not.toBeNull()

    await changeInputValue(imageInput as HTMLInputElement, "bitnami/mysql:8.0")

    expect((container.querySelector('input[name="name"]') as HTMLInputElement | null)?.value).toBe("Mysql")
    expect((container.querySelector('input[name="slug"]') as HTMLInputElement | null)?.value).toBe("mysql")
    expect(container.querySelector('button[data-app-type="StatefulSet"]')?.className).toContain("border-primary")

    await act(async () => {
      root.unmount()
    })
  })

  it("keeps slug synced to manual name edits until slug is edited directly", async () => {
    const { container, root } = await renderDialog()

    const imageInput = container.querySelector('input[placeholder*="image URL"]') as HTMLInputElement | null
    const nameInput = container.querySelector('input[name="name"]') as HTMLInputElement | null

    expect(imageInput).not.toBeNull()
    expect(nameInput).not.toBeNull()

    await changeInputValue(imageInput as HTMLInputElement, "nginx:latest")
    await changeInputValue(nameInput as HTMLInputElement, "My API Service")

    expect((container.querySelector('input[name="name"]') as HTMLInputElement | null)?.value).toBe("My API Service")
    expect((container.querySelector('input[name="slug"]') as HTMLInputElement | null)?.value).toBe("my-api-service")

    await act(async () => {
      root.unmount()
    })
  })

  it("does not override user-edited name, slug, or app type on later image changes", async () => {
    const { container, root } = await renderDialog()

    const imageInput = container.querySelector('input[placeholder*="image URL"]') as HTMLInputElement | null
    const nameInput = container.querySelector('input[name="name"]') as HTMLInputElement | null
    const slugInput = container.querySelector('input[name="slug"]') as HTMLInputElement | null
    const deploymentButton = container.querySelector('button[data-app-type="Deployment"]') as HTMLButtonElement | null

    expect(imageInput).not.toBeNull()
    expect(nameInput).not.toBeNull()
    expect(slugInput).not.toBeNull()
    expect(deploymentButton).not.toBeNull()

    await changeInputValue(imageInput as HTMLInputElement, "bitnami/mysql:8.0")
    await changeInputValue(nameInput as HTMLInputElement, "Custom Name")
    await changeInputValue(slugInput as HTMLInputElement, "custom-slug")
    await clickElement(deploymentButton as HTMLButtonElement)
    await changeInputValue(imageInput as HTMLInputElement, "redis:7")

    expect((container.querySelector('input[name="name"]') as HTMLInputElement | null)?.value).toBe("Custom Name")
    expect((container.querySelector('input[name="slug"]') as HTMLInputElement | null)?.value).toBe("custom-slug")
    expect(container.querySelector('button[data-app-type="Deployment"]')?.className).toContain("border-primary")
    expect(container.querySelector('button[data-app-type="StatefulSet"]')?.className ?? "").not.toContain("border-primary")

    await act(async () => {
      root.unmount()
    })
  })

  it("prevents submit when password is provided without a registry username", async () => {
    const { container, root } = await renderDialog()

    const toggle = container.querySelector('button[aria-label="Registry credentials"]') as HTMLButtonElement | null
    const imageInput = container.querySelector('input[placeholder*="image URL"]') as HTMLInputElement | null

    expect(toggle).not.toBeNull()
    expect(imageInput).not.toBeNull()

    await clickElement(toggle as HTMLButtonElement)
    await changeInputValue(imageInput as HTMLInputElement, "nginx:latest")

    const passwordInput = container.querySelector('input[name="registry_password"]') as HTMLInputElement | null

    expect(passwordInput).not.toBeNull()

    await changeInputValue(passwordInput as HTMLInputElement, "secret")
    await clickElement(container.querySelector('button[type="submit"]') as HTMLButtonElement)

    expect(mockMutate).not.toHaveBeenCalled()
    expect(mockToastError).toHaveBeenCalledWith("Error", {
      description: "Registry username is required when password is provided",
    })

    await act(async () => {
      root.unmount()
    })
  })
})
