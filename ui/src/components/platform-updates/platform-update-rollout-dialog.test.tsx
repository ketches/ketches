import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const {
  mockStatusHook,
  mockRolloutMutation,
  mockRefetch,
} = vi.hoisted(() => ({
  mockStatusHook: vi.fn(),
  mockRolloutMutation: vi.fn(),
  mockRefetch: vi.fn(),
}))

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

vi.mock("@/hooks/use-platform-update", () => ({
  usePlatformUpdateStatusQuery: (enabled?: boolean) => mockStatusHook(enabled),
  useTriggerPlatformRolloutMutation: () => mockRolloutMutation(),
}))

vi.mock("@/lib/build-info", () => ({
  getAppBuildTime: () => "2026-03-14T00:00:00Z",
  getAppBuildVersion: () => "v1.0.0",
}))

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, ...props }: React.ComponentProps<"button">) => <button {...props}>{children}</button>,
}))

vi.mock("@/components/ui/dialog", () => ({
  Dialog: ({ open, children }: { open: boolean; children: React.ReactNode }) => open ? <div>{children}</div> : null,
  DialogContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogDescription: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogFooter: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogTitle: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
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

import { PlatformUpdateRolloutDialog } from "./platform-update-rollout-dialog"

describe("PlatformUpdateRolloutDialog", () => {
  beforeEach(() => {
    mockRolloutMutation.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
    })
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
      isFetching: false,
      refetch: mockRefetch,
    })
  })

  afterEach(() => {
    document.body.innerHTML = ""
    vi.clearAllMocks()
  })

  it("renders without crashing when available platform versions are null", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<PlatformUpdateRolloutDialog open onOpenChange={() => undefined} />)
    })

    expect(container.textContent).toContain("Platform Updates")
    expect(container.textContent).toContain("API v1.0.0 / UI v1.0.0")

    await act(async () => {
      root.unmount()
    })
  })
})
