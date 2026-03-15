import type { Value } from "platejs"

import { COLLAB_EDITOR_EMPTY_VALUE } from "./collab-editor-constants"

const SUPPORTED_BLOCK_TYPES = new Set([
  "p",
  "h1",
  "h2",
  "h3",
  "blockquote",
  "code_block",
  "code_line",
  "a",
])

function createParagraph(text: string): Value[number] {
  return {
    type: "p",
    children: [{ text }],
  }
}

function splitParagraphs(text: string): string[] {
  return text.split(/\r?\n/)
}

function isObjectRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null
}

function isSupportedMarkValue(value: unknown): boolean {
  return value === undefined || typeof value === "boolean" || typeof value === "string"
}

function isSupportedTextNode(node: unknown): node is Record<string, unknown> & { text: string } {
  if (!isObjectRecord(node) || typeof node.text !== "string") {
    return false
  }

  return (
    isSupportedMarkValue(node.bold) &&
    isSupportedMarkValue(node.italic) &&
    isSupportedMarkValue(node.underline) &&
    isSupportedMarkValue(node.strikethrough) &&
    isSupportedMarkValue(node.code) &&
    isSupportedMarkValue(node.color) &&
    isSupportedMarkValue(node.backgroundColor)
  )
}

function hasSupportedListProps(node: Record<string, unknown>): boolean {
  return (
    (node.indent === undefined || typeof node.indent === "number") &&
    (node.listStyleType === undefined || typeof node.listStyleType === "string") &&
    (node.listStart === undefined || typeof node.listStart === "number") &&
    (node.checked === undefined || typeof node.checked === "boolean")
  )
}

function isSupportedInlineNode(node: unknown): boolean {
  if (isSupportedTextNode(node)) {
    return true
  }

  if (!isObjectRecord(node) || node.type !== "a") {
    return false
  }

  return (
    typeof node.url === "string" &&
    Array.isArray(node.children) &&
    node.children.every(isSupportedInlineNode)
  )
}

function isSupportedNode(node: unknown): boolean {
  if (isSupportedTextNode(node)) {
    return true
  }

  if (!isObjectRecord(node) || typeof node.type !== "string") {
    return false
  }

  if (!SUPPORTED_BLOCK_TYPES.has(node.type)) {
    return false
  }

  if (node.type === "a") {
    return (
      typeof node.url === "string" &&
      Array.isArray(node.children) &&
      node.children.every(isSupportedInlineNode)
    )
  }

  if (node.type === "code_block") {
    return Array.isArray(node.children) && node.children.every(isSupportedNode)
  }

  if (node.type === "code_line") {
    return Array.isArray(node.children) && node.children.every(isSupportedTextNode)
  }

  return (
    hasSupportedListProps(node) &&
    Array.isArray(node.children) &&
    node.children.every(isSupportedInlineNode)
  )
}

function isSupportedCollabEditorValue(value: unknown): value is Value {
  return Array.isArray(value) && value.length > 0 && value.every(isSupportedNode)
}

function extractPlainText(node: unknown): string {
  if (typeof node === "string") {
    return node
  }

  if (Array.isArray(node)) {
    return node.map((item) => extractPlainText(item)).join("\n")
  }

  if (!isObjectRecord(node)) {
    return ""
  }

  if (typeof node.text === "string") {
    return node.text
  }

  if (!Array.isArray(node.children)) {
    return ""
  }

  const separator = node.type === "code_block" ? "\n" : ""

  return node.children.map((child) => extractPlainText(child)).join(separator)
}

function toPlainTextParagraphs(input: unknown): Value {
  if (typeof input === "string") {
    const lines = splitParagraphs(input)
    return lines.length > 0 ? lines.map(createParagraph) : COLLAB_EDITOR_EMPTY_VALUE
  }

  if (Array.isArray(input)) {
    const lines = input.flatMap((node) => splitParagraphs(extractPlainText(node)))
    const hasContent = lines.some((line) => line.length > 0)

    return hasContent ? lines.map(createParagraph) : COLLAB_EDITOR_EMPTY_VALUE
  }

  const text = extractPlainText(input)
  return text.length > 0 ? splitParagraphs(text).map(createParagraph) : COLLAB_EDITOR_EMPTY_VALUE
}

export function deserializeCollabEditorValue(input?: string): Value {
  if (!input) {
    return COLLAB_EDITOR_EMPTY_VALUE
  }

  try {
    const parsed: unknown = JSON.parse(input)

    if (isSupportedCollabEditorValue(parsed)) {
      return parsed
    }

    return toPlainTextParagraphs(parsed)
  } catch {
    return toPlainTextParagraphs(input)
  }
}

export function serializeCollabEditorValue(value: Value): string {
  return JSON.stringify(value)
}

export function isCollabEditorEmpty(input?: string): boolean {
  const value = deserializeCollabEditorValue(input)
  const content = value.map((node) => extractPlainText(node)).join("").trim()

  return content.length === 0
}
