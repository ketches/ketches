import ReactDOMClient from "react-dom/client"
import { act } from "react"
import { afterEach, describe, expect, it, vi } from "vitest"

import type { BuilderModelOption } from "./builder-model-selector"

import { BuilderComposer } from "./builder-composer"

const options: BuilderModelOption[] = [
  {
    key: "project-claude-sonnet",
    modelLabel: "Claude 4 Sonnet",
    providerLabel: "Anthropic",
    scope: "project",
    providerKey: "anthropic-project",
    modelProfileKey: "claude-sonnet-4",
  },
]

describe("BuilderComposer", () => {
  afterEach(() => {
    document.body.innerHTML = ""
  })

  it("renders a compact chat-style composer with footer model controls and send action", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(
        <BuilderComposer
          value=""
          onValueChange={vi.fn()}
          onSubmit={vi.fn()}
          isSubmitting={false}
          modelValue={"project-claude-sonnet"}
          modelOptions={options}
          onModelValueChange={vi.fn()}
          modelSelectionHint="Default from project settings"
        />
      )
    })

    const shell = container.querySelector('[data-testid="builder-composer-shell"]')

    expect(shell).not.toBeNull()
    expect(shell?.className).toContain("shrink-0")
    expect(container.querySelector('[data-testid="builder-composer"]')).not.toBeNull()
    expect(container.querySelector('[data-testid="builder-composer-footer"]')).not.toBeNull()
    expect(container.querySelector('[data-testid="builder-model-selector-compact"]')).not.toBeNull()
    expect(container.querySelector('[data-testid="builder-model-selector-selection"]')?.textContent).toContain(
      "Claude 4 Sonnet · Anthropic"
    )
    expect(container.querySelector('[data-testid="builder-send-message"]')).not.toBeNull()
    expect(container.textContent).not.toContain("Model")
    expect(container.textContent).not.toContain("Default from project settings")

    await act(async () => {
      root.unmount()
    })
  })

  it("renders model errors in the compact footer controls", async () => {
    const container = document.createElement("div")
    document.body.appendChild(container)
    const root = ReactDOMClient.createRoot(container)

    await act(async () => {
      root.render(
        <BuilderComposer
          value=""
          onValueChange={vi.fn()}
          onSubmit={vi.fn()}
          isSubmitting={false}
          modelValue={null}
          modelOptions={options}
          onModelValueChange={vi.fn()}
          modelError="Select a model before sending"
        />
      )
    })

    expect(container.querySelector('[data-testid="builder-model-selector-compact"]')).not.toBeNull()
    expect(container.querySelector('[data-slot="field-error"]')).not.toBeNull()
    expect(container.textContent).toContain("Select a model before sending")

    await act(async () => {
      root.unmount()
    })
  })
})
