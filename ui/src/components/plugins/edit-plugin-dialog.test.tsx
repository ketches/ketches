import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const { mockInvalidateQueries, mockMutate } = vi.hoisted(() => ({
  mockInvalidateQueries: vi.fn(),
  mockMutate: vi.fn(),
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
    updatePlugin: vi.fn(),
  },
}))

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

import { EditPluginDialog } from "./edit-plugin-dialog"

async function renderDialog(plugin: Record<string, unknown>) {
  const container = document.createElement("div")
  document.body.appendChild(container)
  const root = ReactDOMClient.createRoot(container)

  await act(async () => {
    root.render(
      <EditPluginDialog
        open
        onOpenChange={() => undefined}
        plugin={plugin}
        projectId="project-1"
      />,
    )
  })
  await act(async () => {
    await Promise.resolve()
  })

  return { container, root }
}

describe("EditPluginDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    ;(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true
  })

  afterEach(() => {
    document.body.innerHTML = ""
    vi.clearAllMocks()
  })

  it("shows a clear password icon for stored registry credentials", async () => {
    const { container, root } = await renderDialog({
      id: "plugin-1",
      name: "My Plugin",
      slug: "my-plugin",
      image: "ghcr.io/acme/plugin:latest",
      image_pull_policy: "IfNotPresent",
      plugin_type: "init",
      registry_username: "robot",
      has_registry_password: true,
      env_vars: [],
    })

    const clearButton = document.body.querySelector('button[aria-label="Clear password"]') as HTMLButtonElement | null
    expect(clearButton).not.toBeNull()
    expect(clearButton?.textContent).toBe("")

    await act(async () => {
      root.unmount()
    })
  })
})
