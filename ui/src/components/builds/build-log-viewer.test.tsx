import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const { mockEventSources, mockParseBuildLogAnsi, mockUseQuery } = vi.hoisted(() => ({
  mockEventSources: [] as MockEventSource[],
  mockParseBuildLogAnsi: vi.fn((input: string) => ({
    text: input,
    decorations: [],
  })),
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

vi.mock("@/lib/build-log-ansi", () => ({
  parseBuildLogAnsi: (input: string) => mockParseBuildLogAnsi(input),
}))

vi.mock("@monaco-editor/react", () => ({
  default: ({ value }: { value?: string }) => <div data-testid="mock-editor">{value}</div>,
}))

import { BuildLogViewer } from "./build-log-viewer"

class MockEventSource {
  private readonly listeners = new Map<string, Array<(event: { data: string }) => void>>()

  constructor(_url: string) {
    mockEventSources.push(this)
  }

  addEventListener(type: string, listener: (event: { data: string }) => void) {
    const current = this.listeners.get(type) ?? []
    current.push(listener)
    this.listeners.set(type, current)
  }

  close() {}

  emit(type: string, data = "") {
    for (const listener of this.listeners.get(type) ?? []) {
      listener({ data })
    }
  }
}

describe("BuildLogViewer", () => {
  beforeEach(() => {
    mockEventSources.length = 0
    mockParseBuildLogAnsi.mockClear()
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
    vi.useRealTimers()
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

  it("buffers burst log chunks until the scheduled flush runs", async () => {
    vi.useFakeTimers()
    mockUseQuery.mockReturnValue({
      data: {
        build_number: 14,
        status: "succeeded",
        log_persist_status: "succeeded",
        log_persist_error: "",
      },
    })

    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<BuildLogViewer buildId="build-3" repoId="repo-1" />)
    })

    mockParseBuildLogAnsi.mockClear()

    await act(async () => {
      mockEventSources[0].emit("log", "chunk-1\n")
    })
    await act(async () => {
      mockEventSources[0].emit("log", "chunk-2\n")
    })

    expect(mockParseBuildLogAnsi).not.toHaveBeenCalled()

    await act(async () => {
      vi.runOnlyPendingTimers()
    })

    expect(mockParseBuildLogAnsi).toHaveBeenCalledTimes(1)
    expect(mockParseBuildLogAnsi).toHaveBeenLastCalledWith("chunk-1\nchunk-2\n")
    expect(container.textContent).toContain("chunk-1")
    expect(container.textContent).toContain("chunk-2")

    await act(async () => {
      root.unmount()
    })
  })

  it("flushes buffered log chunks immediately when the stream ends", async () => {
    vi.useFakeTimers()
    mockUseQuery.mockReturnValue({
      data: {
        build_number: 15,
        status: "succeeded",
        log_persist_status: "succeeded",
        log_persist_error: "",
      },
    })

    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<BuildLogViewer buildId="build-4" repoId="repo-1" />)
    })

    mockParseBuildLogAnsi.mockClear()

    await act(async () => {
      mockEventSources[0].emit("log", "final chunk\n")
    })

    expect(mockParseBuildLogAnsi).not.toHaveBeenCalled()

    await act(async () => {
      mockEventSources[0].emit("done", "stream ended")
    })

    expect(mockParseBuildLogAnsi).toHaveBeenCalledTimes(1)
    expect(mockParseBuildLogAnsi).toHaveBeenLastCalledWith("final chunk\n")
    expect(container.textContent).toContain("final chunk")

    await act(async () => {
      root.unmount()
    })
  })
})
