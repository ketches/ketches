import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, describe, expect, it, vi } from "vitest"

vi.mock("./collab-editor", () => ({
  CollabEditor: ({
    placeholder,
    value,
  }: {
    placeholder?: string
    value?: string
  }) => (
    <div data-testid="collab-editor">
      {placeholder}
      {value}
    </div>
  ),
}))

import { BasicEditor } from "./basic-editor"

describe("BasicEditor", () => {
  afterEach(() => {
    document.body.innerHTML = ""
  })

  it("delegates to the collaboration editor implementation", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(<BasicEditor placeholder="Write details" value="Alpha" />)
    })

    expect(container.querySelector("[data-testid='collab-editor']")?.textContent).toContain(
      "Write details"
    )
    expect(container.querySelector("[data-testid='collab-editor']")?.textContent).toContain(
      "Alpha"
    )

    await act(async () => {
      root.unmount()
    })
  })
})
