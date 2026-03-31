import ReactDOMClient from "react-dom/client"
import { act } from "react"
import { afterEach, describe, expect, it, vi } from "vitest"

import { BuilderModelSelector, type BuilderModelOption } from "./builder-model-selector"

const onValueChangeMock = vi.fn()

const options: BuilderModelOption[] = [
  {
    key: "project-claude-sonnet",
    modelLabel: "Claude 4 Sonnet",
    providerLabel: "Anthropic",
    scope: "project",
    providerKey: "anthropic-project",
    modelProfileKey: "claude-sonnet-4",
  },
  {
    key: "user-gpt-4-1",
    modelLabel: "GPT-4.1",
    providerLabel: "OpenAI",
    scope: "user",
    providerKey: "openai-user",
    modelProfileKey: "gpt-4.1",
  },
]

async function renderSelector(
  props: Partial<React.ComponentProps<typeof BuilderModelSelector>> = {}
) {
  const container = document.createElement("div")
  document.body.appendChild(container)
  const root = ReactDOMClient.createRoot(container)

  await act(async () => {
    root.render(
      <BuilderModelSelector
        value={null}
        options={options}
        onValueChange={onValueChangeMock}
        {...props}
      />
    )
  })

  return { container, root }
}

describe("BuilderModelSelector", () => {
  afterEach(() => {
    document.body.innerHTML = ""
    onValueChangeMock.mockReset()
  })

  it("renders the model label and combobox input", async () => {
    const { container, root } = await renderSelector()

    expect(container.textContent).toContain("Model")
    expect(container.querySelector('input[id="builder-model-selector"]')).not.toBeNull()
    expect(container.querySelector('[data-testid="builder-model-selector-groups"]')).toBeNull()

    await act(async () => {
      root.unmount()
    })
  })

  it("renders compact helper and error messaging inline", async () => {
    const { container, root } = await renderSelector({
      value: "project-claude-sonnet",
      compact: true,
      helperText: "Default from project settings",
      errorText: "Model selection is required",
    })

    expect(container.textContent).toContain("Default from project settings")
    expect(container.textContent).toContain("Model selection is required")
    expect(container.querySelector('[data-testid="builder-model-selector-selection"]')?.textContent).toContain(
      "Claude 4 Sonnet · Anthropic"
    )
    expect(container.querySelector('[data-slot="field-error"]')).not.toBeNull()

    await act(async () => {
      root.unmount()
    })
  })
})
