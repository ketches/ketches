import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const { mockUsePlatformUpdateConfigQuery, mockUseUpdatePlatformUpdateConfigMutation } = vi.hoisted(() => ({
  mockUsePlatformUpdateConfigQuery: vi.fn(),
  mockUseUpdatePlatformUpdateConfigMutation: vi.fn(),
}))

vi.mock("@/hooks/use-platform-update", () => ({
  usePlatformUpdateConfigQuery: () => mockUsePlatformUpdateConfigQuery(),
  useUpdatePlatformUpdateConfigMutation: () => mockUseUpdatePlatformUpdateConfigMutation(),
}))

vi.mock("@/components/ui/dialog", () => ({
  Dialog: ({ open, children }: { open: boolean; children: React.ReactNode }) => open ? <div>{children}</div> : null,
  DialogContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogTitle: ({ children }: { children: React.ReactNode }) => <h1>{children}</h1>,
  DialogDescription: ({ children }: { children: React.ReactNode }) => <p>{children}</p>,
  DialogFooter: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, ...props }: React.ComponentProps<"button">) => <button {...props}>{children}</button>,
}))

vi.mock("@/components/ui/field", () => ({
  Field: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  FieldLabel: ({ children, htmlFor }: { children: React.ReactNode; htmlFor?: string }) => <label htmlFor={htmlFor}>{children}</label>,
  FieldContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

import { PlatformUpdateConfigDialog } from "./platform-update-config-dialog"

describe("PlatformUpdateConfigDialog", () => {
  beforeEach(() => {
    mockUseUpdatePlatformUpdateConfigMutation.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
    })
  })

  afterEach(() => {
    document.body.innerHTML = ""
    vi.clearAllMocks()
  })

  it("prefills official default targets in the config dialog", async () => {
    mockUsePlatformUpdateConfigQuery.mockReturnValue({
      data: {
        api: {
          image_repository: "ghcr.io/ketches/ketches/ketches-api",
          namespace: "ketches",
          deployment_name: "ketches-api",
          container_name: "ketches-api",
        },
        ui: {
          image_repository: "ghcr.io/ketches/ketches/ketches-ui",
          namespace: "ketches",
          deployment_name: "ketches-ui",
          container_name: "ketches-ui",
        },
      },
      isLoading: false,
    })

    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(
        <PlatformUpdateConfigDialog open onOpenChange={() => undefined} />
      )
    })

    expect((container.querySelector('input[name="api-image-repository"]') as HTMLInputElement | null)?.value).toBe("ghcr.io/ketches/ketches/ketches-api")
    expect((container.querySelector('input[name="api-namespace"]') as HTMLInputElement | null)?.value).toBe("ketches")
    expect((container.querySelector('input[name="ui-image-repository"]') as HTMLInputElement | null)?.value).toBe("ghcr.io/ketches/ketches/ketches-ui")
    expect((container.querySelector('input[name="ui-deployment-name"]') as HTMLInputElement | null)?.value).toBe("ketches-ui")

    await act(async () => {
      root.unmount()
    })
  })
})
