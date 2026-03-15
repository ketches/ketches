import { act } from "react"
import ReactDOMClient from "react-dom/client"
import { afterEach, describe, expect, it } from "vitest"

import { DataTable } from "./data-table"

describe("DataTable", () => {
  afterEach(() => {
    document.body.innerHTML = ""
  })

  it("renders an empty state when data is null", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)

    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(
        <DataTable
          columns={[
            {
              accessorKey: "name",
              header: "Name",
            },
          ]}
          data={null as unknown as Array<{ name: string }>}
        />,
      )
    })

    expect(container.textContent).toContain("No results.")

    await act(async () => {
      root.unmount()
    })
  })
})
