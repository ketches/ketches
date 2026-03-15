import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const { mockUseQuery } = vi.hoisted(() => ({
  mockUseQuery: vi.fn(),
}))

vi.mock("@tanstack/react-query", async () => {
  const actual = await vi.importActual<typeof import("@tanstack/react-query")>("@tanstack/react-query")
  return {
    ...actual,
    useQuery: (...args: unknown[]) => mockUseQuery(...args),
  }
})

vi.mock("@/components/platform-settings/operation-log-retention-dialog", () => ({
  OperationLogRetentionDialog: ({
    open,
    retentionDays,
  }: {
    open: boolean
    retentionDays: number
  }) => open ? <div data-testid="retention-dialog">dialog:{retentionDays}</div> : null,
}))

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, ...props }: React.ComponentProps<"button">) => <button {...props}>{children}</button>,
}))

import { PlatformOperationLogRetentionCard } from "./platform-operation-log-retention-card"

describe("PlatformOperationLogRetentionCard", () => {
  beforeEach(() => {
    mockUseQuery.mockReturnValue({
      data: {
        retention_days: 90,
      },
      isLoading: false,
    })
  })

  afterEach(() => {
    document.body.innerHTML = ""
    vi.clearAllMocks()
  })

  it("shows the current retention days and opens the edit dialog", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<PlatformOperationLogRetentionCard />)
    })

    expect(container.textContent).toContain("Operation Log Retention")
    expect(container.textContent).toContain("90 days")

    const editButton = container.querySelector('button[aria-label="Edit retention days"]') as HTMLButtonElement | null
    expect(editButton).not.toBeNull()

    await act(async () => {
      editButton?.click()
    })

    expect(container.querySelector('[data-testid="retention-dialog"]')?.textContent).toContain("dialog:90")

    await act(async () => {
      root.unmount()
    })
  })
})
