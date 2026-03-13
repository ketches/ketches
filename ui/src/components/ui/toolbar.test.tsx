import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, describe, expect, it } from "vitest"

import { Toolbar, ToolbarButton } from "./toolbar"

afterEach(() => {
  document.body.innerHTML = ""
})

describe("ToolbarButton", () => {
  it("renders tooltip-enabled buttons without nesting button elements", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)

    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(
        <Toolbar>
          <ToolbarButton tooltip="Heading 1" type="button">
            Heading 1
          </ToolbarButton>
        </Toolbar>
      )
    })

    const buttons = container.querySelectorAll("button")

    expect(buttons).toHaveLength(1)
    expect(buttons[0]?.querySelector("button")).toBeNull()

    await act(async () => {
      root.unmount()
    })
  })
})
