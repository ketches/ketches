import "@testing-library/jest-dom/vitest"

import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"

vi.mock("@tanstack/react-query", () => ({
  useMutation: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
  useQueryClient: () => ({
    invalidateQueries: vi.fn(),
  }),
}))

vi.mock("@/components/applications/export-apps-dialog", () => ({
  ExportAppsDialog: () => null,
}))

vi.mock("@/components/ui/alert-dialog", () => ({
  AlertDialog: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogAction: ({ children }: { children: React.ReactNode }) => <button type="button">{children}</button>,
  AlertDialogCancel: ({ children }: { children: React.ReactNode }) => <button type="button">{children}</button>,
  AlertDialogContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogDescription: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogFooter: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogTitle: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  TooltipContent: () => null,
  TooltipTrigger: ({
    render,
    children,
  }: {
    render?: React.ReactNode
    children?: React.ReactNode
  }) => <>{render ?? children ?? null}</>,
}))

vi.mock("sonner", () => ({
  toast: {
    error: vi.fn(),
    success: vi.fn(),
  },
}))

import { AppActionIcons } from "./app-action-icons"

describe("AppActionIcons", () => {
  afterEach(() => {
    document.body.innerHTML = ""
  })

  it("opens the more actions dropdown from the trigger button", async () => {
    const user = userEvent.setup()
    const onInteractionChange = vi.fn()

    const { container } = render(
      <AppActionIcons
        appId="app-1"
        envId="env-1"
        actions={[
          {
            action: "restart",
            label: "Restart",
            icon: "rotate-cw",
            category: "secondary",
            variant: "default",
          },
        ]}
        onInteractionChange={onInteractionChange}
      />,
    )

    const trigger = container.querySelector<HTMLButtonElement>('[data-slot="dropdown-menu-trigger"]')

    expect(trigger).toBeInTheDocument()
    await user.click(trigger!)

    expect(await screen.findByText("Restart")).toBeInTheDocument()
    expect(screen.getByText("Export")).toBeInTheDocument()
    await waitFor(() => expect(onInteractionChange).toHaveBeenLastCalledWith(true))

    await user.keyboard("{Escape}")

    await waitFor(() => expect(onInteractionChange).toHaveBeenLastCalledWith(false))
  })
})
