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

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

import type { ContainerRegistry } from "@/api/container-registries"
import { ContainerRegistryDialog } from "./container-registry-dialog"

async function renderDialog(registry: Partial<ContainerRegistry>) {
  const container = document.createElement("div")
  document.body.appendChild(container)
  const root = ReactDOMClient.createRoot(container)

  await act(async () => {
    root.render(
      <ContainerRegistryDialog
        open
        onOpenChange={() => undefined}
        scope="project"
        scopeId="project-1"
        registry={registry as ContainerRegistry}
      />,
    )
  })
  await act(async () => {
    await Promise.resolve()
  })

  return { container, root }
}

describe("ContainerRegistryDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    ;(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true
  })

  afterEach(() => {
    document.body.innerHTML = ""
    vi.clearAllMocks()
  })

  it("shows a clear password icon and restores an editable input after clicking it", async () => {
    const { container, root } = await renderDialog({
      id: "registry-1",
      name: "My Registry",
      provider: "ghcr",
      endpoint: "ghcr.io",
      username: "user",
      has_password: true,
      enabled: true,
      is_default: false,
    })

    const clearButton = document.body.querySelector('button[aria-label="Clear password"]') as HTMLButtonElement | null
    expect(clearButton).not.toBeNull()
    expect(clearButton?.textContent).toBe("")

    const passwordInputBefore = document.body.querySelector("#registry-password") as HTMLInputElement | null
    expect(passwordInputBefore).not.toBeNull()
    expect(passwordInputBefore?.readOnly).toBe(true)

    await act(async () => {
      clearButton?.dispatchEvent(new MouseEvent("click", { bubbles: true }))
    })

    const passwordInputAfter = document.body.querySelector("#registry-password") as HTMLInputElement | null
    expect(passwordInputAfter).not.toBeNull()
    expect(passwordInputAfter?.readOnly).toBe(false)
    expect(passwordInputAfter?.disabled).toBe(false)
    expect(document.body.querySelector('button[aria-label="Clear password"]')).toBeNull()

    await act(async () => {
      root.unmount()
    })
  })
})
