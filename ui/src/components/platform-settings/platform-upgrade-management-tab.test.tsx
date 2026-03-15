import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const {
  mockUsePlatformUpdateConfigQuery,
  mockUseUpdatePlatformUpdateConfigMutation,
} = vi.hoisted(() => ({
  mockUsePlatformUpdateConfigQuery: vi.fn(),
  mockUseUpdatePlatformUpdateConfigMutation: vi.fn(),
}))

vi.mock("@/hooks/use-platform-update", () => ({
  usePlatformUpdateConfigQuery: () => mockUsePlatformUpdateConfigQuery(),
  useUpdatePlatformUpdateConfigMutation: () => mockUseUpdatePlatformUpdateConfigMutation(),
}))

vi.mock("@/components/platform-updates/platform-update-config-dialog", () => ({
  PlatformUpdateConfigDialog: ({ open }: { open: boolean }) =>
    open ? <div data-testid="config-dialog">config dialog</div> : null,
}))

vi.mock("@/components/platform-updates/platform-update-rollout-dialog", () => ({
  PlatformUpdateRolloutDialog: ({ open }: { open: boolean }) =>
    open ? <div data-testid="rollout-dialog">rollout dialog</div> : null,
}))

import { PlatformUpgradeManagementTab } from "./platform-upgrade-management-tab"

describe("PlatformUpgradeManagementTab", () => {
  beforeEach(() => {
    mockUsePlatformUpdateConfigQuery.mockReturnValue({
      data: {
        api: {
          image_repository: "docker.io/ketches/ketches-api",
          namespace: "ketches",
          deployment_name: "ketches-api",
          container_name: "ketches-api",
        },
        ui: {
          image_repository: "docker.io/ketches/ketches-ui",
          namespace: "ketches",
          deployment_name: "ketches-ui",
          container_name: "ketches-ui",
        },
      },
      isLoading: false,
    })
    mockUseUpdatePlatformUpdateConfigMutation.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
    })
  })

  afterEach(() => {
    document.body.innerHTML = ""
    vi.clearAllMocks()
  })

  it("renders the upgrade target, check, and rollout actions inside the management card", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(
        <PlatformUpgradeManagementTab
          status={{
            api: { current_version: "v1.0.0" },
            ui: { current_version: "v1.0.0" },
            recommended_shared_version: "v1.2.0",
            can_rollout: true,
            rollout_blockers: [],
          } as never}
        />
      )
    })

    expect(container.textContent).toContain("Upgrade Target")
    expect(container.textContent).toContain("Check for Updates")
    expect(container.textContent).toContain("Rolling Update")
    expect(container.textContent).not.toContain("Save Targets")

    await act(async () => {
      root.unmount()
    })
  })

  it("opens the upgrade target dialog when Configure is clicked", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<PlatformUpgradeManagementTab />)
    })

    const configureButton = Array.from(container.querySelectorAll("button")).find((button) =>
      button.textContent?.includes("Configure")
    )

    await act(async () => {
      configureButton?.dispatchEvent(new MouseEvent("click", { bubbles: true }))
    })

    expect(container.querySelector('[data-testid="config-dialog"]')).not.toBeNull()

    await act(async () => {
      root.unmount()
    })
  })
})
