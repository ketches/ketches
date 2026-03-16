import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, describe, expect, it, vi } from "vitest"

vi.mock("@/hooks/use-mobile", () => ({
  useIsMobile: () => true,
}))

vi.mock("@/components/ui/dialog", () => ({
  Dialog: ({
    open,
    children,
  }: {
    open?: boolean
    children: React.ReactNode
  }) => (
    <div data-open={open ? "true" : "false"}>
      {open ? children : null}
    </div>
  ),
  DialogContent: ({ children, ...props }: React.ComponentProps<"div">) => (
    <div data-testid="dialog-content" {...props}>{children}</div>
  ),
  DialogHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogTitle: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogDescription: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, ...props }: React.ComponentProps<"button">) => <button type="button" {...props}>{children}</button>,
}))

vi.mock("@/components/ui/input", () => ({
  Input: (props: React.ComponentProps<"input">) => <input {...props} />,
}))

vi.mock("@/components/ui/separator", () => ({
  Separator: (props: React.ComponentProps<"div">) => <div {...props} />,
}))

vi.mock("@/components/ui/skeleton", () => ({
  Skeleton: (props: React.ComponentProps<"div">) => <div {...props} />,
}))

vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  TooltipContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  TooltipTrigger: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

import { Sidebar, SidebarProvider, SidebarTrigger } from "./sidebar"

describe("Sidebar", () => {
  afterEach(() => {
    document.body.innerHTML = ""
  })

  it("opens the mobile sidebar inside dialog content", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(
        <SidebarProvider>
          <Sidebar>
            <div>Mobile Navigation</div>
          </Sidebar>
          <SidebarTrigger />
        </SidebarProvider>
      )
    })

    expect(container.querySelector('[data-testid="dialog-content"]')).toBeNull()

    await act(async () => {
      container.querySelector('button[data-sidebar="trigger"]')?.dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      )
    })

    const dialogContent = container.querySelector('[data-testid="dialog-content"]')
    expect(dialogContent).not.toBeNull()
    expect(dialogContent?.textContent).toContain("Mobile Navigation")

    await act(async () => {
      root.unmount()
    })
  })
})
