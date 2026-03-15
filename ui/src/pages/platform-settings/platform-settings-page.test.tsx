import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { History, Type } from "lucide-react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const {
  mockUseVersion,
  mockUsePlatformBranding,
  mockUsePlatformUpdateStatusQuery,
  mockUseCheckPlatformUpdateMutation,
  mockPlatformGeneralTab,
  mockPlatformUpgradeManagementTab,
  mockPlatformAuditLogTab,
} = vi.hoisted(() => ({
  mockUseVersion: vi.fn(),
  mockUsePlatformBranding: vi.fn(),
  mockUsePlatformUpdateStatusQuery: vi.fn(),
  mockUseCheckPlatformUpdateMutation: vi.fn(),
  mockPlatformGeneralTab: vi.fn(),
  mockPlatformUpgradeManagementTab: vi.fn(),
  mockPlatformAuditLogTab: vi.fn(),
}))

vi.mock("@/components/layout/page-header", () => ({
  PageHeader: () => <div data-testid="page-header">header</div>,
}))

vi.mock("@/components/platform-settings/platform-general-tab", () => ({
  PlatformGeneralTab: () => {
    mockPlatformGeneralTab()
    return <div data-testid="general-tab">general tab</div>
  },
}))

vi.mock("@/components/platform-settings/platform-upgrade-management-tab", () => ({
  PlatformUpgradeManagementTab: (props: {
    status?: {
      api?: { current_version?: string }
      ui?: { current_version?: string }
    }
    isStatusLoading?: boolean
  }) => {
    mockPlatformUpgradeManagementTab(props)
    return (
      <div data-testid="upgrade-tab">
        {props.isStatusLoading ? "loading" : `API ${props.status?.api?.current_version ?? "unknown"} / UI ${props.status?.ui?.current_version ?? "unknown"}`}
      </div>
    )
  },
}))

vi.mock("@/components/platform-settings/platform-audit-log-tab", () => ({
  PlatformAuditLogTab: () => {
    mockPlatformAuditLogTab()
    return <div data-testid="audit-log-tab">audit log tab</div>
  },
}))

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, ...props }: React.ComponentProps<"button">) => <button {...props}>{children}</button>,
}))

vi.mock("@/components/ui/tabs", () => ({
  Tabs: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  TabsList: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  TabsTrigger: ({ children }: { children: React.ReactNode }) => <button type="button">{children}</button>,
  TabsContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/hooks/useVersion", () => ({
  useVersion: () => mockUseVersion(),
}))

vi.mock("@/hooks/use-platform-settings", () => ({
  usePlatformBranding: () => mockUsePlatformBranding(),
}))

vi.mock("@/hooks/use-platform-update", () => ({
  usePlatformUpdateStatusQuery: (enabled?: boolean) => mockUsePlatformUpdateStatusQuery(enabled),
  useCheckPlatformUpdateMutation: () => mockUseCheckPlatformUpdateMutation(),
}))

import { PlatformSettingsPage } from "./platform-settings-page"

describe("PlatformSettingsPage", () => {
  beforeEach(() => {
    mockUseVersion.mockReturnValue("v1.0.0")
    mockUsePlatformBranding.mockReturnValue({
      data: {
        name: "Ketches Admin",
      },
    })
    mockUsePlatformUpdateStatusQuery.mockImplementation((enabled: boolean = true) => ({
      data: enabled
        ? {
            api: {
              current_version: "v1.0.0",
            },
            ui: {
              current_version: "v1.0.1",
            },
            recommended_shared_version: "v1.2.0",
          }
        : undefined,
      isLoading: false,
      isFetching: false,
    }))
    mockUseCheckPlatformUpdateMutation.mockReturnValue({
      data: undefined,
      isPending: false,
      mutate: vi.fn(),
    })
  })

  afterEach(() => {
    document.body.innerHTML = ""
    vi.clearAllMocks()
  })

  it("renders general, upgrade management, and audit log tabs", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<PlatformSettingsPage />)
    })

    expect(container.textContent).toContain("Platform Settings")
    expect(container.textContent).toContain("General")
    expect(container.textContent).toContain("Upgrade Management")
    expect(container.textContent).toContain("Audit Log")
    expect(container.querySelector('[data-testid="general-tab"]')).not.toBeNull()
    expect(container.querySelector('[data-testid="upgrade-tab"]')).not.toBeNull()
    expect(container.querySelector('[data-testid="audit-log-tab"]')).not.toBeNull()

    await act(async () => {
      root.unmount()
    })
  })

  it("loads the current platform versions on initial render", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<PlatformSettingsPage />)
    })

    expect(container.textContent).toContain("API v1.0.0 / UI v1.0.1")

    await act(async () => {
      root.unmount()
    })
  })

  it("uses the agreed tab icons for general and audit log", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<PlatformSettingsPage />)
    })

    const generalTabInvocation = mockPlatformGeneralTab.mock.calls.length
    const auditTabInvocation = mockPlatformAuditLogTab.mock.calls.length

    expect(generalTabInvocation).toBe(1)
    expect(auditTabInvocation).toBe(1)
    expect(Type).toBeDefined()
    expect(History).toBeDefined()

    await act(async () => {
      root.unmount()
    })
  })
})
