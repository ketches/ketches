import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const { mockUseVersion, mockUsePlatformBranding } = vi.hoisted(() => ({
  mockUseVersion: vi.fn(),
  mockUsePlatformBranding: vi.fn(),
}))

vi.mock("@/hooks/useVersion", () => ({
  useVersion: () => mockUseVersion(),
}))

vi.mock("@/hooks/use-platform-settings", () => ({
  usePlatformBranding: () => mockUsePlatformBranding(),
}))

vi.mock("@/components/ui/sidebar", () => ({
  SidebarMenu: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  SidebarMenuItem: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  SidebarMenuButton: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

import { PlatformHeader } from "./platform-header"

describe("PlatformHeader", () => {
  beforeEach(() => {
    mockUseVersion.mockReturnValue("v1.0.0")
    mockUsePlatformBranding.mockReturnValue({
      data: {
        name: "Acme Control Plane",
      },
    })
  })

  afterEach(() => {
    document.body.innerHTML = ""
    vi.clearAllMocks()
  })

  it("renders the configured branding instead of the built-in sidebar title", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<PlatformHeader />)
    })

    expect(container.textContent).toContain("Acme Control Plane")
    expect(container.textContent).not.toContain("Ketches Admin")
    expect((container.querySelector("img") as HTMLImageElement | null)?.getAttribute("src")).toBe("/ketches.svg")

    await act(async () => {
      root.unmount()
    })
  })

  it("does not render the built-in branding while the custom branding is still loading", async () => {
    mockUsePlatformBranding.mockReturnValue({
      data: undefined,
      isLoading: true,
    })

    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<PlatformHeader />)
    })

    expect(container.textContent).not.toContain("Ketches Admin")
    expect(container.querySelector("img")).toBeNull()

    await act(async () => {
      root.unmount()
    })
  })
})
