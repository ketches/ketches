import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const {
  mockUseQuery,
  mockUseMutation,
  mockInvalidateQueries,
  actionMutate,
  markAllReadMutate,
} = vi.hoisted(() => ({
  mockUseQuery: vi.fn(),
  mockUseMutation: vi.fn(),
  mockInvalidateQueries: vi.fn(),
  actionMutate: vi.fn(),
  markAllReadMutate: vi.fn(),
}))

vi.mock("@tanstack/react-query", () => ({
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
  useMutation: (...args: unknown[]) => mockUseMutation(...args),
  useQueryClient: () => ({ invalidateQueries: mockInvalidateQueries }),
}))

vi.mock("@/api/notifications", () => ({
  notificationsApi: {
    list: vi.fn(),
    handleAction: vi.fn(),
    markAllRead: vi.fn(),
  },
}))

vi.mock("@/components/ui/dialog", () => ({
  Dialog: ({ open, children }: { open: boolean; children: React.ReactNode }) => open ? <div>{children}</div> : null,
  DialogContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogTitle: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, ...props }: React.ComponentProps<"button">) => <button {...props}>{children}</button>,
}))

vi.mock("../shared/color-badge", () => ({
  ColorBadge: ({ children }: { children: React.ReactNode }) => <span>{children}</span>,
}))

vi.mock("../shared/empty-state", () => ({
  EmptyState: ({ description }: { description: string }) => <div>{description}</div>,
}))

vi.mock("@/components/platform-updates/platform-update-rollout-dialog", () => ({
  PlatformUpdateRolloutDialog: ({ open }: { open: boolean }) => open ? <div data-testid="rollout-dialog">rollout</div> : null,
}))

import { NotificationDialog } from "./notification-dialog"

describe("NotificationDialog", () => {
  beforeEach(() => {
    let mutationCallCount = 0
    mockUseQuery.mockReturnValue({
      data: {
        items: [{
          id: "notif-1",
          sender_id: "",
          sender_name: "",
          category: "info",
          event_type: "platform_update_available",
          title: "Platform update available",
          message: "Recommended version v1.2.0 is available.",
          status: "pending",
          resource_type: "platform",
          resource_id: "platform",
          project_id: "",
          project_name: "",
          action_data: "{\"recommended_version\":\"v1.2.0\"}",
          created_at: new Date().toISOString(),
        }],
        pagination: {
          total: 1,
          page: 1,
          page_size: 10,
          total_pages: 1,
        },
      },
      isLoading: false,
    })
    mockUseMutation.mockImplementation(() => {
      mutationCallCount += 1
      return mutationCallCount % 2 === 1
        ? { mutate: actionMutate, isPending: false }
        : { mutate: markAllReadMutate, isPending: false }
    })
  })

  afterEach(() => {
    document.body.innerHTML = ""
    vi.clearAllMocks()
  })

  it("renders platform update notifications with Update and Ignore actions", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<NotificationDialog open onOpenChange={() => undefined} />)
    })

    expect(container.textContent).toContain("Update")
    expect(container.textContent).toContain("Ignore")
    expect(container.textContent).not.toContain("Read")

    await act(async () => {
      root.unmount()
    })
  })

  it("marks platform update notifications as read when Ignore is clicked", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<NotificationDialog open onOpenChange={() => undefined} />)
    })

    const ignoreButton = Array.from(container.querySelectorAll("button")).find((button) => button.textContent?.includes("Ignore"))
    ignoreButton?.dispatchEvent(new MouseEvent("click", { bubbles: true }))

    expect(actionMutate).toHaveBeenCalledWith({ id: "notif-1", action: "read" })

    await act(async () => {
      root.unmount()
    })
  })

  it("opens the rollout dialog when Update is clicked", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<NotificationDialog open onOpenChange={() => undefined} />)
    })

    const updateButton = Array.from(container.querySelectorAll("button")).find((button) => button.textContent?.includes("Update"))
    await act(async () => {
      updateButton?.dispatchEvent(new MouseEvent("click", { bubbles: true }))
    })

    expect(container.querySelector('[data-testid="rollout-dialog"]')).not.toBeNull()

    await act(async () => {
      root.unmount()
    })
  })
})
