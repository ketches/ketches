import type { Value } from "platejs"

export const TODO_LIST_STYLE_TYPE = "todo"

export const COLLAB_EDITOR_EMPTY_VALUE: Value = [
  {
    type: "p",
    children: [{ text: "" }],
  },
]

export const COLLAB_TEXT_COLOR_OPTIONS = [
  { label: "Default", value: "default" },
  { label: "Slate", value: "#334155" },
  { label: "Ocean", value: "#0f766e" },
  { label: "Rose", value: "#be123c" },
  { label: "Amber", value: "#b45309" },
  { label: "Indigo", value: "#4338ca" },
] as const

export const COLLAB_BACKGROUND_COLOR_OPTIONS = [
  { label: "Default", value: "default" },
  { label: "Sun", value: "#fef08a" },
  { label: "Mint", value: "#bbf7d0" },
  { label: "Sky", value: "#bfdbfe" },
  { label: "Blush", value: "#fecdd3" },
  { label: "Lavender", value: "#ddd6fe" },
] as const
