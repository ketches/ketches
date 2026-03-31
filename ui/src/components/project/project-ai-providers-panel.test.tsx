import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

vi.mock("@/api/projects", () => ({
  projectsApi: {
    listAiProviders: vi.fn(),
    createAiProvider: vi.fn(),
    updateAiProvider: vi.fn(),
    deleteAiProvider: vi.fn(),
  },
}))

vi.mock("@tanstack/react-query", () => ({
  useQuery: ({ queryFn }: { queryFn: () => Promise<unknown> }) => {
    void queryFn()
    return { data: (globalThis as typeof globalThis & { __projectAiProvidersMockData?: unknown[] }).__projectAiProvidersMockData ?? [] }
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
  EmptyState: ({ title, description }: { title: string; description: string }) => (
    <div>
      <div>{title}</div>
      <div>{description}</div>
    </div>
  ),
}))

import { ProjectAiProvidersPanel } from "./project-ai-providers-panel"
import { projectsApi } from "@/api/projects"

describe("ProjectAiProvidersPanel", () => {
  beforeEach(() => {
    ;(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true
    ;(globalThis as typeof globalThis & { __projectAiProvidersMockData?: unknown[] }).__projectAiProvidersMockData = []
    vi.mocked(projectsApi.listAiProviders).mockResolvedValue([])
  })

  afterEach(() => {
    document.body.innerHTML = ""
    ;(globalThis as typeof globalThis & { __projectAiProvidersMockData?: unknown[] }).__projectAiProvidersMockData = []
    vi.clearAllMocks()
  })

  it("renders an empty state and add-provider affordance when no project providers exist", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<ProjectAiProvidersPanel projectId="project-1" />)
    })

    expect(container.textContent).toContain("Project AI providers")
    expect(container.textContent).toContain("No project AI providers configured yet")
    expect(container.textContent).toContain("Add provider")

    await act(async () => {
      root.unmount()
    })
  })

  it("renders existing project providers and submits create and update mutations", async () => {
    vi.mocked(projectsApi.listAiProviders).mockResolvedValue([
      {
        id: "provider-1",
        provider_key: "anthropic-project",
        display_name: "Anthropic Shared",
        base_url: "https://api.anthropic.com",
        default_model_profile_key: "claude-sonnet-4",
        enabled: true,
        is_default: true,
        created_at: new Date().toISOString(),
      },
    ])
    ;(globalThis as typeof globalThis & { __projectAiProvidersMockData?: unknown[] }).__projectAiProvidersMockData = [
      {
        id: "provider-1",
        provider_key: "anthropic-project",
        display_name: "Anthropic Shared",
        base_url: "https://api.anthropic.com",
        default_model_profile_key: "claude-sonnet-4",
        enabled: true,
        is_default: true,
        created_at: new Date().toISOString(),
      },
    ]
    vi.mocked(projectsApi.createAiProvider).mockResolvedValue({
      id: "provider-2",
      provider_key: "openai-project",
      display_name: "OpenAI Shared",
      base_url: "https://api.openai.com",
      default_model_profile_key: "gpt-4.1",
      enabled: true,
      is_default: false,
      created_at: new Date().toISOString(),
    })
    vi.mocked(projectsApi.updateAiProvider).mockResolvedValue({
      id: "provider-1",
      provider_key: "anthropic-project",
      display_name: "Anthropic Shared Updated",
      base_url: "https://api.anthropic.com",
      default_model_profile_key: "claude-sonnet-4",
      enabled: true,
      is_default: false,
      created_at: new Date().toISOString(),
    })
    vi.mocked(projectsApi.deleteAiProvider).mockResolvedValue(undefined)

    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<ProjectAiProvidersPanel projectId="project-1" />)
    })

    expect(container.textContent).toContain("Anthropic Shared")
    expect(container.textContent).toContain("Enabled")
    expect(container.textContent).toContain("Default")

    const addButton = Array.from(container.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("Add provider")
    ) as HTMLButtonElement | undefined

    await act(async () => {
      addButton?.click()
    })

    expect(container.textContent).toContain("Provider key")

    const providerKeyInput = container.querySelector('input[name="provider_key"]') as HTMLInputElement | null
    const displayNameInput = container.querySelector('input[name="display_name"]') as HTMLInputElement | null
    const baseUrlInput = container.querySelector('input[name="base_url"]') as HTMLInputElement | null
    const apiKeyInput = container.querySelector('input[name="api_key"]') as HTMLInputElement | null
    const modelProfileInput = container.querySelector('input[name="default_model_profile_key"]') as HTMLInputElement | null
    const enabledInput = container.querySelector('input[name="enabled"]') as HTMLInputElement | null
    const defaultInput = container.querySelector('input[name="is_default"]') as HTMLInputElement | null
    const saveButton = Array.from(container.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("Save provider")
    ) as HTMLButtonElement | undefined

    await act(async () => {
      if (providerKeyInput) {
        providerKeyInput.value = "openai-project"
        providerKeyInput.dispatchEvent(new Event("input", { bubbles: true }))
      }
      if (displayNameInput) {
        displayNameInput.value = "OpenAI Shared"
        displayNameInput.dispatchEvent(new Event("input", { bubbles: true }))
      }
      if (baseUrlInput) {
        baseUrlInput.value = "https://api.openai.com"
        baseUrlInput.dispatchEvent(new Event("input", { bubbles: true }))
      }
      if (apiKeyInput) {
        apiKeyInput.value = "shared-key"
        apiKeyInput.dispatchEvent(new Event("input", { bubbles: true }))
      }
      if (modelProfileInput) {
        modelProfileInput.value = "gpt-4.1"
        modelProfileInput.dispatchEvent(new Event("input", { bubbles: true }))
      }
      enabledInput?.click()
      defaultInput?.click()
    })

    await act(async () => {
      saveButton?.click()
    })

    expect(vi.mocked(projectsApi.createAiProvider)).toHaveBeenCalledWith(
      "project-1",
      expect.objectContaining({ display_name: "OpenAI Shared", enabled: false, is_default: true })
    )

    const editButton = Array.from(container.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("Edit")
    ) as HTMLButtonElement | undefined

    await act(async () => {
      editButton?.click()
    })

    expect((container.querySelector('input[name="display_name"]') as HTMLInputElement | null)?.value).toBe("Anthropic Shared")
    expect((container.querySelector('input[name="enabled"]') as HTMLInputElement | null)?.checked).toBe(true)
    expect((container.querySelector('input[name="is_default"]') as HTMLInputElement | null)?.checked).toBe(true)

    const editDisplayNameInput = container.querySelector('input[name="display_name"]') as HTMLInputElement | null
    const updateButton = Array.from(container.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("Update provider")
    ) as HTMLButtonElement | undefined
    const editEnabledInput = container.querySelector('input[name="enabled"]') as HTMLInputElement | null
    const editDefaultInput = container.querySelector('input[name="is_default"]') as HTMLInputElement | null

    await act(async () => {
      if (editDisplayNameInput) {
        editDisplayNameInput.value = "Anthropic Shared Updated"
        editDisplayNameInput.dispatchEvent(new Event("input", { bubbles: true }))
      }
      editEnabledInput?.click()
      editDefaultInput?.click()
    })

    await act(async () => {
      updateButton?.click()
    })

    expect(vi.mocked(projectsApi.updateAiProvider)).toHaveBeenCalledWith(
      "project-1",
      "provider-1",
      expect.objectContaining({ display_name: "Anthropic Shared Updated", enabled: false, is_default: false })
    )

    const deleteButton = Array.from(container.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("Delete")
    ) as HTMLButtonElement | undefined

    await act(async () => {
      deleteButton?.click()
    })

    expect(vi.mocked(projectsApi.deleteAiProvider)).toHaveBeenCalledWith("project-1", "provider-1")

    await act(async () => {
      root.unmount()
    })
  })
})
