import type { ReactNode } from "react"
import { renderToStaticMarkup } from "react-dom/server"
import { afterEach, describe, expect, it, vi } from "vitest"

const { createRootMock, renderMock } = vi.hoisted(() => {
  const renderMock = vi.fn()

  return {
    createRootMock: vi.fn(() => ({ render: renderMock })),
    renderMock,
  }
})

vi.mock("react-dom/client", () => ({
  createRoot: createRootMock,
}))

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
  })
})
