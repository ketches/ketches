import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const { mockEventSources } = vi.hoisted(() => ({
  mockEventSources: [] as MockEventSource[],
}))

class MockEventSource {
  onerror: ((event: Event) => void) | null = null
  onmessage: ((event: MessageEvent<string>) => void) | null = null
  onopen: ((event: Event) => void) | null = null

  constructor() {
    mockEventSources.push(this)
  }

  close() {}

  emitMessage(data: string) {
    this.onmessage?.(new MessageEvent("message", { data }))
  }
}

vi.mock("@/lib/auth-session", () => ({
  hasPersistedAuthSession: () => true,
}))

vi.mock("sonner", () => ({
  toast: {
    error: vi.fn(),
    success: vi.fn(),
  },
}))

vi.mock("./workload-panel-frame", () => ({
  WorkloadPanelFrame: ({
    children,
    toolbar,
    status,
  }: {
    children: React.ReactNode
    toolbar: React.ReactNode
    status: React.ReactNode
  }) => <div>{toolbar}{status}{children}</div>,
}))

vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  TooltipContent: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  TooltipTrigger: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}))

vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenu: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  DropdownMenuCheckboxItem: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  DropdownMenuContent: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  DropdownMenuGroup: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  DropdownMenuLabel: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  DropdownMenuSeparator: () => null,
  DropdownMenuTrigger: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}))

import { LogPanel } from "./log-panel"

const changeInputValue = async (input: HTMLInputElement, value: string) => {
  await act(async () => {
    const valueSetter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")?.set
    valueSetter?.call(input, value)
    input.dispatchEvent(new Event("input", { bubbles: true }))
    input.dispatchEvent(new Event("change", { bubbles: true }))
  })
}

const flushLogBatch = async () => {
  await act(async () => {
    vi.advanceTimersByTime(50)
  })
}

describe("LogPanel", () => {
  beforeEach(() => {
    vi.useFakeTimers()
    mockEventSources.length = 0
    vi.stubGlobal("EventSource", MockEventSource)
    vi.stubGlobal("requestAnimationFrame", (callback: FrameRequestCallback) => {
      callback(0)
      return 0
    })
    vi.stubGlobal("ResizeObserver", class ResizeObserver {
      observe() {}
      disconnect() {}
      unobserve() {}
    })
    ;(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
    document.body.innerHTML = ""
  })

  it("treats special search characters as literal text", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<LogPanel appId="app-1" instanceName="pod-1" containerName="web" />)
    })
    await act(async () => {
      mockEventSources[0]?.emitMessage("value [one] and [TWO]")
    })
    await flushLogBatch()

    const searchInput = container.querySelector('input[placeholder="Search logs..."]') as HTMLInputElement
    await changeInputValue(searchInput, "[")

    const highlights = Array.from(container.querySelectorAll("span.bg-yellow-500\\/40"))
    expect(highlights.map((element) => element.textContent)).toEqual(["[", "["])
    expect(container.textContent).toContain("value [one] and [TWO]")

    await act(async () => {
      root.unmount()
    })
  })

  it("highlights every case-insensitive literal match", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<LogPanel appId="app-1" instanceName="pod-1" containerName="web" />)
    })
    await act(async () => {
      mockEventSources[0]?.emitMessage("Error then ERROR")
    })
    await flushLogBatch()

    const searchInput = container.querySelector('input[placeholder="Search logs..."]') as HTMLInputElement
    await changeInputValue(searchInput, "error")

    const highlights = Array.from(container.querySelectorAll("span.bg-yellow-500\\/40"))
    expect(highlights.map((element) => element.textContent)).toEqual(["Error", "ERROR"])

    await act(async () => {
      root.unmount()
    })
  })

  it("commits streamed logs in batches", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<LogPanel appId="app-1" instanceName="pod-1" containerName="web" />)
    })
    await act(async () => {
      for (let index = 0; index < 100; index += 1) {
        mockEventSources[0]?.emitMessage(`line-${index}`)
      }
    })

    expect(container.textContent).toContain("Lines: 0")
    await flushLogBatch()
    expect(container.textContent).toContain("Lines: 100")

    await act(async () => {
      root.unmount()
    })
  })

  it("retains only the newest entries when the ring buffer reaches capacity", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<LogPanel appId="app-1" instanceName="pod-1" containerName="web" />)
    })
    await act(async () => {
      for (let index = 0; index < 10005; index += 1) {
        mockEventSources[0]?.emitMessage(`entry-${index.toString().padStart(5, "0")}`)
      }
    })
    await flushLogBatch()

    expect(container.textContent).toContain("Lines: 10000")
    expect(container.textContent).toContain("entry-00005")
    expect(container.textContent).not.toContain("entry-00004")

    await act(async () => {
      root.unmount()
    })
  })

  it("virtualizes large log streams without inline positioning styles", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<LogPanel appId="app-1" instanceName="pod-1" containerName="web" />)
    })
    await act(async () => {
      for (let index = 0; index < 5000; index += 1) {
        mockEventSources[0]?.emitMessage(`virtual-${index}`)
      }
    })
    await flushLogBatch()

    const virtualList = container.querySelector('[data-testid="virtual-log-list"]') as HTMLDivElement
    const renderedRows = virtualList.querySelectorAll<HTMLElement>('[data-testid="virtual-log-row"]')
    const spacers = virtualList.querySelectorAll<SVGElement>('[data-testid="virtual-log-spacer"]')
    const spacerHeight = Array.from(spacers).reduce(
      (height, spacer) => height + Number(spacer.getAttribute("height")),
      0
    )

    expect(virtualList.hasAttribute("style")).toBe(false)
    expect(virtualList.querySelector("[style]")).toBeNull()
    expect(spacers).toHaveLength(2)
    expect(spacerHeight + renderedRows.length * 24).toBe(120000)
    expect(renderedRows.length).toBeGreaterThan(0)
    expect(renderedRows.length).toBeLessThan(100)

    await act(async () => {
      root.unmount()
    })
  })
})
