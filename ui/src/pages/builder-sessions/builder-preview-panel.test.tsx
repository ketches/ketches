import ReactDOMClient from "react-dom/client"
import { act } from "react"
import { afterEach, describe, expect, it, vi } from "vitest"

import type { BuilderPreviewSummary } from "@/api/builder-sessions"

import { BuilderPreviewPanel } from "./builder-preview-panel"

const downloadMock = vi.fn()
const previewMock = vi.fn()

function buildPreview(overrides: Partial<BuilderPreviewSummary> = {}): BuilderPreviewSummary {
  return {
    available: false,
    status: "unavailable",
    resolved_run_id: "",
    published_at: null,
    completed_at: null,
    output_root: "",
    default_entry_path: "",
    download_available: false,
    preview_available: false,
    is_stale: false,
    newer_run_id: "",
    newer_run_status: "",
    ...overrides,
  }
}

async function renderPanel(preview: BuilderPreviewSummary) {
  const container = document.createElement("div")
  document.body.appendChild(container)
  const root = ReactDOMClient.createRoot(container)

  await act(async () => {
    root.render(
      <BuilderPreviewPanel
        preview={preview}
        onDownload={downloadMock}
        onOpenPreview={previewMock}
      />
    )
  })

  return { container, root }
}

describe("BuilderPreviewPanel", () => {
  afterEach(() => {
    document.body.innerHTML = ""
    downloadMock.mockReset()
    previewMock.mockReset()
  })

  it("renders an unavailable state when no successful preview exists", async () => {
    const { container, root } = await renderPanel(buildPreview())

    expect(container.textContent).toContain("Preview output")
    expect(container.textContent).toContain("No successful preview is available yet")
    expect(container.textContent).not.toContain("Download snapshot")
    expect(container.textContent).not.toContain("Open preview")

    await act(async () => {
      root.unmount()
    })
  })

  it("renders delivery-only state with download but without preview action", async () => {
    const { container, root } = await renderPanel(
      buildPreview({
        available: true,
        status: "delivery_only",
        resolved_run_id: "run-1",
        output_root: "dist",
        download_available: true,
      })
    )

    expect(container.textContent).toContain("Download snapshot")
    expect(container.textContent).toContain("Preview is unavailable for this output")
    expect(container.textContent).not.toContain("Open preview")

    const button = Array.from(container.querySelectorAll("button")).find((node) =>
      node.textContent?.includes("Download snapshot")
    ) as HTMLButtonElement | undefined

    await act(async () => {
      button?.click()
    })

    expect(downloadMock).toHaveBeenCalledTimes(1)

    await act(async () => {
      root.unmount()
    })
  })

  it("renders previewable state with preview and download actions", async () => {
    const { container, root } = await renderPanel(
      buildPreview({
        available: true,
        status: "previewable",
        resolved_run_id: "run-2",
        output_root: "dist",
        default_entry_path: "dist/index.html",
        download_available: true,
        preview_available: true,
      })
    )

    expect(container.textContent).toContain("Open preview")
    expect(container.textContent).toContain("Download snapshot")

    const button = Array.from(container.querySelectorAll("button")).find((node) =>
      node.textContent?.includes("Open preview")
    ) as HTMLButtonElement | undefined

    await act(async () => {
      button?.click()
    })

    expect(previewMock).toHaveBeenCalledTimes(1)

    await act(async () => {
      root.unmount()
    })
  })

  it("renders staleness and provenance details when a newer run exists", async () => {
    const { container, root } = await renderPanel(
      buildPreview({
        available: true,
        status: "previewable",
        resolved_run_id: "run-2",
        output_root: "dist",
        download_available: true,
        preview_available: true,
        is_stale: true,
        newer_run_id: "run-3",
        newer_run_status: "failed",
      })
    )

    expect(container.textContent).toContain("Latest durable preview")
    expect(container.textContent).toContain("run-2")
    expect(container.textContent).toContain("A newer run exists")
    expect(container.textContent).toContain("run-3")
    expect(container.textContent).toContain("failed")

    await act(async () => {
      root.unmount()
    })
  })
})
