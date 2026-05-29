import { act } from "react"
import ReactDOMClient from "react-dom/client"
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

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, type, ...props }: React.ComponentProps<"button">) => <button type={type ?? "button"} {...props}>{children}</button>,
}))

vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenu: ({
    children,
    onOpenChange,
  }: {
    children: React.ReactNode
    onOpenChange?: (open: boolean) => void
  }) => (
    <div>
      <button type="button" data-testid="open-actions" onClick={() => onOpenChange?.(true)}>
        Open actions
      </button>
      <button type="button" data-testid="close-actions" onClick={() => onOpenChange?.(false)}>
        Close actions
      </button>
      {children}
    </div>
  ),
  DropdownMenuContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DropdownMenuItem: ({ children, ...props }: React.ComponentProps<"button">) => <button type="button" {...props}>{children}</button>,
  DropdownMenuSeparator: () => <div data-testid="menu-separator" />,
  DropdownMenuSub: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DropdownMenuSubContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DropdownMenuSubTrigger: ({ children }: { children: React.ReactNode }) => <button type="button">{children}</button>,
  DropdownMenuTrigger: ({ children, render }: { children?: React.ReactNode; render?: React.ReactNode }) => <>{render ?? children ?? null}</>,
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

  it("reports active interaction while the actions dropdown is open", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)
    const onInteractionChange = vi.fn()

    await act(async () => {
      root.render(
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
    })

    await act(async () => {
      container.querySelector<HTMLButtonElement>('[data-testid="open-actions"]')?.click()
    })

    expect(onInteractionChange).toHaveBeenLastCalledWith(true)

    await act(async () => {
      container.querySelector<HTMLButtonElement>('[data-testid="close-actions"]')?.click()
    })

    expect(onInteractionChange).toHaveBeenLastCalledWith(false)

    await act(async () => {
      root.unmount()
    })
  })
})
