import { describe, expect, it } from "vitest"

import {
  deserializeCollabEditorValue,
  isCollabEditorEmpty,
  serializeCollabEditorValue,
} from "./collab-editor-value"

describe("collab editor value helpers", () => {
  it("returns an empty paragraph for empty input", () => {
    expect(deserializeCollabEditorValue()).toEqual([
      { type: "p", children: [{ text: "" }] },
    ])
  })

  it("converts plain text input into paragraphs", () => {
    expect(deserializeCollabEditorValue("Alpha\nBeta")).toEqual([
      { type: "p", children: [{ text: "Alpha" }] },
      { type: "p", children: [{ text: "Beta" }] },
    ])
  })

  it("round-trips supported collaboration editor json", () => {
    const value = [
      {
        type: "h2",
        children: [{ text: "Heading" }],
      },
      {
        type: "p",
        children: [
          { text: "A " },
          { bold: true, text: "bold" },
          { italic: true, text: " and italic" },
          { underline: true, text: " line" },
          { strikethrough: true, text: " strike" },
        ],
      },
      {
        type: "code_block",
        children: [
          {
            type: "code_line",
            children: [{ text: "const value = 1" }],
          },
        ],
      },
      {
        type: "p",
        children: [
          {
            type: "a",
            url: "https://example.com",
            children: [{ text: "Example" }],
          },
        ],
      },
      {
        type: "p",
        children: [{ color: "#0f766e", text: "Tint" }],
      },
      {
        type: "p",
        children: [{ backgroundColor: "#fef08a", text: "Highlight" }],
      },
      {
        type: "p",
        children: [{ text: "Todo item" }],
        indent: 1,
        listStyleType: "disc",
      },
    ]

    const serialized = serializeCollabEditorValue(value)

    expect(deserializeCollabEditorValue(serialized)).toEqual(value)
  })

  it("downgrades unsupported legacy json to plain text paragraphs", () => {
    const legacyValue = JSON.stringify([
      {
        type: "legacy-card",
        children: [{ text: "Legacy title" }],
      },
      {
        type: "image",
        children: [{ text: "Screenshot caption" }],
      },
      {
        type: "p",
        children: [{ text: "Trailing note" }],
      },
    ])

    expect(deserializeCollabEditorValue(legacyValue)).toEqual([
      { type: "p", children: [{ text: "Legacy title" }] },
      { type: "p", children: [{ text: "Screenshot caption" }] },
      { type: "p", children: [{ text: "Trailing note" }] },
    ])
  })

  it("treats blank content as empty across supported and downgraded shapes", () => {
    expect(isCollabEditorEmpty("")).toBe(true)
    expect(
      isCollabEditorEmpty(JSON.stringify([{ type: "p", children: [{ text: "   " }] }]))
    ).toBe(true)
    expect(
      isCollabEditorEmpty(
        JSON.stringify([
          {
            type: "code_block",
            children: [{ type: "code_line", children: [{ text: "   " }] }],
          },
        ])
      )
    ).toBe(true)
    expect(
      isCollabEditorEmpty(
        JSON.stringify([{ type: "legacy-card", children: [{ text: "text" }] }])
      )
    ).toBe(false)
  })
})
