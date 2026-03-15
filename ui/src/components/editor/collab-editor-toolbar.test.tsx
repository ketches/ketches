import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, describe, expect, it, vi } from "vitest"
import { createPlateEditor, ParagraphPlugin } from "platejs/react"

;(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true

vi.mock("@/components/ui/fixed-toolbar", () => ({
  FixedToolbar: ({
    children,
    ...props
  }: React.ComponentProps<"div">) => <div {...props}>{children}</div>,
}))

vi.mock("@/components/ui/toolbar", () => ({
  ToolbarButton: ({
    children,
    ...props
  }: React.ComponentProps<"button">) => (
    <button type="button" {...props}>
      {children}
    </button>
  ),
  ToolbarGroup: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenu: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DropdownMenuContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DropdownMenuItem: ({
    children,
    onClick,
  }: {
    children: React.ReactNode
    onClick?: () => void
  }) => (
    <button onClick={onClick} type="button">
      {children}
    </button>
  ),
  DropdownMenuTrigger: ({ render }: { render?: React.ReactNode }) => <>{render ?? null}</>,
}))

import { CollabEditorToolbar } from "./collab-editor-toolbar"

describe("CollabEditorToolbar", () => {
  afterEach(() => {
    document.body.innerHTML = ""
  })

  it("renders against an explicit editor instance without Plate context hooks", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)
    const editor = createPlateEditor({
      plugins: [ParagraphPlugin],
      value: [{ type: "p", children: [{ text: "" }] }],
    })
    const Toolbar = CollabEditorToolbar as unknown as React.ComponentType<{
      editor: typeof editor
    }>

    await act(async () => {
      root.render(<Toolbar editor={editor} />)
    })

    expect(container.textContent).toContain("H1")
    expect(container.textContent).toContain("UL")
    expect(container.textContent).toContain("Todo")
    expect(container.textContent).toContain("Link")

    await act(async () => {
      root.unmount()
    })
  })
})
