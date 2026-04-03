import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import { TerminalPanel } from "./terminal-panel"

const websocketUrls: string[] = []
const websocketSends: string[] = []

vi.mock("xterm", () => ({
  Terminal: class MockTerminal {
    cols = 80
    rows = 24

    loadAddon() { }
    open() { }
    focus() { }
    write() { }
    writeln() { }
    dispose() { }

    onData() {
      return {
        dispose() { },
      }
    }
  },
}))

vi.mock("xterm-addon-fit", () => ({
  FitAddon: class MockFitAddon {
    fit() { }
  },
}))

vi.mock("xterm-addon-web-links", () => ({
  WebLinksAddon: class MockWebLinksAddon { },
}))

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, ...props }: React.ComponentProps<"button">) => <button {...props}>{children}</button>,
}))

vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  TooltipContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  TooltipTrigger: ({
    children,
  }: {
    children: React.ReactNode
    render: React.ReactElement
  }) => {
    return <button>{children}</button>
  },
}))

vi.mock("@/components/ui/empty", () => ({
  Empty: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  EmptyContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  EmptyDescription: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  EmptyHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  EmptyMedia: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  EmptyTitle: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/separator", () => ({
  Separator: () => <div />,
}))

class MockWebSocket {
  static CONNECTING = 0
  static OPEN = 1
  static CLOSING = 2
  static CLOSED = 3

  url: string
  readyState = MockWebSocket.CONNECTING
  binaryType = "arraybuffer"
  onopen: ((event: Event) => void) | null = null
  onmessage: ((event: MessageEvent) => void) | null = null
  onerror: ((event: Event) => void) | null = null
  onclose: ((event: CloseEvent) => void) | null = null

  constructor(url: string) {
    this.url = url
    websocketUrls.push(url)
    window.setTimeout(() => {
      this.readyState = MockWebSocket.OPEN
      this.onopen?.(new Event("open"))
    }, 0)
  }

  send(data: string) {
    websocketSends.push(data)
  }

  close(code = 1000, reason = "") {
    this.readyState = MockWebSocket.CLOSED
    this.onclose?.({ code, reason, wasClean: true } as CloseEvent)
  }
}

function createMemoryStorage() {
  const store = new Map<string, string>()
  return {
    clear() {
      store.clear()
    },
    getItem(key: string) {
      return store.get(key) ?? null
    },
    key(index: number) {
      return Array.from(store.keys())[index] ?? null
    },
    removeItem(key: string) {
      store.delete(key)
    },
    setItem(key: string, value: string) {
      store.set(key, value)
    },
    get length() {
      return store.size
    },
  }
}

describe("TerminalPanel", () => {
  beforeEach(() => {
    websocketUrls.length = 0
    websocketSends.length = 0
    vi.useFakeTimers()
    vi.stubGlobal("WebSocket", MockWebSocket)
    vi.stubGlobal("fetch", vi.fn())
    vi.stubGlobal("localStorage", createMemoryStorage())
    vi.stubGlobal(
      "ResizeObserver",
      class ResizeObserver {
        observe() { }
        unobserve() { }
        disconnect() { }
      }
    )
    vi.stubGlobal("requestAnimationFrame", (callback: FrameRequestCallback) => window.setTimeout(() => callback(0), 0))
    vi.stubGlobal("cancelAnimationFrame", (handle: number) => window.clearTimeout(handle))

    Object.defineProperty(HTMLElement.prototype, "getBoundingClientRect", {
      configurable: true,
      value: () => ({
        width: 800,
        height: 400,
        top: 0,
        left: 0,
        right: 800,
        bottom: 400,
        x: 0,
        y: 0,
        toJSON() {
          return {}
        },
      }),
    })

    window.history.replaceState({}, "", "/clusters/test")
    localStorage.setItem("auth-storage", JSON.stringify({
      state: {
        isAuthenticated: true,
      },
    }))
  })

  afterEach(() => {
    document.body.innerHTML = ""
    vi.useRealTimers()
    vi.unstubAllGlobals()
    vi.clearAllMocks()
  })

  it("opens one node exec websocket session without any precreate request", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(
        <TerminalPanel
          appId="cluster-1"
          instanceName="node-a"
          containerName="shell"
          targetType="node"
        />
      )
    })

    await act(async () => {
      vi.runAllTimers()
    })

		expect(container.textContent).toContain("Session 1")
		expect(websocketUrls).toHaveLength(1)
		expect(websocketUrls[0]).toContain("/api/v1/clusters/cluster-1/nodes/node-a/exec")
		expect(websocketUrls[0]).not.toContain("token=")
		expect(fetch).not.toHaveBeenCalled()
		expect(websocketSends.some((value) => value.includes("\"type\":\"resize\""))).toBe(true)

    await act(async () => {
      root.unmount()
    })
  })
})
