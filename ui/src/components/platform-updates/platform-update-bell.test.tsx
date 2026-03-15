import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const {
  mockStatusHook,
  mockRolloutMutation,
  mockAuthState,
  mockRefetch,
} = vi.hoisted(() => ({
  mockStatusHook: vi.fn(),
  mockRolloutMutation: vi.fn(),
  mockAuthState: {
    user: { role: "admin" },
  },
  mockRefetch: vi.fn(),
}))

vi.mock("@/stores/auth", () => ({
  useAuthStore: (selector: (state: typeof mockAuthState) => unknown) => selector(mockAuthState),
}))

vi.mock("@/hooks/use-platform-update", () => ({
  usePlatformUpdateStatusQuery: () => mockStatusHook(),
  useTriggerPlatformRolloutMutation: () => mockRolloutMutation(),
}))

vi.mock("@/components/notifications/notification-bell", () => ({
  NotificationBell: () => <div data-testid="notification-bell">notifications</div>,
}))

vi.mock("@/components/platform-updates/platform-update-config-dialog", () => ({
  PlatformUpdateConfigDialog: ({ open }: { open: boolean }) => open ? <div data-testid="platform-update-config-dialog">config</div> : null,
}))

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, ...props }: React.ComponentProps<"button">) => <button {...props}>{children}</button>,
}))

vi.mock("@/components/ui/input", () => ({
  Input: (props: React.ComponentProps<"input">) => <input {...props} />,
}))

vi.mock("@/components/ui/popover", () => ({
  Popover: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  PopoverTrigger: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  PopoverContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/field", () => ({
  Field: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  FieldLabel: ({ children }: { children: React.ReactNode }) => <label>{children}</label>,
  FieldContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/combobox", () => ({
  Combobox: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  ComboboxInput: (props: React.ComponentProps<"input">) => <input {...props} />,
  ComboboxContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  ComboboxList: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  ComboboxItem: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
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

import { AppHeader } from "@/components/layout/app-header"
import { PlatformUpdateBell } from "./platform-update-bell"

describe("PlatformUpdateBell", () => {
  beforeEach(() => {
    mockAuthState.user = { role: "admin" }
    mockRolloutMutation.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
    })
    mockStatusHook.mockReturnValue({
      data: {
        local_platform_version: "v1.0.0",
        running_in_kubernetes: true,
        can_rollout: true,
        rollout_blockers: [],
        recommended_shared_version: "v1.2.0",
        api: {
          current_version: "v1.0.0",
          latest_version: "v1.2.0",
          update_available: true,
          available_versions: ["v1.2.0", "v1.1.0"],
          rollout_phase: "available",
        },
        ui: {
          current_version: "v1.0.0",
          latest_version: "v1.2.0",
          update_available: true,
          available_versions: ["v1.2.0", "v1.1.0"],
          rollout_phase: "available",
        },
      },
      isLoading: false,
      isFetching: false,
      refetch: mockRefetch,
    })
  })

  afterEach(() => {
    document.body.innerHTML = ""
    vi.clearAllMocks()
  })

  it("renders nothing for non-admin users", async () => {
    mockAuthState.user = { role: "user" }

    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<PlatformUpdateBell />)
    })

    expect(container.textContent).toBe("")

    await act(async () => {
      root.unmount()
    })
  })

  it("shows an update badge and recommended shared version for admins", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<PlatformUpdateBell />)
    })

    expect(container.textContent).toContain("v1.2.0")
    expect(container.textContent).toContain("Update")

    await act(async () => {
      root.unmount()
    })
  })

  it("disables rollout when the backend reports rollout blockers", async () => {
    mockStatusHook.mockReturnValue({
      data: {
        local_platform_version: "v1.0.0",
        running_in_kubernetes: false,
        can_rollout: false,
        rollout_blockers: ["current platform is not running in kubernetes"],
        recommended_shared_version: "v1.2.0",
        api: {
          current_version: "v1.0.0",
          latest_version: "v1.2.0",
          update_available: true,
          available_versions: ["v1.2.0"],
          rollout_phase: "blocked",
        },
        ui: {
          current_version: "v1.0.0",
          latest_version: "v1.2.0",
          update_available: true,
          available_versions: ["v1.2.0"],
          rollout_phase: "blocked",
        },
      },
      isLoading: false,
      isFetching: false,
      refetch: mockRefetch,
    })

    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<PlatformUpdateBell />)
    })

    const rolloutButton = container.querySelector('button[data-role="rollout-submit"]') as HTMLButtonElement | null
    expect(rolloutButton?.disabled).toBe(true)

    await act(async () => {
      root.unmount()
    })
  })

  it("renders beside the existing notification trigger without removing it", async () => {
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

  it("allows API and UI versions to diverge before confirming rollout", async () => {
    const mutate = vi.fn()
    mockRolloutMutation.mockReturnValue({
      mutate,
      isPending: false,
    })

    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<PlatformUpdateBell />)
    })

    const apiInput = container.querySelector('input[name="api-version"]') as HTMLInputElement | null
    const uiInput = container.querySelector('input[name="ui-version"]') as HTMLInputElement | null
    const submitButton = container.querySelector('button[data-role="rollout-submit"]') as HTMLButtonElement | null

    apiInput?.dispatchEvent(new Event("focus", { bubbles: true }))
    if (apiInput) {
      apiInput.value = "v1.2.0"
      apiInput.dispatchEvent(new Event("input", { bubbles: true }))
      apiInput.dispatchEvent(new Event("change", { bubbles: true }))
    }

    uiInput?.dispatchEvent(new Event("focus", { bubbles: true }))
    if (uiInput) {
      uiInput.value = "v1.1.0"
      uiInput.dispatchEvent(new Event("input", { bubbles: true }))
      uiInput.dispatchEvent(new Event("change", { bubbles: true }))
    }

    submitButton?.dispatchEvent(new MouseEvent("click", { bubbles: true }))

    expect(mutate).toHaveBeenCalledWith({
      api_version: "v1.2.0",
      ui_version: "v1.1.0",
    })

    await act(async () => {
      root.unmount()
    })
  })

  it("refreshes platform update status on demand", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<PlatformUpdateBell />)
    })

    const refreshButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent?.includes("Refresh"),
    )
    refreshButton?.dispatchEvent(new MouseEvent("click", { bubbles: true }))

    expect(mockRefetch).toHaveBeenCalledTimes(1)

    await act(async () => {
      root.unmount()
    })
  })

  it("renders without crashing when available platform versions are null", async () => {
    mockStatusHook.mockReturnValue({
      data: {
        local_platform_version: "v1.0.0",
        running_in_kubernetes: false,
        can_rollout: false,
        rollout_blockers: ["failed to list API image tags"],
        recommended_shared_version: "",
        api: {
          current_version: "v1.0.0",
          latest_version: "",
          update_available: false,
          available_versions: null,
          rollout_phase: "blocked",
        },
        ui: {
          current_version: "v1.0.0",
          latest_version: "",
          update_available: false,
          available_versions: null,
          rollout_phase: "blocked",
        },
      },
      isLoading: false,
      isFetching: false,
      refetch: mockRefetch,
    })

    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<PlatformUpdateBell />)
    })

    expect(container.textContent).toContain("API v1.0.0 / UI v1.0.0")

    await act(async () => {
      root.unmount()
    })
  })
})
