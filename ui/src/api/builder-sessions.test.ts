import { beforeEach, describe, expect, it, vi } from "vitest"

const { getMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
}))

vi.mock("./client", () => ({
  default: {
    get: getMock,
  },
}))

import { builderSessionsApi } from "./builder-sessions"

describe("builderSessionsApi.getModelSelection", () => {
  beforeEach(() => {
    getMock.mockReset()
  })

  it("maps snake_case model selection fields to the Builder UI shape", async () => {
    getMock.mockResolvedValue({
      options: [
        {
          key: "project-claude-sonnet",
          model_label: "Claude 4 Sonnet",
          provider_label: "Anthropic",
          scope: "project",
          provider_key: "anthropic-project",
          model_profile_key: "claude-sonnet-4",
        },
        {
          key: "user-gpt-4-1",
          model_label: "GPT-4.1",
          provider_label: "OpenAI",
          scope: "user",
          provider_key: "openai-user",
          model_profile_key: "gpt-4.1",
        },
      ],
      effective_default_source: "project",
      effective_default_option: {
        key: "project-claude-sonnet",
        model_label: "Claude 4 Sonnet",
        provider_label: "Anthropic",
        scope: "project",
        provider_key: "anthropic-project",
        model_profile_key: "claude-sonnet-4",
      },
    })

    await expect(builderSessionsApi.getModelSelection("project-1")).resolves.toEqual({
      options: [
        {
          key: "project-claude-sonnet",
          modelLabel: "Claude 4 Sonnet",
          providerLabel: "Anthropic",
          scope: "project",
          providerKey: "anthropic-project",
          modelProfileKey: "claude-sonnet-4",
        },
        {
          key: "user-gpt-4-1",
          modelLabel: "GPT-4.1",
          providerLabel: "OpenAI",
          scope: "user",
          providerKey: "openai-user",
          modelProfileKey: "gpt-4.1",
        },
      ],
      effectiveDefaultSource: "project",
      effectiveDefaultOption: {
        key: "project-claude-sonnet",
        modelLabel: "Claude 4 Sonnet",
        providerLabel: "Anthropic",
        scope: "project",
        providerKey: "anthropic-project",
        modelProfileKey: "claude-sonnet-4",
      },
    })

    expect(getMock).toHaveBeenCalledWith("/v1/projects/project-1/builder-model-selection")
  })
})

describe("builderSessionsApi.runLogsStreamUrl", () => {
  beforeEach(() => {
    vi.stubGlobal("localStorage", {
      getItem: vi.fn(() =>
        JSON.stringify({
          state: {
            isAuthenticated: true,
          },
        })
      ),
    })
  })

  it("builds an SSE URL rooted at the API base path while auth is handled by cookies", () => {
    expect(builderSessionsApi.runLogsStreamUrl("project-1", "session-1", "run-1")).toBe(
      "/api/v1/projects/project-1/builder-sessions/session-1/runs/run-1/logs"
    )
  })
})
