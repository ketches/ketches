import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const { mockUseQuery } = vi.hoisted(() => ({
  mockUseQuery: vi.fn(),
}))

vi.mock("@tanstack/react-query", () => ({
  useQuery: () => mockUseQuery(),
}))

vi.mock("@/components/theme-provider/theme-provider", () => ({
  useTheme: () => ({ theme: "light" }),
}))

vi.mock("@/components/builds/build-status-badge", () => ({
  BuildStatusBadge: ({ status }: { status: string }) => <span>{status}</span>,
}))

vi.mock("@monaco-editor/react", () => ({
  default: () => <div data-testid="mock-editor">editor</div>,
}))

import { BuildLogViewer } from "./build-log-viewer"

describe("BuildLogViewer", () => {
  beforeEach(() => {
    class MockEventSource {
      addEventListener() {}
      close() {}
    }

    vi.stubGlobal("EventSource", MockEventSource)
    vi.stubGlobal(
      "ResizeObserver",
      class {
        observe() {}
        disconnect() {}
      },
    )
    vi.stubGlobal("localStorage", {
      getItem: () => null,
      setItem: () => undefined,
      removeItem: () => undefined,
      clear: () => undefined,
    })
  })

  afterEach(() => {
    document.body.innerHTML = ""
    vi.clearAllMocks()
    vi.unstubAllGlobals()
  })

  it("shows archive failure messaging for terminal builds", async () => {
    mockUseQuery.mockReturnValue({
      data: {
        build_number: 12,
        status: "failed",
        log_persist_status: "failed",
        log_persist_error: "archive write failed",
      },
    })

    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<BuildLogViewer buildId="build-1" repoId="repo-1" />)
    })

    expect(container.textContent).toContain("Archived build log unavailable")
    expect(container.textContent).toContain("archive write failed")

    await act(async () => {
      root.unmount()
    })
  })

  it("shows archive expired messaging when the archive was deleted", async () => {
    mockUseQuery.mockReturnValue({
      data: {
        build_number: 13,
        status: "succeeded",
        log_persist_status: "expired",
        log_persist_error: "",
      },
    })

    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<BuildLogViewer buildId="build-2" repoId="repo-1" />)
    })

    expect(container.textContent).toContain("Archived build log expired")

    await act(async () => {
      root.unmount()
    })
  })
})
