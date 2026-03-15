import type { AutoformatRule } from "@platejs/autoformat"

import { AutoformatPlugin } from "@platejs/autoformat"
import { toggleCodeBlock } from "@platejs/code-block"
import { ListStyleType, toggleList } from "@platejs/list"

import { TODO_LIST_STYLE_TYPE } from "@/components/editor/collab-editor-constants"

const collabAutoformatRules: AutoformatRule[] = [
  {
    match: "#",
    mode: "block",
    type: "h1",
  },
  {
    match: "##",
    mode: "block",
    type: "h2",
  },
  {
    match: "###",
    mode: "block",
    type: "h3",
  },
  {
    match: ">",
    mode: "block",
    type: "blockquote",
  },
  {
    format: (editor) => {
      toggleList(editor, { listStyleType: ListStyleType.Disc })
    },
    match: ["-", "*"],
    mode: "block",
  },
  {
    format: (editor) => {
      toggleList(editor, { listStyleType: ListStyleType.Decimal })
    },
    match: ["1.", "1)"],
    mode: "block",
  },
  {
    format: (editor) => {
      toggleList(editor, { listStyleType: TODO_LIST_STYLE_TYPE })
    },
    match: ["[]", "[ ]"],
    mode: "block",
  },
  {
    format: (editor) => {
      toggleCodeBlock(editor)
    },
    match: "```",
    mode: "block",
  },
]

export const CollabAutoformatKit = AutoformatPlugin.configure({
  options: {
    rules: collabAutoformatRules,
  },
})
