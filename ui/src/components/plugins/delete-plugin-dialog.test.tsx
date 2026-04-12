import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const { mockMutate, mockInvalidateQueries } = vi.hoisted(() => ({
  mockMutate: vi.fn(),
  mockInvalidateQueries: vi.fn(),
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

vi.mock("@/api/plugins", () => ({
  pluginsApi: {
    deletePlugin: vi.fn(),
  },
}))

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

vi.mock("@/components/ui/alert-dialog", () => ({
  AlertDialog: ({
    open,
    children,
  }: {
    open: boolean
    children: React.ReactNode
  }) => open ? <div>{children}</div> : null,
  AlertDialogContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogDescription: ({ children }: { children: React.ReactNode }) => <p>{children}</p>,
  AlertDialogFooter: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogTitle: ({ children }: { children: React.ReactNode }) => <h1>{children}</h1>,
  AlertDialogCancel: ({ children, ...props }: React.ComponentProps<"button">) => <button type="button" {...props}>{children}</button>,
  AlertDialogAction: ({ children, ...props }: React.ComponentProps<"button">) => <button type="button" {...props}>{children}</button>,
}))

import { DeletePluginDialog } from "./delete-plugin-dialog"

async function renderDialog(installCount: number) {
  const container = document.createElement("div")
  document.body.appendChild(container)
  const root = ReactDOMClient.createRoot(container)

  await act(async () => {
    root.render(
      <DeletePluginDialog
        open
        onOpenChange={() => undefined}
        projectId="project-1"
        plugin={{
          id: "plugin-1",
          slug: "plugin-one",
          name: "Plugin One",
          description: "",
          image: "docker.io/library/migrate:latest",
          registry_username: "",
          command: "",
          env_vars: [],
          plugin_type: "init",
          install_count: installCount,
          created_at: "2026-04-12T00:00:00Z",
          updated_at: "2026-04-12T00:00:00Z",
        }}
      />
    )
  })

  return { container, root }
}

describe("DeletePluginDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    ;(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true
  })

  afterEach(() => {
    document.body.innerHTML = ""
  })

  it("disables deletion and explains why when the plugin is installed in apps", async () => {
    const { container, root } = await renderDialog(2)

    expect(container.textContent).toContain("Plugin In Use")
    expect(container.textContent).toContain("installed in 2 apps")

    const actionButton = Array.from(container.querySelectorAll("button")).find((button) => button.textContent === "Uninstall First")
    expect(actionButton).toBeDefined()
    expect(actionButton?.hasAttribute("disabled")).toBe(true)

    await act(async () => {
      root.unmount()
    })
  })

  it("still allows deletion when the plugin is not installed", async () => {
    const { container, root } = await renderDialog(0)

    const actionButton = Array.from(container.querySelectorAll("button")).find((button) => button.textContent === "Delete Plugin")
    expect(actionButton).toBeDefined()
    expect(actionButton?.hasAttribute("disabled")).toBe(false)

    await act(async () => {
      actionButton?.dispatchEvent(new MouseEvent("click", { bubbles: true }))
    })

    expect(mockMutate).toHaveBeenCalledTimes(1)

    await act(async () => {
      root.unmount()
    })
  })
})
