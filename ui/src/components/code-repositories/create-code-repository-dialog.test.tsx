import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const { mockMutate, mockInvalidateQueries, mockOnOpenChange } = vi.hoisted(() => ({
  mockMutate: vi.fn(),
  mockInvalidateQueries: vi.fn(),
  mockOnOpenChange: vi.fn(),
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

vi.mock("@/api/code-repositories", () => ({
  codeRepositoriesApi: {
    create: vi.fn(),
  },
}))

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
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

vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  TooltipTrigger: ({
    render,
    children,
  }: {
    render?: React.ReactNode
    children?: React.ReactNode
  }) => <>{render ?? children ?? null}</>,
  TooltipContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

import { CreateCodeRepositoryDialog } from "./create-code-repository-dialog"

async function renderDialog() {
  const container = document.createElement("div")
  document.body.appendChild(container)
  const root = ReactDOMClient.createRoot(container)

  await act(async () => {
    root.render(
      <CreateCodeRepositoryDialog
        open
        onOpenChange={mockOnOpenChange}
        projectId="project-1"
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

describe("CreateCodeRepositoryDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    ;(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true
  })

  afterEach(() => {
    document.body.innerHTML = ""
  })

  it("shows the Git Repository URL first and hides credentials by default", async () => {
    const { container, root } = await renderDialog()

    const textInputs = Array.from(
      container.querySelectorAll('input:not([type]), input[type="text"], input[type="password"]')
    ) as HTMLInputElement[]

    expect(textInputs[0]?.placeholder).toContain("github.com")
    expect(container.textContent).not.toContain("Git Username")
    expect(container.textContent).not.toContain("Git Password / Token")

    await act(async () => {
      root.unmount()
    })
  })

  it("reveals Git credentials when the key button is clicked", async () => {
    const { container, root } = await renderDialog()

    const toggle = container.querySelector('button[aria-label="Git credentials"]') as HTMLButtonElement | null

    expect(toggle).not.toBeNull()

    await clickElement(toggle as HTMLButtonElement)

    expect(container.textContent).toContain("Git Username")
    expect(container.textContent).toContain("Git Password / Token")

    await act(async () => {
      root.unmount()
    })
  })

  it("derives name and slug from the repository URL", async () => {
    const { container, root } = await renderDialog()

    const urlInput = container.querySelector('input[name="git_repo_url"]') as HTMLInputElement | null

    expect(urlInput).not.toBeNull()

    await changeInputValue(urlInput as HTMLInputElement, "https://github.com/acme/my-api.git")

    expect((container.querySelector('input[name="name"]') as HTMLInputElement | null)?.value).toBe("My Api")
    expect((container.querySelector('input[name="slug"]') as HTMLInputElement | null)?.value).toBe("my-api")

    await act(async () => {
      root.unmount()
    })
  })

  it("keeps slug synced to manual name edits until slug is edited directly", async () => {
    const { container, root } = await renderDialog()

    const urlInput = container.querySelector('input[name="git_repo_url"]') as HTMLInputElement | null
    const nameInput = container.querySelector('input[name="name"]') as HTMLInputElement | null

    expect(urlInput).not.toBeNull()
    expect(nameInput).not.toBeNull()

    await changeInputValue(urlInput as HTMLInputElement, "https://github.com/acme/my-api.git")
    await changeInputValue(nameInput as HTMLInputElement, "Custom Backend Repo")

    expect((container.querySelector('input[name="slug"]') as HTMLInputElement | null)?.value).toBe("custom-backend-repo")

    await act(async () => {
      root.unmount()
    })
  })

  it("does not override manually edited slug on later name or URL changes", async () => {
    const { container, root } = await renderDialog()

    const urlInput = container.querySelector('input[name="git_repo_url"]') as HTMLInputElement | null
    const nameInput = container.querySelector('input[name="name"]') as HTMLInputElement | null
    const slugInput = container.querySelector('input[name="slug"]') as HTMLInputElement | null

    expect(urlInput).not.toBeNull()
    expect(nameInput).not.toBeNull()
    expect(slugInput).not.toBeNull()

    await changeInputValue(urlInput as HTMLInputElement, "https://github.com/acme/my-api.git")
    await changeInputValue(slugInput as HTMLInputElement, "backend-core")
    await changeInputValue(nameInput as HTMLInputElement, "Renamed Repo")
    await changeInputValue(urlInput as HTMLInputElement, "https://github.com/acme/renamed.git")

    expect((container.querySelector('input[name="name"]') as HTMLInputElement | null)?.value).toBe("Renamed Repo")
    expect((container.querySelector('input[name="slug"]') as HTMLInputElement | null)?.value).toBe("backend-core")

    await act(async () => {
      root.unmount()
    })
  })
})
