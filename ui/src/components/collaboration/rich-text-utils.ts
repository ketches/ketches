import { type Value } from "platejs"

const EMPTY_EDITOR_VALUE: Value = [{ type: "p", children: [{ text: "" }] }]

const isPlateValue = (value: unknown): value is Value => Array.isArray(value)

const toPlainTextValue = (text: string): Value => [{ type: "p", children: [{ text }] }]

export const parseRichTextValue = (value?: string): Value => {
  if (!value) {
    return EMPTY_EDITOR_VALUE
  }

  try {
    const parsed: unknown = JSON.parse(value)
    if (isPlateValue(parsed)) {
      return parsed.length > 0 ? parsed : EMPTY_EDITOR_VALUE
    }

    if (typeof parsed === "string") {
      return toPlainTextValue(parsed)
    }

    return EMPTY_EDITOR_VALUE
  } catch {
    return toPlainTextValue(value)
  }
}

const extractPlainText = (node: unknown): string => {
  if (typeof node === "string") {
    return node
  }

  if (!node || typeof node !== "object") {
    return ""
  }

  const record = node as Record<string, unknown>

  if (typeof record.text === "string") {
    return record.text
  }

  if (!Array.isArray(record.children)) {
    return ""
  }

  return record.children.map((child) => extractPlainText(child)).join("")
}

export const isRichTextEmpty = (value?: string): boolean => {
  const parsed = parseRichTextValue(value)
  const content = parsed.map((node) => extractPlainText(node)).join("").trim()
  return content.length === 0
}
