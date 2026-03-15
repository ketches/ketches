import { act, useState } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, describe, expect, it, vi } from "vitest"

;(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true

const plateState = vi.hoisted(() => {
  let editorCount = 0
  let lastDeps: unknown[] | undefined
  let lastEditor:
    | {
        api: {
          isText: (node: unknown) => boolean
          some: () => boolean
          string: () => string
        }
        id: string
        selection: null
        tf: {
          backgroundColor: { addMark: (value: string) => void }
          blockquote: { toggle: () => void }
          code_block: { toggle: () => void }
          color: { addMark: (value: string) => void }
          focus: () => void
          h1: { toggle: () => void }
          h2: { toggle: () => void }
          h3: { toggle: () => void }
          removeMark: (key: string) => void
          setValue: (value: unknown) => void
        }
      }
    | undefined
  let lastOnValueChange:
    | ((args: { value: Array<{ children: Array<{ text: string }>; type: string }> }) => void)
    | undefined

  const areDepsEqual = (nextDeps?: unknown[]) => {
    if (!lastDeps && !nextDeps) {
      return true
    }

    if (!lastDeps || !nextDeps || lastDeps.length !== nextDeps.length) {
      return false
    }

    return lastDeps.every((dep, index) => Object.is(dep, nextDeps[index]))
  }

  return {
    reset() {
      editorCount = 0
      lastDeps = undefined
      lastEditor = undefined
      lastOnValueChange = undefined
    },
    triggerValueChange(text: string) {
      lastOnValueChange?.({
        value: [{ type: "p", children: [{ text }] }],
      })
    },
    usePlateEditorMock: vi.fn((_: unknown, deps?: unknown[]) => {
      if (!lastEditor || !areDepsEqual(deps)) {
        editorCount += 1
        lastEditor = {
          api: {
            isText: (node) =>
              typeof node === "object" && node !== null && "text" in (node as object),
            some: () => false,
            string: () => "",
          },
          id: `editor-${editorCount}`,
          selection: null,
          tf: {
            backgroundColor: { addMark: () => undefined },
            blockquote: { toggle: () => undefined },
            code_block: { toggle: () => undefined },
            color: { addMark: () => undefined },
            focus: () => undefined,
            h1: { toggle: () => undefined },
            h2: { toggle: () => undefined },
            h3: { toggle: () => undefined },
            removeMark: () => undefined,
            setValue: () => undefined,
          },
        }
        lastDeps = deps ? [...deps] : deps
      }

      return lastEditor
    }),
    setOnValueChange(
      handler?: (args: {
        value: Array<{ children: Array<{ text: string }>; type: string }>
      }) => void
    ) {
      lastOnValueChange = handler
    },
  }
})

vi.mock("platejs/react", async (importOriginal) => {
  const actual = await importOriginal<typeof import("platejs/react")>()

  return {
    ...actual,
    Plate: ({
      children,
      editor,
      onValueChange,
    }: {
      children: React.ReactNode
      editor: { id: string }
      onValueChange?: (args: {
        value: Array<{ children: Array<{ text: string }>; type: string }>
      }) => void
    }) => {
      plateState.setOnValueChange(onValueChange)

      return (
        <div data-testid="plate-root">
          <div data-editor-id={editor.id} key={editor.id}>
            {children}
          </div>
        </div>
      )
    },
    useEditorRef: () => plateState.usePlateEditorMock.mock.results.at(-1)?.value,
    useEditorSelector: (selector: (editor: unknown) => boolean) =>
      selector(plateState.usePlateEditorMock.mock.results.at(-1)?.value),
    usePlateEditor: plateState.usePlateEditorMock,
  }
})

vi.mock("@/components/ui/editor", () => ({
  EditorContainer: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="editor-container">{children}</div>
  ),
  Editor: ({
    className,
    placeholder,
  }: {
    className?: string
    placeholder?: string
  }) => (
    <div
      aria-label={placeholder}
      className={className}
      contentEditable
      data-testid="editor"
      suppressContentEditableWarning
      onInput={(event) => {
        plateState.triggerValueChange(event.currentTarget.textContent ?? "")
      }}
    />
  ),
}))

vi.mock("@/components/ui/fixed-toolbar", () => ({
  FixedToolbar: ({
    children,
    ...props
  }: React.ComponentProps<"div">) => <div {...props}>{children}</div>,
}))

vi.mock("@/components/ui/toolbar", () => ({
  ToolbarButton: ({
    children,
    pressed: _pressed,
    ...props
  }: React.ComponentProps<"button"> & { pressed?: boolean }) => (
    <button type="button" {...props}>
      {children}
    </button>
  ),
  ToolbarGroup: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock("@/components/ui/mark-toolbar-button", () => ({
  MarkToolbarButton: ({
    children,
  }: {
    children: React.ReactNode
    nodeType: string
  }) => <button type="button">{children}</button>,
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

vi.mock("./collab-editor-toolbar", () => ({
  CollabEditorToolbar: () => (
    <div data-testid="collab-editor-toolbar">H1 B Code Link</div>
  ),
}))

import { CollabEditor } from "./collab-editor"

function ControlledEditorHarness({ initialValue = "" }: { initialValue?: string }) {
  const [value, setValue] = useState(initialValue)

  return (
    <div>
      <CollabEditor
        onChange={setValue}
        placeholder="Type your amazing content here..."
        value={value}
      />
      <output data-testid="serialized-value">{value}</output>
    </div>
  )
}

async function inputEditorText(editor: HTMLDivElement, text: string) {
  await act(async () => {
    editor.textContent = text
    editor.dispatchEvent(new Event("input", { bubbles: true }))
  })
}

describe("CollabEditor", () => {
  afterEach(() => {
    document.body.innerHTML = ""
    plateState.reset()
  })

  it("renders a fixed toolbar with the expected formatting groups", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<CollabEditor />)
    })

    const toolbar = container.querySelector("[data-testid='collab-editor-toolbar']")

    expect(toolbar).not.toBeNull()
    expect(toolbar?.textContent).toContain("H1")
    expect(toolbar?.textContent).toContain("B")
    expect(toolbar?.textContent).toContain("Code")
    expect(toolbar?.textContent).toContain("Link")

    await act(async () => {
      root.unmount()
    })
  })

  it("keeps the same editor instance while parent state echoes typed content", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<ControlledEditorHarness />)
    })

    const firstEditorId = container
      .querySelector("[data-editor-id]")
      ?.getAttribute("data-editor-id")
    const editor = container.querySelector("[data-testid='editor']") as HTMLDivElement

    await inputEditorText(editor, "A")
    await inputEditorText(editor, "AB")

    const secondEditorId = container
      .querySelector("[data-editor-id]")
      ?.getAttribute("data-editor-id")

    expect(firstEditorId).toBe(secondEditorId)
    expect(container.querySelector("[data-testid='serialized-value']")?.textContent).toContain(
      "AB"
    )

    await act(async () => {
      root.unmount()
    })
  })

  it("does not lose focus after each parent-driven update during typing", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<ControlledEditorHarness />)
    })

    const editor = container.querySelector("[data-testid='editor']") as HTMLDivElement

    await act(async () => {
      editor.focus()
    })

    expect(document.activeElement).toBe(editor)

    await inputEditorText(editor, "A")

    const editorAfterFirstInput = container.querySelector("[data-testid='editor']")
    expect(document.activeElement).toBe(editorAfterFirstInput)

    await inputEditorText(editorAfterFirstInput as HTMLDivElement, "AB")

    const editorAfterSecondInput = container.querySelector("[data-testid='editor']")
    expect(document.activeElement).toBe(editorAfterSecondInput)

    await act(async () => {
      root.unmount()
    })
  })
})
