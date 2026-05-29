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

    expect(container.textContent).toContain("No matching results.")

    await act(async () => {
      root.unmount()
    })
  })

  it("renders a loading skeleton before the first empty-state decision", async () => {
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
          data={[]}
          sourceDataCount={0}
          isLoading
          useStandaloneEmptyState
        />,
      )
    })

    expect(container.innerHTML).not.toBe("")
    expect(container.textContent).not.toContain("Name")
    expect(container.textContent).not.toContain("No matching results.")

    await act(async () => {
      root.unmount()
    })
  })

  it("renders a standalone empty state when source data is empty after loading", async () => {
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
          data={[]}
          sourceDataCount={0}
          useStandaloneEmptyState
          sourceEmptyContent={<div>No source rows</div>}
        />,
      )
    })

    expect(container.textContent).toContain("No source rows")
    expect(container.textContent).not.toContain("Name")

    await act(async () => {
      root.unmount()
    })
  })

  it("keeps the table visible when only the filtered result is empty", async () => {
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
          data={[]}
          sourceDataCount={2}
          useStandaloneEmptyState
        />,
      )
    })

    expect(container.textContent).toContain("Name")
    expect(container.textContent).toContain("No matching results.")

    await act(async () => {
      root.unmount()
    })
  })

  it("uses custom row ids for controlled row selection", async () => {
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
          data={[
            { id: "app-alpha", name: "Alpha" },
            { id: "app-beta", name: "Beta" },
          ]}
          getRowId={(row) => row.id}
          rowSelection={{ "app-beta": true }}
        />,
      )
    })

    const selectedRow = container.querySelector('tr[data-state="selected"]')
    expect(selectedRow?.textContent).toContain("Beta")

    await act(async () => {
      root.unmount()
    })
  })
})
