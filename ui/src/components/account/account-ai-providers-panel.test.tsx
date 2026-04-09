import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

vi.mock("@/api/users", () => ({
  usersApi: {
    listMyAiProviders: vi.fn(),
    createMyAiProvider: vi.fn(),
    updateMyAiProvider: vi.fn(),
    deleteMyAiProvider: vi.fn(),
  },
}))

vi.mock("@tanstack/react-query", () => ({
  useQuery: ({ queryFn }: { queryFn: () => Promise<unknown> }) => {
    void queryFn()
    return { data: (globalThis as typeof globalThis & { __accountAiProvidersMockData?: unknown[] }).__accountAiProvidersMockData ?? [] }
  },
  useMutation: ({ mutationFn }: { mutationFn: (payload: unknown) => Promise<unknown> }) => ({
    mutateAsync: mutationFn,
    isPending: false,
  }),
  useQueryClient: () => ({
    invalidateQueries: vi.fn(),
  }),
}))

vi.mock("@/components/shared/empty-state", () => ({
  EmptyState: ({ title, description, actionText, onAction }: { title: string; description: string; actionText?: string; onAction?: () => void }) => (
    <div>
      <div>{title}</div>
      <div>{description}</div>
      {actionText && onAction ? <button type="button" onClick={onAction}>{actionText}</button> : null}
    </div>
  ),
}))

vi.mock("@/components/ui/dialog", () => ({
  Dialog: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogTitle: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogDescription: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogFooter: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

import { AccountAiProvidersPanel } from "./account-ai-providers-panel"
import { usersApi } from "@/api/users"

describe("AccountAiProvidersPanel", () => {
  beforeEach(() => {
    ;(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true
    ;(globalThis as typeof globalThis & { __accountAiProvidersMockData?: unknown[] }).__accountAiProvidersMockData = []
    vi.mocked(usersApi.listMyAiProviders).mockResolvedValue([])
  })

  afterEach(() => {
    document.body.innerHTML = ""
    ;(globalThis as typeof globalThis & { __accountAiProvidersMockData?: unknown[] }).__accountAiProvidersMockData = []
    vi.clearAllMocks()
  })

  it("renders an empty state and add-provider affordance when no user providers exist", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<AccountAiProvidersPanel />)
    })

    expect(container.textContent).toContain("No AI providers yet")
    expect(container.textContent).toContain("Add Provider")

    await act(async () => {
      root.unmount()
    })
  })

  it("renders existing providers and submits create and update mutations", async () => {
    vi.mocked(usersApi.listMyAiProviders).mockResolvedValue([
      {
        id: "provider-1",
        provider_key: "openai-user",
        display_name: "OpenAI Personal",
        base_url: "https://api.openai.com",
        default_model_profile_key: "gpt-4.1",
        enabled: true,
        is_default: true,
        created_at: new Date().toISOString(),
      },
    ])
    ;(globalThis as typeof globalThis & { __accountAiProvidersMockData?: unknown[] }).__accountAiProvidersMockData = [
      {
        id: "provider-1",
        provider_key: "openai-user",
        display_name: "OpenAI Personal",
        base_url: "https://api.openai.com",
        default_model_profile_key: "gpt-4.1",
        enabled: true,
        is_default: true,
        created_at: new Date().toISOString(),
      },
    ]
    vi.mocked(usersApi.createMyAiProvider).mockResolvedValue({
      id: "provider-2",
      provider_key: "anthropic-user",
      display_name: "Anthropic Personal",
      base_url: "https://api.anthropic.com",
      default_model_profile_key: "claude-sonnet-4",
      enabled: true,
      is_default: false,
      created_at: new Date().toISOString(),
    })
    vi.mocked(usersApi.updateMyAiProvider).mockResolvedValue({
      id: "provider-1",
      provider_key: "openai-user",
      display_name: "OpenAI Personal Updated",
      base_url: "https://api.openai.com",
      default_model_profile_key: "gpt-4.1",
      enabled: true,
      is_default: false,
      created_at: new Date().toISOString(),
    })
    vi.mocked(usersApi.deleteMyAiProvider).mockResolvedValue(undefined)

    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<AccountAiProvidersPanel />)
    })

    expect(container.textContent).toContain("OpenAI Personal")
    expect(container.textContent).toContain("Enabled")
    expect(container.textContent).toContain("Default")

    const addButton = Array.from(container.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("Add Provider")
    ) as HTMLButtonElement | undefined

    await act(async () => {
      addButton?.click()
    })

    expect(container.textContent).toContain("Add AI Provider")

    const providerKeyInput = container.querySelector('input[name="provider_key"]') as HTMLInputElement | null
    const displayNameInput = container.querySelector('input[name="display_name"]') as HTMLInputElement | null
    const baseUrlInput = container.querySelector('input[name="base_url"]') as HTMLInputElement | null
    const apiKeyInput = container.querySelector('input[name="api_key"]') as HTMLInputElement | null
    const modelProfileInput = container.querySelector('input[name="default_model_profile_key"]') as HTMLInputElement | null
    const enabledInput = container.querySelector('input[name="enabled"]') as HTMLInputElement | null
    const defaultInput = container.querySelector('input[name="is_default"]') as HTMLInputElement | null
    const saveButton = Array.from(container.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("Save Provider")
    ) as HTMLButtonElement | undefined

    await act(async () => {
      if (providerKeyInput) {
        providerKeyInput.value = "anthropic-user"
        providerKeyInput.dispatchEvent(new Event("input", { bubbles: true }))
      }
      if (displayNameInput) {
        displayNameInput.value = "Anthropic Personal"
        displayNameInput.dispatchEvent(new Event("input", { bubbles: true }))
      }
      if (baseUrlInput) {
        baseUrlInput.value = "https://api.anthropic.com"
        baseUrlInput.dispatchEvent(new Event("input", { bubbles: true }))
      }
      if (apiKeyInput) {
        apiKeyInput.value = "test-key"
        apiKeyInput.dispatchEvent(new Event("input", { bubbles: true }))
      }
      if (modelProfileInput) {
        modelProfileInput.value = "claude-sonnet-4"
        modelProfileInput.dispatchEvent(new Event("input", { bubbles: true }))
      }
      enabledInput?.click()
      defaultInput?.click()
    })

    await act(async () => {
      saveButton?.click()
    })

    expect(vi.mocked(usersApi.createMyAiProvider)).toHaveBeenCalledWith(expect.objectContaining({
      provider_key: "anthropic-user",
      display_name: "Anthropic Personal",
      base_url: "https://api.anthropic.com",
      api_key: "test-key",
      default_model_profile_key: "claude-sonnet-4",
      enabled: false,
      is_default: true,
    }))

    const editButton = Array.from(container.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("Edit")
    ) as HTMLButtonElement | undefined

    await act(async () => {
      editButton?.click()
    })

    expect((container.querySelector('input[name="display_name"]') as HTMLInputElement | null)?.value).toBe("OpenAI Personal")
    expect((container.querySelector('input[name="enabled"]') as HTMLInputElement | null)?.checked).toBe(true)
    expect((container.querySelector('input[name="is_default"]') as HTMLInputElement | null)?.checked).toBe(true)

    const editDisplayNameInput = container.querySelector('input[name="display_name"]') as HTMLInputElement | null
    const updateButton = Array.from(container.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("Update Provider")
    ) as HTMLButtonElement | undefined
    const editEnabledInput = container.querySelector('input[name="enabled"]') as HTMLInputElement | null
    const editDefaultInput = container.querySelector('input[name="is_default"]') as HTMLInputElement | null

    await act(async () => {
      if (editDisplayNameInput) {
        editDisplayNameInput.value = "OpenAI Personal Updated"
        editDisplayNameInput.dispatchEvent(new Event("input", { bubbles: true }))
      }
      editEnabledInput?.click()
      editDefaultInput?.click()
    })

    await act(async () => {
      updateButton?.click()
    })

    expect(vi.mocked(usersApi.updateMyAiProvider)).toHaveBeenCalledWith(
      "provider-1",
      expect.objectContaining({ display_name: "OpenAI Personal Updated", enabled: false, is_default: false })
    )

    const deleteButton = Array.from(container.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("Delete")
    ) as HTMLButtonElement | undefined

    await act(async () => {
      deleteButton?.click()
    })

    expect(vi.mocked(usersApi.deleteMyAiProvider)).toHaveBeenCalledWith("provider-1")

    await act(async () => {
      root.unmount()
    })
  })
})
