import type { ReactNode } from "react"
import { renderToStaticMarkup } from "react-dom/server"
import { afterEach, describe, expect, it, vi } from "vitest"
import type { QueryClient } from "@tanstack/react-query"

const { createRootMock, queryClientProviderMock, renderMock } = vi.hoisted(() => {
  const renderMock = vi.fn()

  return {
    createRootMock: vi.fn(() => ({ render: renderMock })),
    queryClientProviderMock: vi.fn(),
    renderMock,
  }
})

vi.mock("react-dom/client", () => ({
  createRoot: createRootMock,
}))

vi.mock("@tanstack/react-query", async () => {
  const actual = await vi.importActual<typeof import("@tanstack/react-query")>("@tanstack/react-query")

  return {
    ...actual,
    QueryClientProvider: ({ children, client }: { children: ReactNode; client: QueryClient }) => {
      queryClientProviderMock({ client })

      return <div data-testid="query-client-provider">{children}</div>
    },
  }
})

vi.mock("./App.tsx", () => ({
  default: () => <div data-testid="app-root">app</div>,
}))

vi.mock("@/components/ui/tooltip", () => ({
  TooltipProvider: ({ children }: { children: ReactNode }) => (
    <div data-testid="tooltip-provider">{children}</div>
  ),
}))

describe("main bootstrap", () => {
  afterEach(() => {
    document.body.innerHTML = ""
    renderMock.mockReset()
    createRootMock.mockClear()
    queryClientProviderMock.mockReset()
    vi.resetModules()
  })

  it("wraps the app with the tooltip provider", async () => {
    const container = document.createElement("div")
    container.id = "root"
    document.body.appendChild(container)

    await import("./main.tsx")

    expect(createRootMock).toHaveBeenCalledWith(container)
    expect(renderMock).toHaveBeenCalledOnce()

    const markup = renderToStaticMarkup(renderMock.mock.calls[0][0])

    expect(markup).toContain('data-testid="tooltip-provider"')
    expect(markup).toContain('data-testid="app-root"')
    expect(markup.indexOf('data-testid="tooltip-provider"')).toBeLessThan(
      markup.indexOf('data-testid="app-root"')
    )

    expect(queryClientProviderMock).toHaveBeenCalledOnce()

    const queryClient = queryClientProviderMock.mock.calls[0][0].client as QueryClient
    const queryDefaults = queryClient.getDefaultOptions().queries

    expect(queryDefaults?.retry).toBe(false)
    expect(queryDefaults?.refetchOnWindowFocus).toBe(false)
    expect(queryDefaults?.staleTime).toBe(30_000)
    expect(queryDefaults?.refetchOnMount).toBe(false)
  })
})
