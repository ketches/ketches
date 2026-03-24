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

async function renderSelector(value: string | null = null) {
  const container = document.createElement("div")
  document.body.appendChild(container)
  const root = ReactDOMClient.createRoot(container)

  await act(async () => {
    root.render(
      <BuilderModelSelector
        value={value}
        options={options}
        onValueChange={onValueChangeMock}
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

  it("renders grouped model options with muted provider labels", async () => {
    const { container, root } = await renderSelector()

    expect(container.textContent).toContain("Model")
    expect(container.textContent).toContain("Project models")
    expect(container.textContent).toContain("My models")
    expect(container.textContent).toContain("Claude 4 Sonnet")
    expect(container.textContent).toContain("Anthropic · Project")
    expect(container.textContent).toContain("GPT-4.1")
    expect(container.textContent).toContain("OpenAI · User")

    await act(async () => {
      root.unmount()
    })
  })
})
