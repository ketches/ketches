import ReactDOMClient from "react-dom/client"
import { act } from "react"
import { afterEach, describe, expect, it, vi } from "vitest"

import type { BuilderSession } from "@/api/builder-sessions"

import { BuilderSessionHistoryRail } from "./builder-session-history-rail"

function buildSession(overrides: Partial<BuilderSession> = {}): BuilderSession {
  return {
    id: "session-1",
    project_id: "project-1",
    build_env_id: "env-1",
    title: "Landing page builder",
    summary: "Build a landing page",
    status: "ready",
    created_by: "user-1",
    created_at: "2026-03-20T00:00:00Z",
    updated_at: "2026-03-20T00:10:00Z",
    last_activity_at: "2026-03-20T00:10:00Z",
    expires_at: null,
    latest_run_id: "run-1",
    latest_run_status: "succeeded",
    current_workspace_id: "workspace-1",
    current_workspace_status: "ready",
    current_workspace_root: "/workspace",
    artifact_count: 0,
    ...overrides,
  }
}

describe("BuilderSessionHistoryRail", () => {
  afterEach(() => {
    document.body.innerHTML = ""
  })

  it("is expanded by default, keeps new conversation at the top, and constrains the list for scrolling", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)
    const sessions = [buildSession(), buildSession({ id: "session-2", title: "API builder" })]

    await act(async () => {
      root.render(
        <BuilderSessionHistoryRail
          sessions={sessions}
          selectedSessionId="session-1"
          onNewConversation={vi.fn()}
          onSelectSession={vi.fn()}
        />
      )
    })

    const rail = container.querySelector('[data-testid="builder-session-history"]')
    const list = container.querySelector('[data-testid="builder-session-history-list"]')

    expect(list).not.toBeNull()
    expect(rail?.className).toContain("h-full")
    expect(rail?.className).toContain("min-h-0")
    expect(list?.className).toContain("overflow-y-auto")

    const header = container.querySelector('[data-testid="builder-session-history-header"]')
    const footer = container.querySelector('[data-testid="builder-session-history-footer"]')
    const newConversationButton = container.querySelector('[data-testid="builder-session-history-new"]') as HTMLButtonElement | null
    const toggle = container.querySelector('[data-testid="builder-session-history-toggle"]') as HTMLButtonElement | null

    expect(header).not.toBeNull()
    expect(footer).not.toBeNull()
    expect(newConversationButton?.className).toContain("w-full")
    expect(toggle?.className).toContain("w-full")
    expect(header?.contains(newConversationButton ?? null)).toBe(true)
    expect(footer?.contains(toggle ?? null)).toBe(true)

    await act(async () => {
      toggle?.click()
    })

    expect(container.querySelector('[data-testid="builder-session-history-list"]')).toBeNull()

    await act(async () => {
      root.unmount()
    })
  })
})
