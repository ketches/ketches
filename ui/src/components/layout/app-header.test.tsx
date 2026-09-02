import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, describe, expect, it, vi } from "vitest"

vi.mock("@/components/notifications/notification-bell", () => ({
  NotificationBell: () => <div data-testid="notification-bell">notifications</div>,
}))

vi.mock("@/components/platform-updates/platform-update-bell", () => ({
  PlatformUpdateBell: () => <div data-testid="platform-update-bell">updates</div>,
}))

vi.mock("@/contexts/use-breadcrumbs", () => ({
  useBreadcrumbs: () => ({ breadcrumbs: [] }),
}))

vi.mock("@/components/ui/sidebar", () => ({
  SidebarTrigger: () => <button type="button">toggle</button>,
}))

vi.mock("@/components/ui/separator", () => ({
  Separator: () => <div />,
}))

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, ...props }: React.ComponentProps<"button">) => <button {...props}>{children}</button>,
}))

import { AppHeader } from "./app-header"

describe("AppHeader", () => {
  afterEach(() => {
    document.body.innerHTML = ""
    vi.clearAllMocks()
  })

  it("renders the notification and platform update bells in the right-side action area", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<AppHeader />)
    })

    expect(container.querySelector('[data-testid="notification-bell"]')).not.toBeNull()
    expect(container.querySelector('[data-testid="platform-update-bell"]')).not.toBeNull()

    await act(async () => {
      root.unmount()
    })
  })
})
