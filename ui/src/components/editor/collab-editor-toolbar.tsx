'use client'

import { createPlateEditor } from "platejs/react"
import { ListStyleType, toggleList } from "@platejs/list"
import { toggleCodeBlock } from "@platejs/code-block"
import { unwrapLink, upsertLink } from "@platejs/link"

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { FixedToolbar } from "@/components/ui/fixed-toolbar"
import { ToolbarButton, ToolbarGroup } from "@/components/ui/toolbar"

import {
  COLLAB_BACKGROUND_COLOR_OPTIONS,
  COLLAB_TEXT_COLOR_OPTIONS,
  TODO_LIST_STYLE_TYPE,
} from "./collab-editor-constants"

type CollabToolbarEditor = ReturnType<typeof createPlateEditor>

function BlockToolbarButton({
  editor,
  label,
  nodeType,
}: {
  editor: CollabToolbarEditor
  label: string
  nodeType: "blockquote" | "code_block" | "h1" | "h2" | "h3"
}) {
  const toggleBlock = () => {
    switch (nodeType) {
      case "blockquote":
        editor.tf.toggleBlock("blockquote")
        break
      case "code_block":
        toggleCodeBlock(editor)
        break
      case "h1":
        editor.tf.toggleBlock("h1")
        break
      case "h2":
        editor.tf.toggleBlock("h2")
        break
      case "h3":
        editor.tf.toggleBlock("h3")
        break
    }

    editor.tf.focus()
  }

  return (
    <ToolbarButton
      onClick={toggleBlock}
      onMouseDown={(event) => event.preventDefault()}
    >
      {label}
    </ToolbarButton>
  )
}

function ListToolbarButton({
  editor,
  label,
  nodeType,
}: {
  editor: CollabToolbarEditor
  label: string
  nodeType: string
}) {
  return (
    <ToolbarButton
      onClick={() => {
        toggleList(editor, { listStyleType: nodeType })
        editor.tf.focus()
      }}
      onMouseDown={(event) => event.preventDefault()}
    >
      {label}
    </ToolbarButton>
  )
}

function TodoToolbarButton({ editor }: { editor: CollabToolbarEditor }) {
  return (
    <ToolbarButton
      onClick={() => {
        toggleList(editor, { listStyleType: TODO_LIST_STYLE_TYPE })
        editor.tf.focus()
      }}
      onMouseDown={(event) => event.preventDefault()}
    >
      Todo
    </ToolbarButton>
  )
}

function MarkColorToolbarButton({
  clearKey,
  editor,
  label,
  options,
}: {
  clearKey: "backgroundColor" | "color"
  editor: CollabToolbarEditor
  label: string
  options: ReadonlyArray<{ label: string; value: string }>
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <ToolbarButton onMouseDown={(event) => event.preventDefault()}>
            {label}
          </ToolbarButton>
        }
      />
      <DropdownMenuContent className="w-40">
        {options.map((option) => (
          <DropdownMenuItem
            key={option.value}
            onClick={() => {
              if (option.value === "default") {
                editor.tf.removeMark(clearKey)
                return
              }

              if (clearKey === "color") {
                editor.tf.addMark("color", option.value)
                return
              }

              editor.tf.addMark("backgroundColor", option.value)
            }}
          >
            <span
              className="size-3 rounded-sm border border-border"
              style={{
                backgroundColor:
                  clearKey === "backgroundColor" && option.value !== "default"
                    ? option.value
                    : undefined,
              }}
            />
            <span
              style={{
                color:
                  clearKey === "color" && option.value !== "default"
                    ? option.value
                    : undefined,
              }}
            >
              {option.label}
            </span>
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

function LinkToolbarButton({ editor }: { editor: CollabToolbarEditor }) {
  return (
    <ToolbarButton
      onClick={() => {
        const selectionText = editor.selection ? editor.api.string(editor.selection) : ""
        const nextUrl = window.prompt("Enter link URL", "https://")

        if (!nextUrl) {
          editor.tf.focus()
          return
        }

        upsertLink(editor, {
          text: selectionText || nextUrl,
          url: nextUrl,
        })
        editor.tf.focus()
      }}
      onMouseDown={(event) => event.preventDefault()}
    >
      Link
    </ToolbarButton>
  )
}

function UnlinkToolbarButton({ editor }: { editor: CollabToolbarEditor }) {
  return (
    <ToolbarButton
      onClick={() => {
        unwrapLink(editor)
        editor.tf.focus()
      }}
      onMouseDown={(event) => event.preventDefault()}
    >
      Unlink
    </ToolbarButton>
  )
}

function MarkToolbarButton({
  editor,
  label,
  nodeType,
}: {
  editor: CollabToolbarEditor
  label: string
  nodeType: "bold" | "italic" | "underline" | "strikethrough"
}) {
  return (
    <ToolbarButton
      onClick={() => {
        editor.tf.toggleMark(nodeType)
        editor.tf.focus()
      }}
      onMouseDown={(event) => event.preventDefault()}
    >
      {label}
    </ToolbarButton>
  )
}

export function CollabEditorToolbar({
  editor,
}: {
  editor: CollabToolbarEditor
}) {
  return (
    <FixedToolbar
      className="flex justify-start rounded-none border-x-0 border-t-0"
      data-testid="collab-editor-toolbar"
    >
      <ToolbarGroup>
        <BlockToolbarButton editor={editor} label="H1" nodeType="h1" />
        <BlockToolbarButton editor={editor} label="H2" nodeType="h2" />
        <BlockToolbarButton editor={editor} label="H3" nodeType="h3" />
      </ToolbarGroup>

      <ToolbarGroup>
        <MarkToolbarButton editor={editor} label="B" nodeType="bold" />
        <MarkToolbarButton editor={editor} label="I" nodeType="italic" />
        <MarkToolbarButton editor={editor} label="U" nodeType="underline" />
        <MarkToolbarButton editor={editor} label="S" nodeType="strikethrough" />
      </ToolbarGroup>

      <ToolbarGroup>
        <MarkColorToolbarButton
          clearKey="color"
          editor={editor}
          label="Text"
          options={COLLAB_TEXT_COLOR_OPTIONS}
        />
        <MarkColorToolbarButton
          clearKey="backgroundColor"
          editor={editor}
          label="BG"
          options={COLLAB_BACKGROUND_COLOR_OPTIONS}
        />
      </ToolbarGroup>

      <ToolbarGroup>
        <BlockToolbarButton editor={editor} label="Quote" nodeType="blockquote" />
        <BlockToolbarButton editor={editor} label="Code" nodeType="code_block" />
      </ToolbarGroup>

      <ToolbarGroup>
        <ListToolbarButton editor={editor} label="UL" nodeType={ListStyleType.Disc} />
        <ListToolbarButton editor={editor} label="OL" nodeType={ListStyleType.Decimal} />
        <TodoToolbarButton editor={editor} />
      </ToolbarGroup>

      <ToolbarGroup>
        <LinkToolbarButton editor={editor} />
        <UnlinkToolbarButton editor={editor} />
      </ToolbarGroup>
    </FixedToolbar>
  )
}
