import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const { mockInvalidateQueries, mockOnOpenChange } = vi.hoisted(() => ({
  mockInvalidateQueries: vi.fn(),
  mockOnOpenChange: vi.fn(),
}))

vi.mock("@tanstack/react-query", () => ({
  useMutation: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
  useQueryClient: () => ({
    invalidateQueries: mockInvalidateQueries,
  }),
}))

vi.mock("@/api/code-repositories", () => ({
  codeRepositoriesApi: {
    update: vi.fn(),
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
  DialogContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
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
  Field: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  FieldContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
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

import type { CodeRepository } from "@/api/code-repositories"
import { EditCodeRepositoryDialog } from "./edit-code-repository-dialog"

function buildRepo(overrides: Partial<CodeRepository> = {}): CodeRepository {
  return {
    id: "repo-1",
    project_id: "project-1",
    name: "Repo One",
    slug: "repo-one",
    git_repo_url: "https://github.com/acme/repo-one.git",
    git_username: "",
    git_password: "",
    has_git_password: false,
    created_at: "2026-03-16T00:00:00Z",
    updated_at: "2026-03-16T00:00:00Z",
    ...overrides,
  }
}

async function renderDialog(repo: CodeRepository) {
  const container = document.createElement("div")
  document.body.appendChild(container)
  const root = ReactDOMClient.createRoot(container)

  await act(async () => {
    root.render(
      <EditCodeRepositoryDialog
        open
        onOpenChange={mockOnOpenChange}
        repo={repo}
      />
    )
  })

  return { container, root }
}

const clickElement = async (element: HTMLElement) => {
  await act(async () => {
    element.dispatchEvent(new MouseEvent("click", { bubbles: true }))
  })
}

describe("EditCodeRepositoryDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    ;(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true
  })

  afterEach(() => {
    document.body.innerHTML = ""
  })

  it("shows credentials immediately when the repo already has credentials", async () => {
    const { container, root } = await renderDialog(buildRepo({
      git_username: "git-user",
      has_git_password: true,
    }))

    expect(container.textContent).toContain("Git Username")
    expect(container.textContent).toContain("Git Password / Token")

    await act(async () => {
      root.unmount()
    })
  })

  it("keeps credentials collapsed when the repo has none until the key button is clicked", async () => {
    const { container, root } = await renderDialog(buildRepo())

    expect(container.textContent).not.toContain("Git Username")
    expect(container.textContent).not.toContain("Git Password / Token")

    const toggle = container.querySelector('button[aria-label="Git credentials"]') as HTMLButtonElement | null

    expect(toggle).not.toBeNull()

    await clickElement(toggle as HTMLButtonElement)

    expect(container.textContent).toContain("Git Username")
    expect(container.textContent).toContain("Git Password / Token")

    await act(async () => {
      root.unmount()
    })
  })
})
