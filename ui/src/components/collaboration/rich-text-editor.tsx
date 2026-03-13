import { BlockquotePlugin, H1Plugin, H2Plugin, H3Plugin } from "@platejs/basic-nodes/react"
import { ImagePlugin } from "@platejs/media/react"
import { Bold, Heading1, Heading2, Heading3, Italic, Quote, Underline } from "lucide-react"
import { Plate, usePlateEditor } from "platejs/react"
import { useMemo } from "react"

import { BasicNodesKit } from "@/components/editor/plugins/basic-nodes-kit"
import { Editor, EditorContainer } from "@/components/ui/editor"
import { FixedToolbar } from "@/components/ui/fixed-toolbar"
import { MarkToolbarButton } from "@/components/ui/mark-toolbar-button"
import { ToolbarButton, ToolbarGroup } from "@/components/ui/toolbar"
import { cn } from "@/lib/utils"
import { parseRichTextValue } from "./rich-text-utils"

interface RichTextEditorProps {
  value?: string
  onChange?: (json: string) => void
  placeholder?: string
  className?: string
}

export function RichTextEditor({
  value,
  onChange,
  placeholder = "Enter content...",
  className,
}: RichTextEditorProps) {
  const initialValue = useMemo(() => parseRichTextValue(value), [value])

  const editor = usePlateEditor(
    {
      plugins: [...BasicNodesKit, ImagePlugin],
      value: initialValue,
    },
    [initialValue]
  )

  return (
    <div className={cn("rounded-md border", className)}>
      <Plate
        editor={editor}
        onValueChange={({ value: currentValue }) => {
          onChange?.(JSON.stringify(currentValue))
        }}
      >
        <FixedToolbar className="justify-start rounded-t-md">
          <ToolbarGroup>
            <ToolbarButton
              type="button"
              onClick={() => editor.getTransforms(H1Plugin)[H1Plugin.key].toggle()}
              tooltip="Heading 1"
            >
              <Heading1 className="size-4" />
            </ToolbarButton>
            <ToolbarButton
              type="button"
              onClick={() => editor.getTransforms(H2Plugin)[H2Plugin.key].toggle()}
              tooltip="Heading 2"
            >
              <Heading2 className="size-4" />
            </ToolbarButton>
            <ToolbarButton
              type="button"
              onClick={() => editor.getTransforms(H3Plugin)[H3Plugin.key].toggle()}
              tooltip="Heading 3"
            >
              <Heading3 className="size-4" />
            </ToolbarButton>
            <ToolbarButton
              type="button"
              onClick={() => editor.getTransforms(BlockquotePlugin).blockquote.toggle()}
              tooltip="Quote"
            >
              <Quote className="size-4" />
            </ToolbarButton>
          </ToolbarGroup>

          <ToolbarGroup>
            <MarkToolbarButton type="button" nodeType="bold" tooltip="Bold">
              <Bold className="size-4" />
            </MarkToolbarButton>
            <MarkToolbarButton type="button" nodeType="italic" tooltip="Italic">
              <Italic className="size-4" />
            </MarkToolbarButton>
            <MarkToolbarButton
              type="button"
              nodeType="underline"
              tooltip="Underline"
            >
              <Underline className="size-4" />
            </MarkToolbarButton>
          </ToolbarGroup>
        </FixedToolbar>

        <EditorContainer variant="select">
          <Editor className="min-h-24" placeholder={placeholder} variant="select" />
        </EditorContainer>
      </Plate>
    </div>
  )
}
