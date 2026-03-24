import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

vi.mock("@/api/users", () => ({
  usersApi: {
    listMyAiProviders: vi.fn(),
    createMyAiProvider: vi.fn(),
    updateMyAiProvider: vi.fn(),
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
  EmptyState: ({ title, description }: { title: string; description: string }) => (
    <div>
      <div>{title}</div>
      <div>{description}</div>
    </div>
  ),
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

    expect(container.textContent).toContain("Personal AI providers")
    expect(container.textContent).toContain("No personal AI providers configured yet")
    expect(container.textContent).toContain("Add provider")

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
      created_at: new Date().toISOString(),
    })
    vi.mocked(usersApi.updateMyAiProvider).mockResolvedValue({
      id: "provider-1",
      provider_key: "openai-user",
      display_name: "OpenAI Personal Updated",
      base_url: "https://api.openai.com",
      default_model_profile_key: "gpt-4.1",
      enabled: true,
      created_at: new Date().toISOString(),
    })

    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<AccountAiProvidersPanel />)
    })

    expect(container.textContent).toContain("OpenAI Personal")

    const addButton = Array.from(container.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("Add provider")
    ) as HTMLButtonElement | undefined

    await act(async () => {
      addButton?.click()
    })

    expect(vi.mocked(usersApi.createMyAiProvider)).toHaveBeenCalled()

    const editButton = Array.from(container.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("Edit")
    ) as HTMLButtonElement | undefined

    await act(async () => {
      editButton?.click()
    })

    expect(vi.mocked(usersApi.updateMyAiProvider)).toHaveBeenCalledWith(
      "provider-1",
      expect.objectContaining({ display_name: "OpenAI Personal Updated" })
    )

    await act(async () => {
      root.unmount()
    })
  })
})
