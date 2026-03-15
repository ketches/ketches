import { describe, expect, it } from "vitest"

import {
  deserializeBasicEditorValue,
  isBasicEditorEmpty,
  serializeBasicEditorValue,
} from "./basic-editor-value"

describe("basic editor value helpers", () => {
  it("keeps the legacy helper contract through the collaboration adapter", () => {
    const value = deserializeBasicEditorValue("Alpha")

    expect(value).toEqual([{ type: "p", children: [{ text: "Alpha" }] }])
    expect(serializeBasicEditorValue(value)).toBe(
      JSON.stringify([{ type: "p", children: [{ text: "Alpha" }] }])
    )
    expect(isBasicEditorEmpty("")).toBe(true)
    expect(isBasicEditorEmpty("Alpha")).toBe(false)
  })
})
